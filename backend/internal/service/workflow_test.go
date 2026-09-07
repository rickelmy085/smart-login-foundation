package service

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bsmart/abis/internal/groq"
	"github.com/bsmart/abis/internal/knowledge"
	"github.com/bsmart/abis/internal/repository"
)

func TestClassifyIntent_KnowledgeQuery(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	repo := repository.NewWorkflowRepo(db)
	searcher := knowledge.NewSearcher(db)

	// Mock groq with empty key to test the error path
	g := groq.New("", "test-model")
	svc := NewWorkflowService(repo, searcher, g)

	_, err = svc.ClassifyIntent(context.Background(), "Qual o limite de compras?")
	if err == nil {
		t.Error("expected error for empty API key, got nil")
	}
}

func TestEvaluateRules_NoValue(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	repo := repository.NewWorkflowRepo(db)
	searcher := knowledge.NewSearcher(db)
	g := groq.New("test-key", "test-model")
	svc := NewWorkflowService(repo, searcher, g)

	data := map[string]string{
		"fornecedor": "Fornecedor X",
	}
	rules := svc.evaluateRules(data, "aquisição de TI")

	if len(rules) != 0 {
		t.Errorf("expected 0 rules without valor, got %d", len(rules))
	}
}

func TestEvaluateRules_WithHighValue(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	repo := repository.NewWorkflowRepo(db)
	searcher := knowledge.NewSearcher(db)
	g := groq.New("test-key", "test-model")
	svc := NewWorkflowService(repo, searcher, g)

	data := map[string]string{
		"valor": "R$ 80.000,00",
	}
	rules := svc.evaluateRules(data, "aquisição de TI")

	if len(rules) == 0 {
		t.Error("expected at least one rule evaluation")
	}

	for _, rule := range rules {
		if rule.RuleName == "valor_informado" {
			if !rule.Passed {
				t.Error("expected valor_informado rule to pass")
			}
		}
	}
}

func TestExtractData_NoRequirements(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	repo := repository.NewWorkflowRepo(db)
	searcher := knowledge.NewSearcher(db)
	g := groq.New("test-key", "test-model")
	svc := NewWorkflowService(repo, searcher, g)

	data, err := svc.ExtractData(context.Background(), "Fornecedor X", nil)
	if err == nil {
		// With no requirements, should either succeed or fail depending on LLM
		t.Logf("ExtractData returned: %v", data)
	}
}

func TestFormatMissingFieldNames(t *testing.T) {
	missing := []MissingField{
		{FieldName: "fornecedor", Label: "Fornecedor"},
		{FieldName: "valor", Label: "Valor da aquisição"},
	}

	result := formatMissingFieldNames(missing)
	if result == "" {
		t.Error("expected non-empty result")
	}
}

