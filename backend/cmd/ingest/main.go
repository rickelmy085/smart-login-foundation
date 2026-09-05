// cmd/ingest é o CLI para indexar PDFs da pasta storage/normativos no SQLite.
//
// Uso:
//
//	go run ./cmd/ingest                  # ingere ./storage/normativos
//	go run ./cmd/ingest -dir=outro/path  # ingere outra pasta
//	go run ./cmd/ingest -stats           # só mostra totais e sai
//	go run ./cmd/ingest -query="compliance" -limit=5  # busca e imprime
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bsmart/abis/internal/config"
	"github.com/bsmart/abis/internal/database"
	"github.com/bsmart/abis/internal/knowledge"
)

func main() {
	dir := flag.String("dir", "storage/normativos", "diretório com PDFs para indexar (relativo a backend/)")
	showStats := flag.Bool("stats", false, "apenas mostra documentos/chunks indexados e sai")
	query := flag.String("query", "", "se informado, faz busca e imprime resultados em vez de ingerir")
	limit := flag.Int("limit", 5, "limite de resultados da busca")
	flag.Parse()

	cfg := config.Load()
	db, err := database.Connect(cfg.DatabasePath)
	if err != nil {
		slog.Error("connect db", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx := context.Background()
	if err := database.MigrateKnowledge(ctx, db); err != nil {
		slog.Error("migrate knowledge", "error", err)
		os.Exit(1)
	}

	searcher := knowledge.NewSearcher(db)

	// Modo busca
	if *query != "" {
		hits, err := searcher.Search(ctx, *query, knowledge.SearchOptions{Limit: *limit})
		if err != nil {
			slog.Error("search", "error", err)
			os.Exit(1)
		}
		printHits(*query, hits)
		return
	}

	// Modo stats
	if *showStats {
		st, err := searcher.Stats(ctx)
		if err != nil {
			slog.Error("stats", "error", err)
			os.Exit(1)
		}
		fmt.Printf("Documents: %d\nChunks: %d\n", st.Documents, st.Chunks)
		return
	}

	// Modo ingest
	abs, _ := filepath.Abs(*dir)
	fmt.Printf("Ingerindo PDFs de %s ...\n\n", abs)

	ing := knowledge.NewIngestor(db, knowledge.ChunkOptions{Size: 800, Overlap: 100})
	ing.OnFile = func(current, total int, sf knowledge.ScannedFile, err error) {
		if err != nil {
			fmt.Printf("  [%2d/%2d] ✗ %s  (%v)\n", current, total, sf.Title, err)
			return
		}
		fmt.Printf("  [%2d/%2d] ✓ %s\n", current, total, sf.Title)
	}

	started := time.Now()
	stats, err := ing.IngestDir(ctx, *dir)
	if err != nil {
		slog.Error("ingest", "error", err)
		os.Exit(1)
	}

	fmt.Printf("\nResumo:\n")
	fmt.Printf("  PDFs encontrados : %d\n", stats.Scanned)
	fmt.Printf("  Ingeridos        : %d\n", stats.Ingested)
	fmt.Printf("  Já indexados     : %d\n", stats.Skipped)
	fmt.Printf("  Falharam         : %d\n", stats.Failed)
	fmt.Printf("  Chunks gerados   : %d\n", stats.ChunksTotal)
	fmt.Printf("  Tempo total      : %s\n", stats.Duration.Round(time.Millisecond))
	fmt.Printf("  Total decorrido  : %s\n", time.Since(started).Round(time.Millisecond))
}

// printHits formata os hits de busca de forma legível.
func printHits(query string, hits []knowledge.SearchHit) {
	fmt.Printf("Busca: %q\n", query)
	if len(hits) == 0 {
		fmt.Println("  (sem resultados)")
		return
	}
	for i, h := range hits {
		fmt.Printf("\n#%d  score=%.4f  %s\n", i+1, h.Score, h.Document.Title)
		fmt.Printf("     source: %s\n", filepath.Base(h.Document.SourcePath))
		// snippet vem com << >> marcando o match; destaca visualmente.
		snip := strings.ReplaceAll(h.Snippet, "<<", "\033[1;33m")
		snip = strings.ReplaceAll(snip, ">>", "\033[0m")
		fmt.Printf("     %s\n", snip)
	}
}
