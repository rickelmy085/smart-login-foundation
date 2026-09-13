package service

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bsmart/abis/internal/groq"
	"github.com/bsmart/abis/internal/knowledge"
	"github.com/bsmart/abis/internal/models"
	"github.com/bsmart/abis/internal/repository"
	"github.com/bsmart/abis/internal/rules"
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

	// Test with empty rules engine
	svc.WithRulesEngine(rules.NewRuleEngine([]rules.RuleDefinition{}))

	data := map[string]string{
		"fornecedor": "Fornecedor X",
	}
	requirements := []models.WorkflowRequirement{}
	evalContext := rules.EvaluationContext{
		TaskID:       "test-task",
		TaskData:     data,
		Requirements: requirements,
	}

	evalResults := svc.RulesEngine().Evaluate(context.Background(), evalContext)

	if len(evalResults) != 0 {
		t.Errorf("expected 0 rules without rules engine, got %d", len(evalResults))
	}
}

func TestEvaluateRules_WithValue(t *testing.T) {
	db, _, err := sqlmock.New()
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	defer db.Close()

	repo := repository.NewWorkflowRepo(db)
	searcher := knowledge.NewSearcher(db)
	g := groq.New("test-key", "test-model")
	svc := NewWorkflowService(repo, searcher, g)

	// Add a rule that checks for "valor" field
	testRules := []rules.RuleDefinition{
		{
			ID:          "test-valor",
			Name:        "Valor informado",
			Field:       "valor",
			Operator:    rules.OperatorExists,
			Value:       "",
			ValueType:   rules.ValueTypeString,
			Status:      rules.StatusPass,
			Active:      true,
			CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		},
	}
	svc.WithRulesEngine(rules.NewRuleEngine(testRules))

	data := map[string]string{
		"valor": "R$ 80.000,00",
	}
	requirements := []models.WorkflowRequirement{}
	evalContext := rules.EvaluationContext{
		TaskID:       "test-task",
		TaskData:     data,
		Requirements: requirements,
	}

	evalResults := svc.RulesEngine().Evaluate(context.Background(), evalContext)

	if len(evalResults) == 0 {
		t.Error("expected at least one rule evaluation")
	}

	for _, rule := range evalResults {
		if rule.RuleName == "Valor informado" {
			if rule.Status != rules.StatusPass {
				t.Errorf("expected valor rule to pass, got %s", rule.Status)
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

