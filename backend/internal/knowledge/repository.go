// Repository do pacote knowledge: SQL específico das tabelas documents/chunks.
// Mantém a separação de camadas: knowledge → repository → database/sql.
package knowledge

import (
	"context"
	"database/sql"
)

// DocumentRepo encapsula o acesso às tabelas documents/chunks/chunks_fts.
type DocumentRepo struct {
	db *sql.DB
}

func NewDocumentRepo(db *sql.DB) *DocumentRepo {
	return &DocumentRepo{db: db}
}

// FindByHash devolve o ID interno (== hash) de um documento já indexado,
// ou "" se ele ainda não existe. Usado para deduplicação no ingest.
func (r *DocumentRepo) FindByHash(ctx context.Context, hash string) (string, error) {
	var id string
	err := r.db.QueryRowContext(ctx, `SELECT id FROM documents WHERE hash = ?`, hash).Scan(&id)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return id, err
}

// InsertDocument grava o cabeçalho do documento (id = hash).
// Se o doc já existir (ON CONFLICT), atualiza metadados e timestamp.
func (r *DocumentRepo) InsertDocument(ctx context.Context, d Document) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO documents (id, source_path, title, page_count, char_count, hash, status)
		VALUES (?, ?, ?, ?, ?, ?, 'ok')
		ON CONFLICT(id) DO UPDATE SET
			source_path = excluded.source_path,
			title       = excluded.title,
			page_count  = excluded.page_count,
			char_count  = excluded.char_count,
			ingested_at = datetime('now')
	`, d.ID, d.SourcePath, d.Title, d.PageCount, d.CharCount, d.Hash)
	return err
}

// DeleteChunks remove todos os chunks antigos de um documento
// (usado antes de re-ingerir para não duplicar).
func (r *DocumentRepo) DeleteChunks(ctx context.Context, documentID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM chunks WHERE document_id = ?`, documentID)
	return err
}

// InsertChunk grava um chunk. O trigger chunks_ai cuida do FTS5.
func (r *DocumentRepo) InsertChunk(ctx context.Context, documentID string, ord int, content string, start, end int) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO chunks (document_id, ord, content, char_start, char_end)
		VALUES (?, ?, ?, ?, ?)
	`, documentID, ord, content, start, end)
	return err
}

// CountDocuments e CountChunks são helpers para logs/observabilidade.
func (r *DocumentRepo) CountDocuments(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM documents`).Scan(&n)
	return n, err
}

func (r *DocumentRepo) CountChunks(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT count(*) FROM chunks`).Scan(&n)
	return n, err
}
