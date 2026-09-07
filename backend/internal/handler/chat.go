// Handler HTTP do chat (RAG). Mantido separado do handler de auth/knowledge
// para isolar responsabilidades.
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/bsmart/abis/internal/chat"
	"github.com/bsmart/abis/internal/repository"
	"github.com/bsmart/abis/internal/service"
	"github.com/google/uuid"
)

// ChatHandler expõe o endpoint /api/chat.
type ChatHandler struct {
	svc         *chat.Service
	workflow    *service.WorkflowService
	employeeRepo *repository.EmployeeRepo
	chatRepo    *repository.ChatRepo
}

func NewChatHandler(s *chat.Service, workflow *service.WorkflowService, empRepo *repository.EmployeeRepo, chatRepo *repository.ChatRepo) *ChatHandler {
	return &ChatHandler{svc: s, workflow: workflow, employeeRepo: empRepo, chatRepo: chatRepo}
}

// ChatRequest é o corpo aceito por POST /api/chat.
type ChatRequest struct {
	Question       string `json:"question"`
	AllowWebSearch bool   `json:"allowWebSearch"`
}

// SourceDTO é o formato de uma fonte devolvida ao front-end.
// Named type para evitar repetir o struct anônimo em vários lugares.
type SourceDTO struct {
	DocumentID string  `json:"documentId"`
	Title      string  `json:"title"`
	Source     string  `json:"source"`
	ChunkOrd   int     `json:"chunkOrd"`
	Content    string  `json:"content"`
	Snippet    string  `json:"snippet"`
	Score      float64 `json:"score"`
}

// ChatResponse é o JSON devolvido. Mantém compatibilidade com o que
// o front-end já espera: { answer, sources: [...] }.
type ChatResponse struct {
	Answer          string      `json:"answer"`
	Sources         []SourceDTO `json:"sources"`
	DocumentRunID   string      `json:"documentRunId,omitempty"`
	DocumentDocxURL string      `json:"documentDocxUrl,omitempty"`
	DocumentPdfURL  string      `json:"documentPdfUrl,omitempty"`
}

// GenerateDocumentRequest é o corpo para geração de documento via chat.
type GenerateDocumentRequest struct {
	Question string `json:"question"`
}

// GenerateDocumentResponse é a resposta da geração de documento.
type GenerateDocumentResponse struct {
	DocumentRunID string `json:"documentRunId"`
	DocxURL       string `json:"docxUrl"`
	PdfURL        string `json:"pdfUrl"`
	Message       string `json:"message"`
}

// Chat (POST /api/chat) recebe a pergunta, executa o pipeline RAG
// (busca FTS5 + prompt + Groq) e devolve a resposta.
//
// Se a pergunta for um pedido de documento, inicia o workflow completo
// (classificação → normativos → requisitos → template compatível → geração).
// O chat NUNCA escolhe templates arbitrariamente nem inventa valores default.
//
// Protegido pelo middleware de auth (rota exige JWT válido).
func (h *ChatHandler) Chat(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[CHAT] Chat handler iniciado")
	var body ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		fmt.Printf("[CHAT] Erro decode JSON: %v\n", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	fmt.Printf("[CHAT] Pergunta recebida: %s allowWebSearch=%v\n", body.Question, body.AllowWebSearch)

	employeeID, _ := r.Context().Value("employee_id").(string)

	if h.chatRepo != nil && employeeID != "" {
		msgID := "msg-" + uuid.NewString()
		now := time.Now().UTC().Format("2006-01-02 15:04:05")
		err := h.chatRepo.InsertMessage(r.Context(), repository.ChatMessage{
			ID:          msgID,
			EmployeeID:  employeeID,
			Role:        "user",
			Content:     body.Question,
			AllowWebSearch: body.AllowWebSearch,
			CreatedAt:   now,
		})
		if err != nil {
			fmt.Printf("[CHAT] Erro salvando mensagem do usuario: %v\n", err)
		}
	}

	resp, err := h.svc.Ask(r.Context(), chat.ChatRequest{Question: body.Question, AllowWebSearch: body.AllowWebSearch})
	if err != nil {
		fmt.Printf("[CHAT] Erro no service Ask: %v\n", err)
		// Erros de validação são 400; erros de LLM são 502 (upstream problem).
		status := http.StatusBadGateway
		if err == chat.ErrEmptyQuestion {
			status = http.StatusBadRequest
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	fmt.Printf("[CHAT] Resposta gerada com %d fontes\n", len(resp.Sources))

	if h.chatRepo != nil && employeeID != "" {
		msgID := "msg-" + uuid.NewString()
		now := time.Now().UTC().Format("2006-01-02 15:04:05")
		sourcesCount := len(resp.Sources)
		err := h.chatRepo.InsertMessage(r.Context(), repository.ChatMessage{
			ID:          msgID,
			EmployeeID:  employeeID,
			Role:        "assistant",
			Content:     resp.Answer,
			AllowWebSearch: body.AllowWebSearch,
			SourcesCount: sourcesCount,
			CreatedAt:   now,
		})
		if err != nil {
			fmt.Printf("[CHAT] Erro salvando mensagem do assistente: %v\n", err)
		}
	}

	out := ChatResponse{
		Answer:  resp.Answer,
		Sources: make([]SourceDTO, 0, len(resp.Sources)),
	}

	// Se o usuário pediu um documento, inicia o workflow completo.
	// O chat não gera o documento diretamente — ele delega para o workflow,
	// que é o único caminho de geração documental.
	if isDocumentRequest(body.Question) && h.workflow != nil && employeeID != "" {
		fmt.Println("[CHAT] Detectado pedido de documento, iniciando workflow...")
		result, err := h.workflow.HandleDocumentRequest(r.Context(), employeeID, body.Question)
		if err != nil {
			fmt.Printf("[CHAT] Erro no workflow de documento: %v\n", err)
			out.Answer += "\n\nOcorreu um erro ao processar o pedido de documento."
		} else if result.Blocked {
			out.Answer += "\n\n" + result.Message
		} else if result.DocumentRun != nil {
			out.DocumentRunID = result.DocumentRun.ID
			out.DocumentDocxURL = result.DocxURL
			out.DocumentPdfURL = result.PdfURL
			out.Answer += "\n\n---\n📄 Documento gerado: [" + result.Template.Name + "](" + out.DocumentDocxURL + ")"
		} else if len(result.MissingFields) > 0 {
			out.Answer += "\n\n" + result.Message
		}
	}

	for _, src := range resp.Sources {
		out.Sources = append(out.Sources, SourceDTO{
			DocumentID: src.Document.ID,
			Title:      src.Document.Title,
			Source:     src.Document.SourcePath,
			ChunkOrd:   src.Chunk.Ord,
			Content:    src.Chunk.Content,
			Snippet:    src.Snippet,
			Score:      src.Score,
		})
	}

	writeJSON(w, http.StatusOK, out)
}

// buildTaskDataFromEmployeeAndQuestion was removed — the chat handler no longer
// invents values. Data collection now happens through the workflow pipeline
// (ProcessMessage / SetData), which only stores values the user explicitly provided.
// The unified HandleDocumentRequest method handles the full flow.

func isDocumentRequest(question string) bool {
	q := strings.ToLower(question)
	return strings.Contains(q, "documento") ||
		strings.Contains(q, "modelo") ||
		strings.Contains(q, "formulário") ||
		strings.Contains(q, "formulario") ||
		strings.Contains(q, "solicitação") ||
		strings.Contains(q, "solicitacao")
}

// GenerateDocument (POST /api/chat/generate-document) inicia o fluxo de geração
// de documento via workflow. O chat NUNCA escolhe templates arbitrariamente
// nem inventa valores default — a geração passa integralmente pelo workflow.
func (h *ChatHandler) GenerateDocument(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[CHAT] GenerateDocument handler iniciado")
	var body GenerateDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		fmt.Printf("[CHAT] Erro decode JSON: %v\n", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	fmt.Printf("[CHAT] GenerateDocument pergunta recebida: %s\n", body.Question)

	if strings.TrimSpace(body.Question) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "question is required"})
		return
	}

	employeeID, _ := r.Context().Value("employee_id").(string)
	if employeeID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "employee not authenticated"})
		return
	}

	// Inicia o workflow completo de geração de documento.
	// O chat é apenas um ponto de entrada — a lógica real vive no workflow.
	result, err := h.workflow.HandleDocumentRequest(r.Context(), employeeID, body.Question)
	if err != nil {
		fmt.Printf("[CHAT] GenerateDocument erro workflow: %v\n", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	if result.Blocked {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"message": result.Message,
			"blocked": true,
		})
		return
	}

	if result.DocumentRun == nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"message":       result.Message,
			"missingFields": result.MissingFields,
		})
		return
	}

	writeJSON(w, http.StatusOK, GenerateDocumentResponse{
		DocumentRunID: result.DocumentRun.ID,
		DocxURL:       result.DocxURL,
		PdfURL:        result.PdfURL,
		Message:       result.Message,
	})
}
