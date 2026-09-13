package repository

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/bsmart/abis/internal/models"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	// Create schema
	_, err = db.Exec(`
		CREATE TABLE document_templates (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			document_type TEXT NOT NULL,
			template_key TEXT NOT NULL DEFAULT '',
			version TEXT NOT NULL DEFAULT '1.0',
			active INTEGER NOT NULL DEFAULT 1,
			template_path TEXT NOT NULL,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
		CREATE UNIQUE INDEX idx_document_templates_key_version 
		ON document_templates(template_key, version);
	`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	return db
}

func TestGetTemplateByKey_ExactMatch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewWorkflowRepo(db)
	ctx := context.Background()

	// Insert test templates
	_, err := db.ExecContext(ctx, `
		INSERT INTO document_templates (id, name, description, document_type, template_key, version, active, template_path)
		VALUES 
			('tmpl-1', 'Solicitação de Aquisição', 'Doc type 1', 'solicitacao', 'solicitacao_aquisicao_ti', '1.0', 1, '/path/1'),
			('tmpl-2', 'Memorando Interno', 'Doc type 2', 'memorando', 'memorando_interno', '1.0', 1, '/path/2'),
			('tmpl-3', 'Relatório Operacional', 'Doc type 3', 'relatorio', 'relatorio_operacional', '1.0', 1, '/path/3')
	`)
	if err != nil {
		t.Fatalf("insert templates: %v", err)
	}

	tests := []struct {
		name        string
		key         string
		wantFound   bool
		wantID      string
		wantKey     string
	}{
		{
			name:      "exact match solicitacao",
			key:       "solicitacao_aquisicao_ti",
			wantFound: true,
			wantID:    "tmpl-1",
			wantKey:   "solicitacao_aquisicao_ti",
		},
		{
			name:      "exact match memorando",
			key:       "memorando_interno",
			wantFound: true,
			wantID:    "tmpl-2",
			wantKey:   "memorando_interno",
		},
		{
			name:      "exact match relatorio",
			key:       "relatorio_operacional",
			wantFound: true,
			wantID:    "tmpl-3",
			wantKey:   "relatorio_operacional",
		},
		{
			name:      "no partial match - memorando alone",
			key:       "memorando",
			wantFound: false,
		},
		{
			name:      "no partial match - aquisicao alone",
			key:       "aquisicao",
			wantFound: false,
		},
		{
			name:      "nonexistent key",
			key:       "template_inexistente",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template, err := repo.GetTemplateByKey(ctx, tt.key)
			if tt.wantFound {
				if err != nil {
					t.Errorf("GetTemplateByKey(%q) error = %v, want nil", tt.key, err)
					return
				}
				if template.ID != tt.wantID {
					t.Errorf("GetTemplateByKey(%q) ID = %q, want %q", tt.key, template.ID, tt.wantID)
				}
				if template.TemplateKey != tt.wantKey {
					t.Errorf("GetTemplateByKey(%q) TemplateKey = %q, want %q", tt.key, template.TemplateKey, tt.wantKey)
				}
			} else {
				if err != sql.ErrNoRows {
					t.Errorf("GetTemplateByKey(%q) error = %v, want sql.ErrNoRows", tt.key, err)
				}
			}
		})
	}
}

func TestGetTemplateByKeyAndVersion(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewWorkflowRepo(db)
	ctx := context.Background()

	// Insert template with multiple versions
	_, err := db.ExecContext(ctx, `
		INSERT INTO document_templates (id, name, description, document_type, template_key, version, active, template_path)
		VALUES 
			('tmpl-v1', 'Memorando v1', 'Version 1', 'memorando', 'memorando_interno', '1.0', 1, '/path/v1'),
			('tmpl-v2', 'Memorando v2', 'Version 2', 'memorando', 'memorando_interno', '2.0', 1, '/path/v2')
	`)
	if err != nil {
		t.Fatalf("insert templates: %v", err)
	}

	tests := []struct {
		name      string
		key       string
		version   string
		wantFound bool
		wantID    string
	}{
		{
			name:      "find v1.0",
			key:       "memorando_interno",
			version:   "1.0",
			wantFound: true,
			wantID:    "tmpl-v1",
		},
		{
			name:      "find v2.0",
			key:       "memorando_interno",
			version:   "2.0",
			wantFound: true,
			wantID:    "tmpl-v2",
		},
		{
			name:      "version not found",
			key:       "memorando_interno",
			version:   "3.0",
			wantFound: false,
		},
		{
			name:      "key not found",
			key:       "unknown_key",
			version:   "1.0",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			template, err := repo.GetTemplateByKeyAndVersion(ctx, tt.key, tt.version)
			if tt.wantFound {
				if err != nil {
					t.Errorf("GetTemplateByKeyAndVersion(%q, %q) error = %v, want nil", tt.key, tt.version, err)
					return
				}
				if template.ID != tt.wantID {
					t.Errorf("GetTemplateByKeyAndVersion(%q, %q) ID = %q, want %q", tt.key, tt.version, template.ID, tt.wantID)
				}
			} else {
				if err != sql.ErrNoRows {
					t.Errorf("GetTemplateByKeyAndVersion(%q, %q) error = %v, want sql.ErrNoRows", tt.key, tt.version, err)
				}
			}
		})
	}
}

func TestListTemplateKeys(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewWorkflowRepo(db)
	ctx := context.Background()

	// Insert test templates
	_, err := db.ExecContext(ctx, `
		INSERT INTO document_templates (id, name, description, document_type, template_key, version, active, template_path)
		VALUES 
			('tmpl-1', 'Template 1', 'Desc 1', 'type1', 'solicitacao_aquisicao_ti', '1.0', 1, '/path/1'),
			('tmpl-2', 'Template 2', 'Desc 2', 'type2', 'memorando_interno', '1.0', 1, '/path/2'),
			('tmpl-3', 'Template 3', 'Desc 3', 'type3', 'relatorio_operacional', '1.0', 1, '/path/3'),
			('tmpl-4', 'Inactive Template', 'Desc 4', 'type4', 'inactive_template', '1.0', 0, '/path/4')
	`)
	if err != nil {
		t.Fatalf("insert templates: %v", err)
	}

	keys, err := repo.ListTemplateKeys(ctx)
	if err != nil {
		t.Fatalf("ListTemplateKeys() error = %v", err)
	}

	// Should return only active templates
	if len(keys) != 3 {
		t.Errorf("ListTemplateKeys() returned %d keys, want 3", len(keys))
	}

	expectedKeys := map[string]bool{
		"solicitacao_aquisicao_ti": false,
		"memorando_interno":        false,
		"relatorio_operacional":    false,
	}

	for _, k := range keys {
		if _, ok := expectedKeys[k]; ok {
			expectedKeys[k] = true
		} else {
			t.Errorf("ListTemplateKeys() returned unexpected key: %q", k)
		}
	}

	for k, found := range expectedKeys {
		if !found {
			t.Errorf("ListTemplateKeys() missing expected key: %q", k)
		}
	}
}

func TestGetTemplateByKey_NoLIKE(t *testing.T) {
	// This test specifically verifies that GetTemplateByKey does NOT use LIKE matching
	db := setupTestDB(t)
	defer db.Close()

	repo := NewWorkflowRepo(db)
	ctx := context.Background()

	// Insert templates with similar names
	_, err := db.ExecContext(ctx, `
		INSERT INTO document_templates (id, name, description, document_type, template_key, version, active, template_path)
		VALUES 
			('tmpl-1', 'Memorando Interno', 'Standard memo', 'memorando', 'memorando_interno', '1.0', 1, '/path/1'),
			('tmpl-2', 'Memorando Interno Anexo', 'Memo with attachment', 'memorando', 'memorando_interno_anexo', '1.0', 1, '/path/2')
	`)
	if err != nil {
		t.Fatalf("insert templates: %v", err)
	}

	// Request exact match for "memorando_interno"
	template, err := repo.GetTemplateByKey(ctx, "memorando_interno")
	if err != nil {
		t.Fatalf("GetTemplateByKey('memorando_interno') error = %v", err)
	}

	// Should return only the exact match, not the similar one
	if template.ID != "tmpl-1" {
		t.Errorf("GetTemplateByKey('memorando_interno') ID = %q, want 'tmpl-1'", template.ID)
	}
	if template.TemplateKey != "memorando_interno" {
		t.Errorf("GetTemplateByKey('memorando_interno') TemplateKey = %q, want 'memorando_interno'", template.TemplateKey)
	}

	// Verify it's NOT returning the similar template
	if template.ID == "tmpl-2" {
		t.Error("GetTemplateByKey('memorando_interno') incorrectly returned 'memorando_interno_anexo' template (LIKE behavior)")
	}
}

func TestCreateTask_WithTemplateKey(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Add workflow_tasks table
	_, err := db.Exec(`
		CREATE TABLE workflow_tasks (
			id TEXT PRIMARY KEY,
			employee_id TEXT NOT NULL,
			intent TEXT NOT NULL,
			procedure TEXT,
			status TEXT NOT NULL DEFAULT 'detected',
			original_request TEXT NOT NULL,
			template_id TEXT,
			template_key TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);
	`)
	if err != nil {
		t.Fatalf("create workflow_tasks: %v", err)
	}

	repo := NewWorkflowRepo(db)
	ctx := context.Background()

	task := models.WorkflowTask{
		ID:             "task-123",
		EmployeeID:     "emp-456",
		Intent:         models.IntentDocumentGeneration,
		Procedure:      "aquisição de notebooks",
		Status:         models.StatusDetected,
		OriginalRequest: "Preciso comprar 5 notebooks",
		TemplateKey:    "solicitacao_aquisicao_ti",
	}

	err = repo.CreateTask(ctx, task)
	if err != nil {
		t.Fatalf("CreateTask() error = %v", err)
	}

	// Retrieve and verify
	retrieved, err := repo.GetTask(ctx, "task-123")
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}

	if retrieved.TemplateKey != "solicitacao_aquisicao_ti" {
		t.Errorf("GetTask() TemplateKey = %q, want 'solicitacao_aquisicao_ti'", retrieved.TemplateKey)
	}
}
