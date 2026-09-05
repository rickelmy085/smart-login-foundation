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
	Limit int    // número máximo de hits (default 10)
	DocID string // restringe a busca a um documento (opcional)
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
	if opt.Limit <= 0 {
		opt.Limit = 10
	}
	q := SanitizeFTS5(query)
	if q == "" {
		return nil, nil
	}

	significant := significantTermsString(q)
	if significant == "" {
		return nil, nil
	}
	escaped := strings.ReplaceAll(significant, "'", "''")

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
			return nil, err
		}
		h.Score = rank
		h.Document.ID = h.Chunk.DocumentID
		hits = append(hits, h)
	}
	return hits, rows.Err()
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
