package documentengine

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bsmart/abis/internal/docspec"
)

func TestClient_Health_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(HealthResponse{
			Status:          "ok",
			Service:         "document-engine",
			Version:         "1.0.0",
			TemplatesLoaded: 1,
			TemplateKeys:    []string{"test"},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", 5*time.Second)
	resp, err := client.Health(context.Background())
	if err != nil {
		t.Fatalf("Health() error = %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("status = %q, want %q", resp.Status, "ok")
	}
	if resp.TemplatesLoaded != 1 {
		t.Errorf("templates_loaded = %d, want 1", resp.TemplatesLoaded)
	}
}

func TestClient_Health_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", 5*time.Second)
	_, err := client.Health(context.Background())
	if err == nil {
		t.Error("Health() should fail on server error")
	}
}

func TestClient_Generate_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/generate" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.Header.Get("X-Internal-Secret") != "test-secret" {
			t.Error("missing or wrong X-Internal-Secret header")
		}

		var spec docspec.DocumentSpec
		if err := json.NewDecoder(r.Body).Decode(&spec); err != nil {
			t.Errorf("decode spec: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(GenerateResponse{
			Status: "generated",
			SpecID: spec.SpecID,
			RunID:  spec.RunID,
			TemplateUsed: TemplateInfo{
				Key:     spec.TemplateKey,
				Version: spec.TemplateVersion,
			},
			Files: []FileOutput{
				{
					Format:     "docx",
					Filename:   "test.docx",
					SizeBytes:  1024,
					SHA256:     "abc123",
					DataBase64: "UEsDBBQ=",
				},
			},
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-secret", 5*time.Second)
	spec := &docspec.DocumentSpec{
		SpecID:          "550e8400-e29b-41d4-a716-446655440000",
		RunID:           "550e8400-e29b-41d4-a716-446655440001",
		TaskID:          "550e8400-e29b-41d4-a716-446655440002",
		DocumentType:    "test_doc",
		TemplateKey:     "test_doc",
		TemplateVersion: "1.0",
		Output: docspec.DocumentOutput{
			Formats: []string{"docx"},
		},
	}

	resp, err := client.Generate(context.Background(), spec)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if resp.Status != "generated" {
		t.Errorf("status = %q, want %q", resp.Status, "generated")
	}
	if len(resp.Files) != 1 {
		t.Fatalf("len(files) = %d, want 1", len(resp.Files))
	}
	if resp.Files[0].Format != "docx" {
		t.Errorf("files[0].format = %q, want %q", resp.Files[0].Format, "docx")
	}
}

func TestClient_Generate_InvalidSpec(t *testing.T) {
	client := NewClient("http://localhost:9999", "secret", 5*time.Second)
	spec := &docspec.DocumentSpec{
		SpecID: "invalid-uuid",
	}

	_, err := client.Generate(context.Background(), spec)
	if err == nil {
		t.Error("Generate() should fail validation")
	}
}

func TestClient_Generate_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(ErrorDetail{
			Status:    "error",
			ErrorCode: "GENERATION_FAILED",
			Message:   "Template rendering failed",
		})
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", 5*time.Second)
	spec := &docspec.DocumentSpec{
		SpecID:          "550e8400-e29b-41d4-a716-446655440000",
		RunID:           "550e8400-e29b-41d4-a716-446655440001",
		TaskID:          "550e8400-e29b-41d4-a716-446655440002",
		DocumentType:    "test",
		TemplateKey:     "test",
		TemplateVersion: "1.0",
		Output: docspec.DocumentOutput{
			Formats: []string{"docx"},
		},
	}

	_, err := client.Generate(context.Background(), spec)
	if err == nil {
		t.Fatal("Generate() should fail on server error")
	}

	engineErr, ok := IsEngineError(err)
	if !ok {
		t.Fatalf("expected EngineError, got %T", err)
	}
	if engineErr.Code != "GENERATION_FAILED" {
		t.Errorf("error code = %q, want %q", engineErr.Code, "GENERATION_FAILED")
	}
}

func TestClient_Generate_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "secret", 100*time.Millisecond)
	spec := &docspec.DocumentSpec{
		SpecID:          "550e8400-e29b-41d4-a716-446655440000",
		RunID:           "550e8400-e29b-41d4-a716-446655440001",
		TaskID:          "550e8400-e29b-41d4-a716-446655440002",
		DocumentType:    "test",
		TemplateKey:     "test",
		TemplateVersion: "1.0",
		Output: docspec.DocumentOutput{
			Formats: []string{"docx"},
		},
	}

	_, err := client.Generate(context.Background(), spec)
	if err == nil {
		t.Error("Generate() should fail on timeout")
	}
}

func TestClient_Generate_ConnectionRefused(t *testing.T) {
	client := NewClient("http://127.0.0.1:9999", "secret", 1*time.Second)
	spec := &docspec.DocumentSpec{
		SpecID:          "550e8400-e29b-41d4-a716-446655440000",
		RunID:           "550e8400-e29b-41d4-a716-446655440001",
		TaskID:          "550e8400-e29b-41d4-a716-446655440002",
		DocumentType:    "test",
		TemplateKey:     "test",
		TemplateVersion: "1.0",
		Output: docspec.DocumentOutput{
			Formats: []string{"docx"},
		},
	}

	_, err := client.Generate(context.Background(), spec)
	if err == nil {
		t.Error("Generate() should fail on connection refused")
	}
}

func TestIsEngineError(t *testing.T) {
	err := &EngineError{
		HTTPStatus: 500,
		Code:       "TEST_ERROR",
		Message:    "test message",
	}

	engineErr, ok := IsEngineError(err)
	if !ok {
		t.Error("IsEngineError() should return true for EngineError")
	}
	if engineErr.Code != "TEST_ERROR" {
		t.Errorf("code = %q, want %q", engineErr.Code, "TEST_ERROR")
	}

	_, ok = IsEngineError(nil)
	if ok {
		t.Error("IsEngineError(nil) should return false")
	}
}
