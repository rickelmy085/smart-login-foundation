// Continuação do package database: tabelas da camada de conhecimento.
//
// Mantemos as migrações de knowledge separadas das de auth para não
// acoplar o ciclo de vida das duas camadas (e facilitar evolução
// independente rumo a RAG/embeddings).
package database

import (
	"context"
	"database/sql"
	"fmt" // debug prints
	"log/slog"
)

// MigrateKnowledge cria as tabelas e o índice FTS5 usados pelo pipeline
// de ingestão de PDFs. Idempotente como Migrate principal.
//
// Esquema:
//
//   documents          → 1 linha por PDF (id, source_path, hash, ...).
//   chunks             → N linhas por documento (trechos de texto).
//   chunks_fts         → Virtual table FTS5 sincronizada com chunks.content
//                        via triggers; habilita busca lexical com BM25.
//
// O conteúdo em chunks_fts é a forma "normalizada" do texto:
// mesmo conteúdo que está em chunks.content. Manter a coluna `content`
// na tabela regular ajuda a devolver o trecho exato ao usuário,
// enquanto o FTS é só pra rankeamento.
func MigrateKnowledge(ctx context.Context, db *sql.DB) error {
	fmt.Println("[DB] MigrateKnowledge iniciada")
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS documents (
			id TEXT PRIMARY KEY,
			source_path TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			page_count INTEGER NOT NULL DEFAULT 0,
			char_count INTEGER NOT NULL DEFAULT 0,
			hash TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'ok',
			error TEXT,
			ingested_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
	`); err != nil {
		fmt.Printf("[DB] MigrateKnowledge erro create documents: %v\n", err)
		return err
	}
	fmt.Println("[DB] MigrateKnowledge tabela documents criada/ok")

	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS chunks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			document_id TEXT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
			ord INTEGER NOT NULL,
			content TEXT NOT NULL,
			char_start INTEGER NOT NULL,
			char_end INTEGER NOT NULL,
			UNIQUE(document_id, ord)
		);
	`); err != nil {
		fmt.Printf("[DB] MigrateKnowledge erro create chunks: %v\n", err)
		return err
	}
	fmt.Println("[DB] MigrateKnowledge tabela chunks criada/ok")

	if _, err := db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_chunks_document_id ON chunks(document_id);`); err != nil {
		fmt.Printf("[DB] MigrateKnowledge erro create index: %v\n", err)
		return err
	}
	fmt.Println("[DB] MigrateKnowledge indice idx_chunks_document_id criado/ok")

	// FTS5: "external content" usa chunks.content como source-of-truth,
	// evitando duplicar texto. tokenize='unicode61 remove_diacritics 2'
	// faz busca tolerante a acentos (RegrAS ≈ Regras).
	if _, err := db.ExecContext(ctx, `
		CREATE VIRTUAL TABLE IF NOT EXISTS chunks_fts USING fts5(
			content,
			content='chunks',
			content_rowid='id',
			tokenize='unicode61 remove_diacritics 2'
		);
	`); err != nil {
		fmt.Printf("[DB] MigrateKnowledge erro create chunks_fts: %v\n", err)
		return err
	}
	fmt.Println("[DB] MigrateKnowledge tabela chunks_fts criada/ok")

	// Triggers mantêm chunks_fts em sincronia com chunks.
	// Sem eles, o FTS ficaria congelado após o INSERT inicial.
	for i, stmt := range []string{
		`CREATE TRIGGER IF NOT EXISTS chunks_ai AFTER INSERT ON chunks BEGIN
			INSERT INTO chunks_fts(rowid, content) VALUES (new.id, new.content);
		END;`,
		`CREATE TRIGGER IF NOT EXISTS chunks_ad AFTER DELETE ON chunks BEGIN
			INSERT INTO chunks_fts(chunks_fts, rowid, content) VALUES('delete', old.id, old.content);
		END;`,
		`CREATE TRIGGER IF NOT EXISTS chunks_au AFTER UPDATE ON chunks BEGIN
			INSERT INTO chunks_fts(chunks_fts, rowid, content) VALUES('delete', old.id, old.content);
			INSERT INTO chunks_fts(rowid, content) VALUES (new.id, new.content);
		END;`,
	} {
		fmt.Printf("[DB] MigrateKnowledge criando trigger %d/3\n", i+1)
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			fmt.Printf("[DB] MigrateKnowledge erro trigger %d: %v\n", i+1, err)
			return err
		}
	}
	fmt.Println("[DB] MigrateKnowledge triggers criados/ok")

	slog.Info("knowledge schema ready")
	fmt.Println("[DB] MigrateKnowledge schema pronto")
	return nil
}
