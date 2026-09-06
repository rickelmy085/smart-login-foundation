// Handler HTTP do chat (RAG). Mantido separado do handler de auth/knowledge
// para isolar responsabilidades.
package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
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

	// Se o usuário pediu um documento, gera automaticamente usando dados reais
	if isDocumentRequest(body.Question) && h.workflow != nil && employeeID != "" {
		fmt.Println("[CHAT] Detectado pedido de documento, gerando...")
		templates, err := h.workflow.ListTemplates(r.Context())
		if err == nil && len(templates) > 0 {
			template := templates[0]
			taskData := buildTaskDataFromEmployeeAndQuestion(r.Context(), employeeID, body.Question, h.employeeRepo)
			result, err := h.workflow.GenerateDocumentFromTemplate(r.Context(), template.ID, employeeID, taskData)
			if err == nil {
				out.DocumentRunID = result.ID
				out.DocumentDocxURL = "/api/documents/" + result.ID + "/docx"
				out.DocumentPdfURL = "/api/documents/" + result.ID + "/pdf"
				out.Answer += "\n\n---\n📄 Documento gerado: [" + template.Name + "](" + out.DocumentDocxURL + ")"
			} else {
				fmt.Printf("[CHAT] Erro ao gerar documento: %v\n", err)
			}
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

// buildTaskDataFromEmployeeAndQuestion extrai dados do funcionário e da pergunta
// para preencher automaticamente os campos do template de documento.
func buildTaskDataFromEmployeeAndQuestion(ctx context.Context, employeeID, question string, empRepo *repository.EmployeeRepo) map[string]string {
	taskData := map[string]string{
		"area_solicitante":   "Área de Crédito",
		"fornecedor":         "N/A",
		"valor":              "R$ 10.000,00",
		"justificativa":      question,
		"categoria_produto":  "Crédito",
		"aprovacao_cade":     "Pendente",
		"observacoes":        "Documento gerado via chat ABIS",
		"data_solicitacao":   time.Now().Format("02/01/2006"),
		"numero_solicitacao": "SJ-" + uuid.NewString()[:8],
	}

	// Busca dados reais do funcionário
	if empRepo != nil {
		if emp, err := empRepo.FindByID(ctx, employeeID); err == nil {
			taskData["solicitante"] = emp.Name
			taskData["area_solicitante"] = emp.Role
		} else {
			fmt.Printf("[CHAT] Erro buscando employee para documento: %v\n", err)
			taskData["solicitante"] = "Colaborador " + employeeID
		}
	} else {
		taskData["solicitante"] = "Colaborador " + employeeID
	}

	// Extrai valor (ex: "10 mil reais", "R$ 50.000", "R$ 100000,00")
	taskData["valor"] = extractValor(question)

	// Extrai categoria a partir de palavras-chave
	taskData["categoria_produto"] = extractCategoria(question)

	return taskData
}

// extractValor busca um valor monetário na pergunta.
func extractValor(question string) string {
	// Tenta formato "R$ 10.000,00" ou "R$ 10.000" ou "R$ 10000,00"
	re := regexp.MustCompile(`(?:R\$\s*)?([\d.]+)(?:,(\d{1,2}))?`)
	matches := re.FindStringSubmatch(question)
	if len(matches) >= 2 {
		intPart := strings.ReplaceAll(matches[1], ".", "")
		intPart = strings.ReplaceAll(intPart, ",", "")
		if v, err := strconv.ParseFloat(intPart, 64); err == nil {
			if v >= 1000 {
				return fmt.Sprintf("R$ %s", formatCurrencyBR(v))
			}
		}
	}

	// Tenta formato "10 mil" → 10000
	reMil := regexp.MustCompile(`(?i)(\d+)\s*mil`)
	milMatch := reMil.FindStringSubmatch(question)
	if milMatch != nil {
		if v, err := strconv.ParseFloat(milMatch[1], 64); err == nil {
			return fmt.Sprintf("R$ %s", formatCurrencyBR(v * 1000))
		}
	}

	return "R$ 10.000,00"
}

// formatCurrencyBR formata um float como moeda brasileira: 10000 → "10.000,00"
func formatCurrencyBR(v float64) string {
	formatted := strconv.FormatFloat(v, 'f', 2, 64)
	parts := strings.Split(formatted, ".")
	intPart := parts[0]
	varDec := "00"
	if len(parts) > 1 {
		varDec = parts[1]
	}
	var b strings.Builder
	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			b.WriteString(".")
		}
		b.WriteRune(c)
	}
	result := b.String()
	if len(varDec) < 2 {
		varDec = varDec + "0"
	}
	return result + "," + varDec
}

// extractCategoria identifica a categoria do produto a partir de palavras-chave.
func extractCategoria(question string) string {
	q := strings.ToLower(question)
	if strings.Contains(q, "empréstimo") || strings.Contains(q, "emprestimo") || strings.Contains(q, "crédito") || strings.Contains(q, "credito") {
		return "Crédito"
	}
	if strings.Contains(q, "seguro") {
		return "Seguros"
	}
	if strings.Contains(q, "investimento") || strings.Contains(q, "aplicação") || strings.Contains(q, "aplicacao") {
		return "Investimentos"
	}
	if strings.Contains(q, "cartão") || strings.Contains(q, "cartao") {
		return "Cartões"
	}
	return "Crédito"
}

func isDocumentRequest(question string) bool {
	q := strings.ToLower(question)
	return strings.Contains(q, "documento") ||
		strings.Contains(q, "modelo") ||
		strings.Contains(q, "formulário") ||
		strings.Contains(q, "formulario") ||
		strings.Contains(q, "solicitação") ||
		strings.Contains(q, "solicitacao")
}

// GenerateDocument (POST /api/chat/generate-document) recebe um pedido de documento,
// busca o template mais adequado via RAG e gera um DOCX/PDF.
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

	// Buscar template mais adequado via RAG
	templates, err := h.workflow.ListTemplates(r.Context())
	if err != nil {
		fmt.Printf("[CHAT] GenerateDocument erro list templates: %v\n", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list templates"})
		return
	}
	if len(templates) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no templates available"})
		return
	}

	// Usar o primeiro template disponível (em produção, classificaria a intenção)
	template := templates[0]
	fmt.Printf("[CHAT] GenerateDocument usando template: %s\n", template.Name)

	// Extrair dados reais do funcionário e da pergunta
	taskData := buildTaskDataFromEmployeeAndQuestion(r.Context(), employeeID, body.Question, h.employeeRepo)

	// Gerar documento usando o workflow service
	result, err := h.workflow.GenerateDocumentFromTemplate(r.Context(), template.ID, employeeID, taskData)
	if err != nil {
		fmt.Printf("[CHAT] GenerateDocument erro generate: %v\n", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, GenerateDocumentResponse{
		DocumentRunID: result.ID,
		DocxURL:       "/api/documents/" + result.ID + "/docx",
		PdfURL:        "/api/documents/" + result.ID + "/pdf",
		Message:       "Documento gerado com sucesso",
	})
}
