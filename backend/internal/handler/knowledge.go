// Package handler: endpoints HTTP da camada de conhecimento.
// Mantido separado dos handlers de auth (auth.go) para isolar
// responsabilidades. O endpoint é protegido pelo AuthMiddleware
// para garantir que apenas usuários autenticados consultem o RAG.
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/bsmart/abis/internal/knowledge"
)

// KnowledgeHandler expõe endpoints de busca sobre o índice FTS5.
type KnowledgeHandler struct {
	searcher *knowledge.Searcher
}

func NewKnowledgeHandler(s *knowledge.Searcher) *KnowledgeHandler {
	return &KnowledgeHandler{searcher: s}
}

// SearchRequest é o corpo aceito por POST /api/search.
//
// Estratégia: aceitar tanto query string quanto JSON para flexibilidade
// (CLI/cURL com ?q=, ou front-end com body).
type SearchRequest struct {
	Query string `json:"q"`
	Limit int    `json:"limit"`
}

// SearchHitResponse é o que devolvemos ao cliente. Mantemos só os campos
// relevantes para o front-end e omitimos paths absolutos do servidor.
type SearchHitResponse struct {
	DocumentID string  `json:"documentId"`
	Title      string  `json:"title"`
	Source     string  `json:"source"`
	ChunkOrd   int     `json:"chunkOrd"`
	Content    string  `json:"content"`
	Snippet    string  `json:"snippet"`
	Score      float64 `json:"score"`
}

// Search (POST /api/search) faz busca lexical FTS5.
// Aceita ?q=...&limit=... ou body JSON {"q":"...", "limit":10}.
func (h *KnowledgeHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	limitStr := r.URL.Query().Get("limit")
	fmt.Printf("[KNOWLEDGE] Search query=%s limitStr=%s\n", q, limitStr)

	// Se não veio em query string, tenta ler do body.
	if q == "" && r.ContentLength > 0 {
		var body SearchRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			fmt.Printf("[KNOWLEDGE] Search erro decode JSON: %v\n", err)
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		q = body.Query
		if body.Limit > 0 {
			limitStr = strconv.Itoa(body.Limit)
		}
		fmt.Printf("[KNOWLEDGE] Search body query=%s limit=%d\n", q, body.Limit)
	}

	if q == "" {
		fmt.Println("[KNOWLEDGE] Search query vazia")
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "q is required"})
		return
	}

	limit := 10
	if limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 && n <= 50 {
			limit = n
		}
	}
	fmt.Printf("[KNOWLEDGE] Search executando busca q=%s limit=%d\n", q, limit)

	hits, err := h.searcher.Search(r.Context(), q, knowledge.SearchOptions{Limit: limit})
	if err != nil {
		fmt.Printf("[KNOWLEDGE] Search erro: %v\n", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	fmt.Printf("[KNOWLEDGE] Search sucesso: %d resultados\n", len(hits))

	resp := make([]SearchHitResponse, 0, len(hits))
	for _, hit := range hits {
		resp = append(resp, SearchHitResponse{
			DocumentID: hit.Document.ID,
			Title:      hit.Document.Title,
			Source:     hit.Document.SourcePath,
			ChunkOrd:   hit.Chunk.Ord,
			Content:    hit.Chunk.Content,
			Snippet:    hit.Snippet,
			Score:      hit.Score,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"query":   q,
		"count":   len(resp),
		"results": resp,
	})
}
