// Handler HTTP do chat (RAG). Mantido separado do handler de auth/knowledge
// para isolar responsabilidades.
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/bsmart/abis/internal/chat"
)

// ChatHandler expõe o endpoint /api/chat.
type ChatHandler struct {
	svc *chat.Service
}

func NewChatHandler(s *chat.Service) *ChatHandler {
	return &ChatHandler{svc: s}
}

// ChatRequest é o corpo aceito por POST /api/chat.
type ChatRequest struct {
	Question string `json:"question"`
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
	Answer  string      `json:"answer"`
	Sources []SourceDTO `json:"sources"`
}

// Chat (POST /api/chat) recebe a pergunta, executa o pipeline RAG
// (busca FTS5 + prompt + Groq) e devolve a resposta.
//
// Protegido pelo middleware de auth (rota exige JWT válido).
func (h *ChatHandler) Chat(w http.ResponseWriter, r *http.Request) {
	var body ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	resp, err := h.svc.Ask(r.Context(), chat.ChatRequest{Question: body.Question})
	if err != nil {
		// Erros de validação são 400; erros de LLM são 502 (upstream problem).
		status := http.StatusBadGateway
		if err == chat.ErrEmptyQuestion {
			status = http.StatusBadRequest
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	out := ChatResponse{
		Answer:  resp.Answer,
		Sources: make([]SourceDTO, 0, len(resp.Sources)),
	}
	for _, h := range resp.Sources {
		out.Sources = append(out.Sources, SourceDTO{
			DocumentID: h.Document.ID,
			Title:      h.Document.Title,
			Source:     h.Document.SourcePath,
			ChunkOrd:   h.Chunk.Ord,
			Content:    h.Chunk.Content,
			Snippet:    h.Snippet,
			Score:      h.Score,
		})
	}

	writeJSON(w, http.StatusOK, out)
}
