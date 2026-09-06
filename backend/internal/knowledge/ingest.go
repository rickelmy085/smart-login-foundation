package knowledge

import (
	"context"
	"database/sql"
	"fmt" // debug prints
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
	fmt.Println("[KNOWLEDGE] NewIngestor criado")
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
	fmt.Printf("[KNOWLEDGE] IngestDir iniciado root=%s\n", root)
	started := time.Now()
	files, err := ScanDir(root)
	if err != nil {
		fmt.Printf("[KNOWLEDGE] IngestDir erro scan dir: %v\n", err)
		return IngestStats{}, fmt.Errorf("scan dir %s: %w", root, err)
	}

	var stats IngestStats
	stats.Scanned = len(files)
	fmt.Printf("[KNOWLEDGE] IngestDir arquivos escaneados: %d\n", stats.Scanned)

	for i, sf := range files {
		select {
		case <-ctx.Done():
			fmt.Println("[KNOWLEDGE] IngestDir cancelado")
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
		fmt.Printf("[KNOWLEDGE] IngestDir processando arquivo %d/%d: %s\n", i+1, stats.Scanned, sf.Path)

		// Deduplicação por hash.
		existingID, _ := ing.repo.FindByHash(ctx, sf.Hash)
		if existingID != "" {
			fmt.Printf("[KNOWLEDGE] IngestDir arquivo ja ingerido hash=%s\n", sf.Hash)
			stats.Skipped++
			if ing.OnFile != nil {
				ing.OnFile(i+1, stats.Scanned, sf, nil)
			}
			continue
		}

		chunks, err := ing.processOne(ctx, sf)
		if err != nil {
			fmt.Printf("[KNOWLEDGE] IngestDir erro processOne: %v\n", err)
			slog.Warn("ingest failed", "file", sf.Path, "error", err.Error())
			stats.Failed++
			if ing.OnFile != nil {
				ing.OnFile(i+1, stats.Scanned, sf, err)
			}
			continue
		}

		stats.Ingested++
		stats.ChunksTotal += len(chunks)
		fmt.Printf("[KNOWLEDGE] IngestDir sucesso ingested=%d chunks=%d\n", stats.Ingested, stats.ChunksTotal)
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
	fmt.Printf("[KNOWLEDGE] IngestDir finalizado scanned=%d ingested=%d skipped=%d failed=%d chunks=%d duration=%v\n",
		stats.Scanned, stats.Ingested, stats.Skipped, stats.Failed, stats.ChunksTotal, stats.Duration)
	return stats, nil
}

// processOne faz: extract → normalize → chunk → grava.
// Tudo dentro de uma transação para que um PDF só apareça no banco
// se foi ingerido inteiro.
func (ing *Ingestor) processOne(ctx context.Context, sf ScannedFile) ([]Chunk, error) {
	fmt.Printf("[KNOWLEDGE] processOne iniciado path=%s\n", sf.Path)
	res, err := ExtractPDF(sf.Path)
	if err != nil {
		fmt.Printf("[KNOWLEDGE] processOne erro extract PDF: %v\n", err)
		return nil, err
	}
	fmt.Printf("[KNOWLEDGE] processOne PDF extraido pages=%d text_length=%d\n", res.PageCount, len(res.Text))
	text := Normalize(res.Text)
	if text == "" {
		fmt.Printf("[KNOWLEDGE] processOne texto vazio apos normalize\n")
		return nil, fmt.Errorf("empty text after extraction")
	}
	fmt.Printf("[KNOWLEDGE] processOne texto normalizado length=%d\n", len(text))

	doc := Document{
		ID:         sf.Hash,
		SourcePath: sf.Path,
		Title:      sf.Title,
		PageCount:  res.PageCount,
		CharCount:  len(text),
		Hash:       sf.Hash,
	}
	chunks := Split(text, ing.opts)
	fmt.Printf("[KNOWLEDGE] processOne chunks gerados: %d\n", len(chunks))

	// Transação: document + chunks em um único commit.
	tx, err := ing.repo.db.BeginTx(ctx, nil)
	if err != nil {
		fmt.Printf("[KNOWLEDGE] processOne erro begin tx: %v\n", err)
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
		fmt.Printf("[KNOWLEDGE] processOne erro insert document: %v\n", err)
		return nil, err
	}
	fmt.Printf("[KNOWLEDGE] processOne document inserido/atualizado id=%s\n", doc.ID)

	// Limpa chunks anteriores (re-ingest limpo) e reinsere.
	if _, err := tx.ExecContext(ctx, `DELETE FROM chunks WHERE document_id = ?`, doc.ID); err != nil {
		fmt.Printf("[KNOWLEDGE] processOne erro delete chunks: %v\n", err)
		return nil, err
	}

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO chunks (document_id, ord, content, char_start, char_end)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		fmt.Printf("[KNOWLEDGE] processOne erro prepare stmt: %v\n", err)
		return nil, err
	}
	defer stmt.Close()

	for _, c := range chunks {
		if _, err := stmt.ExecContext(ctx, doc.ID, c.Ord, c.Content, c.CharStart, c.CharEnd); err != nil {
			fmt.Printf("[KNOWLEDGE] processOne erro insert chunk: %v\n", err)
			return nil, err
		}
	}
	fmt.Printf("[KNOWLEDGE] processOne %d chunks inseridos\n", len(chunks))

	if err := tx.Commit(); err != nil {
		fmt.Printf("[KNOWLEDGE] processOne erro commit: %v\n", err)
		return nil, err
	}
	fmt.Printf("[KNOWLEDGE] processOne commit realizado com sucesso\n")
	return chunks, nil
}
