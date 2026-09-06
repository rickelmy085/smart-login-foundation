package database

import (
	"context"
	"database/sql"
	"fmt" // debug prints
	"log/slog"
)

// MigrateWorkflow cria as tabelas do sistema de workflow e geração documental.
// Idempotente: CREATE TABLE IF NOT EXISTS.
func MigrateWorkflow(ctx context.Context, db *sql.DB) error {
	fmt.Println("[DB] MigrateWorkflow iniciada")
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS document_templates (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			document_type TEXT NOT NULL,
			version TEXT NOT NULL DEFAULT '1.0',
			active INTEGER NOT NULL DEFAULT 1,
			template_path TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);`,

		`CREATE TABLE IF NOT EXISTS template_fields (
			id TEXT PRIMARY KEY,
			template_id TEXT NOT NULL REFERENCES document_templates(id) ON DELETE CASCADE,
			field_name TEXT NOT NULL,
			label TEXT NOT NULL,
			type TEXT NOT NULL DEFAULT 'text',
			required INTEGER NOT NULL DEFAULT 0,
			validation_rule TEXT,
			source_requirement TEXT,
			normative_document TEXT,
			normative_chunk_id INTEGER REFERENCES chunks(id) ON DELETE SET NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);`,

		`CREATE TABLE IF NOT EXISTS workflow_tasks (
			id TEXT PRIMARY KEY,
			employee_id TEXT NOT NULL REFERENCES employees(id),
			intent TEXT NOT NULL,
			procedure TEXT,
			status TEXT NOT NULL DEFAULT 'detected',
			original_request TEXT NOT NULL,
			template_id TEXT REFERENCES document_templates(id),
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);`,

		`CREATE TABLE IF NOT EXISTS workflow_requirements (
			id TEXT PRIMARY KEY,
			task_id TEXT NOT NULL REFERENCES workflow_tasks(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			label TEXT,
			required INTEGER NOT NULL DEFAULT 1,
			source_document TEXT,
			source_chunk_id INTEGER REFERENCES chunks(id) ON DELETE SET NULL,
			source_snippet TEXT,
			rank INTEGER,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);`,

		`CREATE TABLE IF NOT EXISTS workflow_data (
			id TEXT PRIMARY KEY,
			task_id TEXT NOT NULL REFERENCES workflow_tasks(id) ON DELETE CASCADE,
			field_name TEXT NOT NULL,
			value TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);`,

		`CREATE TABLE IF NOT EXISTS document_runs (
			id TEXT PRIMARY KEY,
			task_id TEXT NOT NULL REFERENCES workflow_tasks(id),
			employee_id TEXT NOT NULL REFERENCES employees(id),
			template_id TEXT NOT NULL REFERENCES document_templates(id),
			template_version TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'generating',
			docx_path TEXT,
			pdf_path TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);`,

		`CREATE TABLE IF NOT EXISTS document_sources (
			id TEXT PRIMARY KEY,
			document_run_id TEXT NOT NULL REFERENCES document_runs(id) ON DELETE CASCADE,
			normative_document TEXT NOT NULL,
			normative_chunk_id INTEGER,
			normative_snippet TEXT,
			requirement TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);`,

		`CREATE INDEX IF NOT EXISTS idx_workflow_tasks_employee ON workflow_tasks(employee_id);`,
		`CREATE INDEX IF NOT EXISTS idx_workflow_tasks_status ON workflow_tasks(status);`,
		`CREATE INDEX IF NOT EXISTS idx_workflow_data_task ON workflow_data(task_id);`,
		`CREATE INDEX IF NOT EXISTS idx_document_runs_task ON document_runs(task_id);`,
		`CREATE INDEX IF NOT EXISTS idx_document_sources_run ON document_sources(document_run_id);`,
		`CREATE INDEX IF NOT EXISTS idx_template_fields_template ON template_fields(template_id);`,

		`DROP VIEW IF EXISTS history_view;`,
		`CREATE VIEW history_view AS
		SELECT
			cm.id AS id,
			'chat' AS type,
			SUBSTR(cm.content, 1, 80) || CASE WHEN LENGTH(cm.content) > 80 THEN '…' ELSE '' END AS title,
			cm.content AS preview,
			'completed' AS status,
			cm.created_at AS created_at,
			cm.employee_id AS employee_id,
			cm.sources_count AS sources
		FROM chat_messages cm
		WHERE cm.role = 'user'
		UNION ALL
		SELECT
			wt.id AS id,
			'task' AS type,
			wt.original_request AS title,
			COALESCE(wt.procedure, wt.original_request) AS preview,
			wt.status AS status,
			wt.created_at AS created_at,
			wt.employee_id AS employee_id,
			(SELECT COUNT(*) FROM document_sources ds JOIN document_runs dr ON ds.document_run_id = dr.id WHERE dr.task_id = wt.id) AS sources
		FROM workflow_tasks wt
		UNION ALL
		SELECT
			dr.id AS id,
			'document' AS type,
			dt.name AS title,
			wt.original_request AS preview,
			dr.status AS status,
			dr.created_at AS created_at,
			dr.employee_id AS employee_id,
			(SELECT COUNT(*) FROM document_sources ds WHERE ds.document_run_id = dr.id) AS sources
		FROM document_runs dr
		JOIN workflow_tasks wt ON wt.id = dr.task_id
		JOIN document_templates dt ON dt.id = dr.template_id;`,
	}
	stmts = append(stmts, []string{
		`CREATE TABLE IF NOT EXISTS chat_messages (
			id TEXT PRIMARY KEY,
			employee_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			allow_web_search INTEGER NOT NULL DEFAULT 0,
			sources_count INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		);`,
		`CREATE INDEX IF NOT EXISTS idx_chat_messages_employee ON chat_messages(employee_id);`,
		`CREATE INDEX IF NOT EXISTS idx_chat_messages_created ON chat_messages(created_at);`,
	}...)

	for i, stmt := range stmts {
		fmt.Printf("[DB] MigrateWorkflow executando stmt %d/%d\n", i+1, len(stmts))
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			fmt.Printf("[DB] MigrateWorkflow erro stmt %d: %v\n", i+1, err)
			return err
		}
	}

	slog.Info("workflow schema ready")
	fmt.Println("[DB] MigrateWorkflow schema pronto")
	return nil
}
