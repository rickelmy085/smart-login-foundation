package rules

import (
	"context"
	"testing"
	"time"

	"github.com/bsmart/abis/internal/models"
)

func TestRuleEngine_BasicOperators(t *testing.T) {
	engine := NewRuleEngine([]RuleDefinition{
		{
			ID:          "test-equals",
			Name:        "Test Equals",
			Field:       "status",
			Operator:    OperatorEquals,
			Value:       "active",
			ValueType:   ValueTypeString,
			Status:      StatusPass,
			Active:      true,
			CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		},
		{
			ID:          "test-greater",
			Name:        "Test Greater Than",
			Field:       "valor",
			Operator:    OperatorGreaterThan,
			Value:       "100",
			ValueType:   ValueTypeNumber,
			Status:      StatusPass,
			Active:      true,
			CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		},
		{
			ID:          "test-contains",
			Name:        "Test Contains",
			Field:       "observacoes",
			Operator:    OperatorContains,
			Value:       "urgente",
			ValueType:   ValueTypeString,
			Status:      StatusPass,
			Active:      true,
			CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		},
	})

	ctxData := EvaluationContext{
		TaskID: "test-task",
		TaskData: map[string]string{
			"status":      "active",
			"valor":       "200",
			"observacoes": "Este é um caso urgente",
		},
		Requirements: []models.WorkflowRequirement{},
	}

	results := engine.Evaluate(context.Background(), ctxData)

	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	for _, r := range results {
		if r.Status != StatusPass {
			t.Errorf("rule %s expected PASS, got %s", r.RuleName, r.Status)
		}
	}
}

func TestRuleEngine_MissingRequiredField(t *testing.T) {
	engine := NewRuleEngine([]RuleDefinition{
		{
			ID:          "test-required",
			Name:        "Required Field",
			Field:       "valor",
			Operator:    OperatorExists,
			Value:       "",
			ValueType:   ValueTypeString,
			Status:      StatusPass,
			Active:      true,
			CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		},
	})

	ctxData := EvaluationContext{
		TaskID: "test-task",
		TaskData: map[string]string{
			"fornecedor": "Test",
		},
		Requirements: []models.WorkflowRequirement{
			{Name: "valor", Required: true},
		},
	}

	results := engine.Evaluate(context.Background(), ctxData)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Status != StatusInsufficientEvidence {
		t.Errorf("expected INSUFFICIENT_EVIDENCE, got %s", results[0].Status)
	}

	if !results[0].Input.Missing {
		t.Error("expected input to be marked as missing")
	}

	if !results[0].Input.Required {
		t.Error("expected input to be marked as required")
	}
}

func TestRuleEngine_NotApplicable(t *testing.T) {
	engine := NewRuleEngine([]RuleDefinition{
		{
			ID:          "test-optional",
			Name:        "Optional Field",
			Field:       "observacoes",
			Operator:    OperatorExists,
			Value:       "",
			ValueType:   ValueTypeString,
			Status:      StatusPass,
			Active:      true,
			CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		},
	})

	ctxData := EvaluationContext{
		TaskID: "test-task",
		TaskData: map[string]string{
			"valor": "100",
		},
		Requirements: []models.WorkflowRequirement{
			{Name: "valor", Required: true},
		},
	}

	results := engine.Evaluate(context.Background(), ctxData)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Status != StatusNotApplicable {
		t.Errorf("expected NOT_APPLICABLE, got %s", results[0].Status)
	}
}

func TestRuleEngine_NumberComparison(t *testing.T) {
	engine := NewRuleEngine([]RuleDefinition{
		{
			ID:          "test-gt",
			Name:        "Greater Than 100",
			Field:       "valor",
			Operator:    OperatorGreaterThan,
			Value:       "100",
			ValueType:   ValueTypeNumber,
			Status:      StatusPass,
			Active:      true,
			CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		},
		{
			ID:          "test-lte",
			Name:        "Less Than or Equal 500",
			Field:       "valor",
			Operator:    OperatorLessOrEqual,
			Value:       "500",
			ValueType:   ValueTypeNumber,
			Status:      StatusPass,
			Active:      true,
			CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		},
	})

	ctxData := EvaluationContext{
		TaskID: "test-task",
		TaskData: map[string]string{
			"valor": "250",
		},
		Requirements: []models.WorkflowRequirement{},
	}

	results := engine.Evaluate(context.Background(), ctxData)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	for _, r := range results {
		if r.Status != StatusPass {
			t.Errorf("rule %s expected PASS, got %s", r.RuleName, r.Status)
		}
	}
}

func TestRuleEngine_ContainsOperator(t *testing.T) {
	engine := NewRuleEngine([]RuleDefinition{
		{
			ID:          "test-contains",
			Name:        "Contains Urgente",
			Field:       "observacoes",
			Operator:    OperatorContains,
			Value:       "urgente",
			ValueType:   ValueTypeString,
			Status:      StatusPass,
			Active:      true,
			CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		},
		{
			ID:          "test-not-contains",
			Name:        "Not Contains Cancelado",
			Field:       "observacoes",
			Operator:    OperatorNotContains,
			Value:       "cancelado",
			ValueType:   ValueTypeString,
			Status:      StatusPass,
			Active:      true,
			CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		},
	})

	ctxData := EvaluationContext{
		TaskID: "test-task",
		TaskData: map[string]string{
			"observacoes": "Este pedido é urgente e importante",
		},
		Requirements: []models.WorkflowRequirement{},
	}

	results := engine.Evaluate(context.Background(), ctxData)

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	for _, r := range results {
		if r.Status != StatusPass {
			t.Errorf("rule %s expected PASS, got %s", r.RuleName, r.Status)
		}
	}
}

func TestRuleEngine_CurrencyParsing(t *testing.T) {
	engine := NewRuleEngine([]RuleDefinition{
		{
			ID:          "test-currency-gt",
			Name:        "Value Greater Than 10000",
			Field:       "valor",
			Operator:    OperatorGreaterThan,
			Value:       "10000",
			ValueType:   ValueTypeNumber,
			Status:      StatusPass,
			Active:      true,
			CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		},
	})

	ctxData := EvaluationContext{
		TaskID: "test-task",
		TaskData: map[string]string{
			"valor": "R$ 15.000,00",
		},
		Requirements: []models.WorkflowRequirement{},
	}

	results := engine.Evaluate(context.Background(), ctxData)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Status != StatusPass {
		t.Errorf("expected PASS for R$ 15.000,00 > 10000, got %s", results[0].Status)
	}
}

func TestRuleEngine_InOperator(t *testing.T) {
	engine := NewRuleEngine([]RuleDefinition{
		{
			ID:          "test-in",
			Name:        "Category In Allowed",
			Field:       "categoria",
			Operator:    OperatorIn,
			Value:       "TI, RH, Financeiro",
			ValueType:   ValueTypeString,
			Status:      StatusPass,
			Active:      true,
			CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		},
	})

	ctxData := EvaluationContext{
		TaskID: "test-task",
		TaskData: map[string]string{
			"categoria": "TI",
		},
		Requirements: []models.WorkflowRequirement{},
	}

	results := engine.Evaluate(context.Background(), ctxData)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if results[0].Status != StatusPass {
		t.Errorf("expected PASS for TI in allowed list, got %s", results[0].Status)
	}
}

func TestRuleRegistry(t *testing.T) {
	registry := NewRuleRegistry()

	rule := RuleDefinition{
		ID:        "test-1",
		Name:      "Test Rule",
		Field:     "test",
		Operator:  OperatorEquals,
		Value:     "value",
		ValueType: ValueTypeString,
		Status:    StatusPass,
		Active:    true,
	}

	err := registry.Register(rule)
	if err != nil {
		t.Fatalf("failed to register rule: %v", err)
	}

	// Try to register duplicate
	err = registry.Register(rule)
	if err == nil {
		t.Error("expected error for duplicate rule ID")
	}

	retrieved, ok := registry.Get("test-1")
	if !ok {
		t.Error("rule not found in registry")
	}
	if retrieved.Name != "Test Rule" {
		t.Error("retrieved rule has wrong name")
	}

	active := registry.ListActive()
	if len(active) != 1 {
		t.Errorf("expected 1 active rule, got %d", len(active))
	}
}

func TestSummarize(t *testing.T) {
	results := []RuleEvaluationResult{
		{RuleID: "1", Status: StatusPass},
		{RuleID: "2", Status: StatusPass},
		{RuleID: "3", Status: StatusFail},
		{RuleID: "4", Status: StatusNeedsReview},
		{RuleID: "5", Status: StatusInsufficientEvidence},
		{RuleID: "6", Status: StatusNotApplicable},
	}

	summary := Summarize("test-task", results)

	if summary.PassCount != 2 {
		t.Errorf("expected 2 PASS, got %d", summary.PassCount)
	}
	if summary.FailCount != 1 {
		t.Errorf("expected 1 FAIL, got %d", summary.FailCount)
	}
	if summary.NeedsReviewCount != 1 {
		t.Errorf("expected 1 NEEDS_REVIEW, got %d", summary.NeedsReviewCount)
	}
	if summary.InsufficientCount != 1 {
		t.Errorf("expected 1 INSUFFICIENT_EVIDENCE, got %d", summary.InsufficientCount)
	}
	if summary.NotApplicableCount != 1 {
		t.Errorf("expected 1 NOT_APPLICABLE, got %d", summary.NotApplicableCount)
	}

	// Overall status should be FAIL (highest priority)
	if summary.OverallStatus != StatusFail {
		t.Errorf("expected overall FAIL, got %s", summary.OverallStatus)
	}

	// Test with only PASS
	passOnly := []RuleEvaluationResult{
		{RuleID: "1", Status: StatusPass},
		{RuleID: "2", Status: StatusPass},
	}
	summary2 := Summarize("test-task", passOnly)
	if summary2.OverallStatus != StatusPass {
		t.Errorf("expected overall PASS for all pass, got %s", summary2.OverallStatus)
	}
}