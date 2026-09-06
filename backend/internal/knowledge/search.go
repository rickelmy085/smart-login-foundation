package knowledge

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"unicode"
)

// Searcher faz busca lexical com FTS5 (BM25).
type Searcher struct {
	db *sql.DB
}

func NewSearcher(db *sql.DB) *Searcher {
	return &Searcher{db: db}
}

// SearchOptions permite customizar a busca.
type SearchOptions struct {
	Limit    int    // número máximo de hits (default 10)
	DocID    string // restringe a busca a um documento (opcional)
	MinScore float64 // limiar mínimo de relevância BM25 (menor = melhor; default 15.0)
	MaxChunksPerDoc int // máximo de chunks por documento para diversidade (default 3)
}

// SanitizeFTS5 limpa a query do usuário para uso seguro no FTS5.
// Mantém apenas letras (unicode), números e espaços. Tudo o mais
// (aspas, curingas, operadores booleanos, parênteses) é removido.
// O resultado é uma sequência de palavras separadas por espaço,
// que o FTS5 interpreta como AND implícito.
func SanitizeFTS5(q string) string {
	var b strings.Builder
	prevSpace := false
	for _, r := range strings.TrimSpace(q) {
		switch {
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			b.WriteRune(r)
			prevSpace = false
		case unicode.IsSpace(r):
			if !prevSpace {
				b.WriteRune(' ')
			}
			prevSpace = true
		default:
			// Remove pontuação, aspas, curingas, operadores, etc.
			prevSpace = false
		}
	}
	return strings.TrimSpace(b.String())
}

// Search executa MATCH no FTS5 e devolve os trechos mais relevantes.
// Score menor = mais relevante (convenção BM25 do SQLite).
//
// A query é sanitizada; termos muito comuns (stopwords do unicode61)
// são ignorados automaticamente pelo tokenizer.
func (s *Searcher) Search(ctx context.Context, query string, opt SearchOptions) ([]SearchHit, error) {
	fmt.Printf("[KNOWLEDGE] Search query=%s limit=%d docID=%s\n", query, opt.Limit, opt.DocID)
	if opt.Limit <= 0 {
		opt.Limit = 10
	}
	if opt.MinScore <= 0 {
		opt.MinScore = 15.0
	}
	if opt.MaxChunksPerDoc <= 0 {
		opt.MaxChunksPerDoc = 3
	}

	q := SanitizeFTS5(query)
	if q == "" {
		fmt.Println("[KNOWLEDGE] Search query vazia apos sanitize")
		return nil, nil
	}

	significant := significantTermsString(q)
	if significant == "" {
		fmt.Println("[KNOWLEDGE] Search nenhum termo significativo")
		return nil, nil
	}
	escaped := strings.ReplaceAll(significant, "'", "''")
	fmt.Printf("[KNOWLEDGE] Search significant=%s escaped=%s\n", significant, escaped)

	// Junta chunks (texto) + documents (metadados) + score bm25().
	sqlStmt := `
		SELECT
			c.id, c.document_id, c.ord, c.content, c.char_start, c.char_end,
			d.source_path, d.title, d.page_count, d.char_count, d.hash,
			bm25(chunks_fts) AS rank,
			snippet(chunks_fts, 0, '<<', '>>', '…', 12) AS snip
		FROM chunks_fts
		JOIN chunks     AS c ON c.id = chunks_fts.rowid
		JOIN documents  AS d ON d.id = c.document_id
		WHERE chunks_fts MATCH '` + escaped + `'
	`
	args := []any{}
	if opt.DocID != "" {
		sqlStmt += ` AND d.id = ? `
		args = append(args, opt.DocID)
	}
	sqlStmt += ` ORDER BY rank LIMIT ? `
	args = append(args, opt.Limit)

	rows, err := s.db.QueryContext(ctx, sqlStmt, args...)
	if err != nil {
		fmt.Printf("[KNOWLEDGE] Search erro query: %v\n", err)
		return nil, fmt.Errorf("fts5 query: %w", err)
	}
	defer rows.Close()

	var hits []SearchHit
	for rows.Next() {
		var h SearchHit
		var rank float64
		if err := rows.Scan(
			&h.Chunk.ID, &h.Chunk.DocumentID, &h.Chunk.Ord, &h.Chunk.Content,
			&h.Chunk.CharStart, &h.Chunk.CharEnd,
			&h.Document.SourcePath, &h.Document.Title, &h.Document.PageCount,
			&h.Document.CharCount, &h.Document.Hash,
			&rank, &h.Snippet,
		); err != nil {
			fmt.Printf("[KNOWLEDGE] Search erro scan: %v\n", err)
			return nil, err
		}
		h.Score = rank
		h.Document.ID = h.Chunk.DocumentID
		hits = append(hits, h)
	}
	fmt.Printf("[KNOWLEDGE] Search sucesso: %d hits\n", len(hits))

	// Filtra por limiar de relevância BM25.
	filtered := make([]SearchHit, 0, len(hits))
	for _, h := range hits {
		if h.Score <= opt.MinScore {
			filtered = append(filtered, h)
		} else {
			fmt.Printf("[KNOWLEDGE] Search hit descartado por score chunk=%d doc=%s score=%.4f\n",
				h.Chunk.ID, h.Document.ID, h.Score)
		}
	}
	fmt.Printf("[KNOWLEDGE] Search apos filtro de relevancia: %d hits\n", len(filtered))

	// Deduplicação/diversidade: limita chunks por mesmo documento.
	perDoc := make(map[string]int)
	diverse := make([]SearchHit, 0, len(filtered))
	for _, h := range filtered {
		n := perDoc[h.Document.ID]
		if n >= opt.MaxChunksPerDoc {
			fmt.Printf("[KNOWLEDGE] Search hit descartado por diversidade doc=%s chunk=%d\n",
				h.Document.ID, h.Chunk.ID)
			continue
		}
		perDoc[h.Document.ID] = n + 1
		diverse = append(diverse, h)
	}
	fmt.Printf("[KNOWLEDGE] Search apos diversidade: %d hits\n", len(diverse))

	return diverse, rows.Err()
}

// Stats devolve totais (útil pra CLI e pra mostrar no dashboard).
type Stats struct {
	Documents int `json:"documents"`
	Chunks    int `json:"chunks"`
}

func (s *Searcher) Stats(ctx context.Context) (Stats, error) {
	var st Stats
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM documents`).Scan(&st.Documents); err != nil {
		if err == sql.ErrNoRows {
			return st, nil
		}
		return st, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM chunks`).Scan(&st.Chunks); err != nil {
		if err == sql.ErrNoRows {
			return st, nil
		}
		return st, err
	}
	return st, nil
}
