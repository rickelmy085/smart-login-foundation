package knowledge

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"
)

// Ingestor orquestra o pipeline: scan → extract → normalize → chunk → DB.
// Mantém o estado de progresso (totais) e expõe callbacks opcionais
// para o chamador (CLI, servidor, job) acompanhar em tempo real.
type Ingestor struct {
	repo   *DocumentRepo
	opts   ChunkOptions
	OnFile func(current, total int, sf ScannedFile, err error) // opcional
}

// NewIngestor monta um ingestor. ChunkOptions{0,0} usa defaults.
func NewIngestor(db *sql.DB, opts ChunkOptions) *Ingestor {
	return &Ingestor{
		repo: NewDocumentRepo(db),
		opts: opts,
	}
}

// IngestStats sumariza o que aconteceu.
type IngestStats struct {
	Scanned     int
	Ingested    int
	Skipped     int
	Failed      int
	ChunksTotal int
	Duration    time.Duration
}

// IngestDir varre `root`, processa cada PDF e persiste.
// Idempotente: PDFs já presentes (mesmo hash) são pulados.
func (ing *Ingestor) IngestDir(ctx context.Context, root string) (IngestStats, error) {
	started := time.Now()
	files, err := ScanDir(root)
	if err != nil {
		return IngestStats{}, fmt.Errorf("scan dir %s: %w", root, err)
	}

	var stats IngestStats
	stats.Scanned = len(files)

	for i, sf := range files {
		select {
		case <-ctx.Done():
			return stats, ctx.Err()
		default:
		}

		if !IsPDF(sf.Path) {
			stats.Skipped++
			if ing.OnFile != nil {
				ing.OnFile(i+1, stats.Scanned, sf, fmt.Errorf("not a valid PDF header"))
			}
			continue
		}

		// Deduplicação por hash.
		existingID, _ := ing.repo.FindByHash(ctx, sf.Hash)
		if existingID != "" {
			stats.Skipped++
			if ing.OnFile != nil {
				ing.OnFile(i+1, stats.Scanned, sf, nil)
			}
			continue
		}

		chunks, err := ing.processOne(ctx, sf)
		if err != nil {
			slog.Warn("ingest failed", "file", sf.Path, "error", err.Error())
			stats.Failed++
			if ing.OnFile != nil {
				ing.OnFile(i+1, stats.Scanned, sf, err)
			}
			continue
		}

		stats.Ingested++
		stats.ChunksTotal += len(chunks)
		if ing.OnFile != nil {
			ing.OnFile(i+1, stats.Scanned, sf, nil)
		}
	}

	stats.Duration = time.Since(started)
	slog.Info("ingest finished",
		"scanned", stats.Scanned,
		"ingested", stats.Ingested,
		"skipped", stats.Skipped,
		"failed", stats.Failed,
		"chunks", stats.ChunksTotal,
		"duration", stats.Duration.String(),
	)
	return stats, nil
}

// processOne faz: extract → normalize → chunk → grava.
// Tudo dentro de uma transação para que um PDF só apareça no banco
// se foi ingerido inteiro.
func (ing *Ingestor) processOne(ctx context.Context, sf ScannedFile) ([]Chunk, error) {
	res, err := ExtractPDF(sf.Path)
	if err != nil {
		return nil, err
	}
	text := Normalize(res.Text)
	if text == "" {
		return nil, fmt.Errorf("empty text after extraction")
	}

	doc := Document{
		ID:         sf.Hash,
		SourcePath: sf.Path,
		Title:      sf.Title,
		PageCount:  res.PageCount,
		CharCount:  len(text),
		Hash:       sf.Hash,
	}
	chunks := Split(text, ing.opts)

	// Transação: document + chunks em um único commit.
	tx, err := ing.repo.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO documents (id, source_path, title, page_count, char_count, hash, status)
		VALUES (?, ?, ?, ?, ?, ?, 'ok')
		ON CONFLICT(id) DO UPDATE SET
			source_path = excluded.source_path,
			title       = excluded.title,
			page_count  = excluded.page_count,
			char_count  = excluded.char_count,
			ingested_at = datetime('now')
	`, doc.ID, doc.SourcePath, doc.Title, doc.PageCount, doc.CharCount, doc.Hash); err != nil {
		return nil, err
	}

	// Limpa chunks anteriores (re-ingest limpo) e reinsere.
	if _, err := tx.ExecContext(ctx, `DELETE FROM chunks WHERE document_id = ?`, doc.ID); err != nil {
		return nil, err
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO chunks (document_id, ord, content, char_start, char_end)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	for _, c := range chunks {
		if _, err := stmt.ExecContext(ctx, doc.ID, c.Ord, c.Content, c.CharStart, c.CharEnd); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return chunks, nil
}
