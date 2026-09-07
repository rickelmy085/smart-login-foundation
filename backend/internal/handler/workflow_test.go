package handler

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/bsmart/abis/internal/knowledge"
	"github.com/bsmart/abis/internal/repository"
	"github.com/bsmart/abis/internal/service"
	"github.com/bsmart/abis/internal/groq"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}

	ctx := context.Background()
	if err := runMigrations(ctx, db); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	return db
}

func runMigrations(ctx context.Context, db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS employees (
			id TEXT PRIMARY KEY,
			re TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			role TEXT NOT NULL,
			email TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL,
			active INTEGER NOT NULL DEFAULT 1,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);`,
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
		`CREATE TABLE IF NOT EXISTS document_runs (
			id TEXT PRIMARY KEY,
			task_id TEXT NOT NULL,
			employee_id TEXT NOT NULL REFERENCES employees(id),
			template_id TEXT NOT NULL REFERENCES document_templates(id),
			template_version TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'generating',
			docx_path TEXT,
			pdf_path TEXT,
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			updated_at TEXT NOT NULL DEFAULT (datetime('now'))
		);`,
		`CREATE TABLE IF NOT EXISTS documents (
			id TEXT PRIMARY KEY,
			source_path TEXT NOT NULL UNIQUE,
			title TEXT NOT NULL,
			page_count INTEGER NOT NULL DEFAULT 0,
			char_count INTEGER NOT NULL DEFAULT 0,
			hash TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'ok',
			error TEXT,
			ingested_at TEXT NOT NULL DEFAULT (datetime('now'))
		);`,
		`CREATE TABLE IF NOT EXISTS chunks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			document_id TEXT NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
			ord INTEGER NOT NULL,
			content TEXT NOT NULL,
			char_start INTEGER NOT NULL,
			char_end INTEGER NOT NULL,
			UNIQUE(document_id, ord)
		);`,
	}
	for _, stmt := range stmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func seedTestData(t *testing.T, db *sql.DB, empID, runID, docxPath, pdfPath string) {
	t.Helper()
	ctx := context.Background()
	if _, err := db.ExecContext(ctx,
		`INSERT INTO employees (id, re, name, role, email, password) VALUES (?, '123', 'Test User', 'analyst', 'test@test.com', 'hash')`,
		empID); err != nil {
		t.Fatalf("failed to insert employee: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO document_templates (id, name, document_type, template_path) VALUES (?, 'test-template', 'test', '/test/path')`,
		"tmpl-1"); err != nil {
		t.Fatalf("failed to insert template: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO document_runs (id, task_id, employee_id, template_id, template_version, status, docx_path, pdf_path) VALUES (?, 'task-1', ?, 'tmpl-1', '1.0', 'completed', ?, ?)`,
		runID, empID, docxPath, pdfPath); err != nil {
		t.Fatalf("failed to insert document run: %v", err)
	}
}

func newTestHandler(t *testing.T, db *sql.DB, docDir string) *WorkflowHandler {
	t.Helper()
	repo := repository.NewWorkflowRepo(db)
	searcher := knowledge.NewSearcher(db)
	g := groq.New("", "test-model")
	svc := service.NewWorkflowService(repo, searcher, g)
	h := NewWorkflowHandler(svc, repo)
	h.documentsDir = docDir
	return h
}

func dispatchDownload(t *testing.T, h *WorkflowHandler, method, fullPath, empID string) *httptest.ResponseRecorder {
	t.Helper()
	router := http.NewServeMux()
	router.Handle("/api/documents/{id}/docx", http.HandlerFunc(h.DownloadDocx))
	router.Handle("/api/documents/{id}/pdf", http.HandlerFunc(h.DownloadPdf))

	req := httptest.NewRequest(method, fullPath, nil)
	ctx := req.Context()
	ctx = context.WithValue(ctx, "employee_id", empID)
	req = req.WithContext(ctx)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func TestDownloadDocx_NoAuth_Returns401(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	h := newTestHandler(t, db, "data/documents")

	rr := dispatchDownload(t, h, http.MethodGet, "/api/documents/run-1/docx", "")

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestDownloadDocx_DocumentNotInDB_Returns404(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	h := newTestHandler(t, db, "data/documents")

	rr := dispatchDownload(t, h, http.MethodGet, "/api/documents/nonexistent/docx", "emp-1")

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestDownloadDocx_OtherUserDocument_Returns404(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	tmpDir := t.TempDir()
	docxPath := filepath.Join(tmpDir, "document.docx")
	pdfPath := filepath.Join(tmpDir, "document.pdf")
	seedTestData(t, db, "emp-owner", "run-1", docxPath, pdfPath)
	h := newTestHandler(t, db, tmpDir)

	rr := dispatchDownload(t, h, http.MethodGet, "/api/documents/run-1/docx", "emp-other")

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unauthorized access, got %d", rr.Code)
	}
}

func TestDownloadDocx_ValidUser_DownloadsFile(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	empID, runID := "emp-1", "run-1"
	docDir := t.TempDir()
	docxPath := filepath.Join(docDir, "document.docx")
	if err := os.WriteFile(docxPath, []byte("fake-docx-content"), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	seedTestData(t, db, empID, runID, docxPath, docxPath)
	h := newTestHandler(t, db, docDir)

	rr := dispatchDownload(t, h, http.MethodGet, "/api/documents/run-1/docx", empID)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	ctype := rr.Header().Get("Content-Type")
	if ctype != "application/vnd.openxmlformats-officedocument.wordprocessingml.document" {
		t.Errorf("expected docx content type, got %s", ctype)
	}

	cd := rr.Header().Get("Content-Disposition")
	expected := `attachment; filename="ABIS-documento-run-1.docx"`
	if cd != expected {
		t.Errorf("expected Content-Disposition: %s, got %s", expected, cd)
	}

	if rr.Body.String() != "fake-docx-content" {
		t.Errorf("expected file content, got %s", rr.Body.String())
	}
}

func TestDownloadPdf_ValidUser_DownloadsFile(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	empID, runID := "emp-1", "run-1"
	docDir := t.TempDir()
	docxPath := filepath.Join(docDir, "document.docx")
	pdfPath := filepath.Join(docDir, "document.pdf")
	if err := os.WriteFile(docxPath, []byte("fake-docx"), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	if err := os.WriteFile(pdfPath, []byte("fake-pdf"), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	seedTestData(t, db, empID, runID, docxPath, pdfPath)
	h := newTestHandler(t, db, docDir)

	rr := dispatchDownload(t, h, http.MethodGet, "/api/documents/run-1/pdf", empID)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	cd := rr.Header().Get("Content-Disposition")
	expected := `attachment; filename="ABIS-documento-run-1.pdf"`
	if cd != expected {
		t.Errorf("expected Content-Disposition: %s, got %s", expected, cd)
	}

	ctype := rr.Header().Get("Content-Type")
	if ctype != "application/pdf" {
		t.Errorf("expected pdf content type, got %s", ctype)
	}

	if rr.Body.String() != "fake-pdf" {
		t.Errorf("expected pdf content, got %s", rr.Body.String())
	}
}

func TestDownloadPdf_UsesDerivedPath_WhenPdfPathEmpty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	empID, runID := "emp-1", "run-1"
	docDir := t.TempDir()
	docxPath := filepath.Join(docDir, "document.docx")
	pdfPath := filepath.Join(docDir, "document.pdf")
	if err := os.WriteFile(docxPath, []byte("fake-docx"), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	if err := os.WriteFile(pdfPath, []byte("fake-pdf"), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	seedTestData(t, db, empID, runID, docxPath, "")
	h := newTestHandler(t, db, docDir)

	rr := dispatchDownload(t, h, http.MethodGet, "/api/documents/run-1/pdf", empID)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
	if rr.Body.String() != "fake-pdf" {
		t.Errorf("expected pdf content, got %s", rr.Body.String())
	}
}

func TestValidateAndReadFile_RejectsPathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	if _, err := validateAndReadFile(filepath.Join(tmpDir, "..", "..", "etc", "passwd"), tmpDir); err == nil {
		t.Error("expected error for path traversal, got nil")
	}
}

func TestValidateAndReadFile_RejectsRelativeTraversal(t *testing.T) {
	if _, err := validateAndReadFile("../../../etc/passwd", "data/documents"); err == nil {
		t.Error("expected error for relative path traversal, got nil")
	}
}

func TestDownloadDocx_OutsideAllowedDirectory_ReturnsError(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	empID, runID := "emp-1", "run-1"
	seedTestData(t, db, empID, runID, "/tmp/evil.docx", "/tmp/evil.pdf")
	h := newTestHandler(t, db, "data/documents")

	rr := dispatchDownload(t, h, http.MethodGet, "/api/documents/run-1/docx", empID)

	if rr.Code == http.StatusOK {
		t.Error("expected error for path outside allowed directory, got 200")
	}
}
