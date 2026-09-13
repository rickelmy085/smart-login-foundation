package rules

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bsmart/abis/internal/models"
)

// RuleStatus represents the result of evaluating a rule.
type RuleStatus string

const (
	StatusPass               RuleStatus = "PASS"
	StatusFail               RuleStatus = "FAIL"
	StatusNeedsReview        RuleStatus = "NEEDS_REVIEW"
	StatusInsufficientEvidence RuleStatus = "INSUFFICIENT_EVIDENCE"
	StatusNotApplicable      RuleStatus = "NOT_APPLICABLE"
)

// RuleOperator represents the type of comparison operation.
type RuleOperator string

const (
	OperatorEquals            RuleOperator = "equals"
	OperatorNotEquals         RuleOperator = "not_equals"
	OperatorGreaterThan       RuleOperator = "greater_than"
	OperatorGreaterOrEqual    RuleOperator = "greater_or_equal"
	OperatorLessThan          RuleOperator = "less_than"
	OperatorLessOrEqual       RuleOperator = "less_or_equal"
	OperatorContains          RuleOperator = "contains"
	OperatorNotContains       RuleOperator = "not_contains"
	OperatorExists            RuleOperator = "exists"
	OperatorNotExists         RuleOperator = "not_exists"
	OperatorIn                RuleOperator = "in"
	OperatorNotIn             RuleOperator = "not_in"
)

// RuleValueType represents the expected type of a field value.
type RuleValueType string

const (
	ValueTypeString  RuleValueType = "string"
	ValueTypeNumber  RuleValueType = "number"
	ValueTypeBoolean RuleValueType = "boolean"
	ValueTypeDate    RuleValueType = "date"
)

// RuleSource represents the normative source backing a rule.
type RuleSource struct {
	Document       string `json:"document,omitempty"`
	ChunkID        *int64 `json:"chunk_id,omitempty"`
	Snippet        string `json:"snippet,omitempty"`
	Requirement    string `json:"requirement,omitempty"`
}

// RuleInput represents a field input for rule evaluation.
type RuleInput struct {
	FieldName   string       `json:"field_name"`
	Value       string       `json:"value"`
	ValueType   RuleValueType `json:"value_type"`
	Required    bool         `json:"required"`
	Missing     bool         `json:"missing"`
}

// RuleDefinition represents a single deterministic rule.
type RuleDefinition struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Field       string       `json:"field"`
	Operator    RuleOperator `json:"operator"`
	Value       string       `json:"value"`
	ValueType   RuleValueType `json:"value_type"`
	Status      RuleStatus   `json:"status_when_met"`
	Sources     []RuleSource `json:"sources"`
	Active      bool         `json:"active"`
	CreatedAt   string       `json:"created_at"`
}

// RuleEvaluationResult represents the result of evaluating a single rule.
type RuleEvaluationResult struct {
	RuleID        string       `json:"rule_id"`
	RuleName      string       `json:"rule_name"`
	Status        RuleStatus   `json:"status"`
	Message       string       `json:"message"`
	Input         RuleInput    `json:"input"`
	ExpectedValue string       `json:"expected_value,omitempty"`
	ActualValue   string       `json:"actual_value,omitempty"`
	Sources       []RuleSource `json:"sources"`
	EvaluatedAt   string       `json:"evaluated_at"`
}

// RuleEngine evaluates rules against input data.
type RuleEngine struct {
	rules []RuleDefinition
}

// NewRuleEngine creates a new rule engine with the given rules.
func NewRuleEngine(rules []RuleDefinition) *RuleEngine {
	return &RuleEngine{rules: rules}
}

// AddRule adds a rule to the engine.
func (e *RuleEngine) AddRule(rule RuleDefinition) {
	e.rules = append(e.rules, rule)
}

// GetRule retrieves a rule by ID.
func (e *RuleEngine) GetRule(id string) (RuleDefinition, bool) {
	for _, r := range e.rules {
		if r.ID == id {
			return r, true
		}
	}
	return RuleDefinition{}, false
}

// ListActiveRules returns all active rules.
func (e *RuleEngine) ListActiveRules() []RuleDefinition {
	var active []RuleDefinition
	for _, r := range e.rules {
		if r.Active {
			active = append(active, r)
		}
	}
	return active
}

// EvaluationContext holds the context for rule evaluation.
type EvaluationContext struct {
	TaskID       string
	TaskData     map[string]string
	Requirements []models.WorkflowRequirement
	TemplateKey  string
}

// Evaluate runs all applicable rules against the input data.
func (e *RuleEngine) Evaluate(ctx context.Context, ctxData EvaluationContext) []RuleEvaluationResult {
	var results []RuleEvaluationResult
	now := time.Now().UTC().Format(time.RFC3339)

	for _, rule := range e.rules {
		if !rule.Active {
			continue
		}

		// Check if rule applies to this template (if specified)
		// For now, we evaluate all active rules. Template filtering can be added later.

		result := e.evaluateRule(ctx, rule, ctxData, now)
		results = append(results, result)
	}

	return results
}

// evaluateRule evaluates a single rule.
func (e *RuleEngine) evaluateRule(ctx context.Context, rule RuleDefinition, ctxData EvaluationContext, evaluatedAt string) RuleEvaluationResult {
	// Build input
	input := RuleInput{
		FieldName: rule.Field,
		ValueType: rule.ValueType,
		Required:  false, // Will be set based on requirements
		Missing:   false,
	}

	// Find the value in task data
	value, exists := ctxData.TaskData[rule.Field]
	if !exists || value == "" {
		input.Missing = true
		input.Value = ""
	} else {
		input.Value = value
	}

	// Check if field is required based on requirements
	for _, req := range ctxData.Requirements {
		if req.Name == rule.Field && req.Required {
			input.Required = true
			break
		}
	}

	// Evaluate based on operator
	var status RuleStatus
	var message string

	if input.Missing {
		if input.Required {
			status = StatusInsufficientEvidence
			message = "Campo obrigatório não informado"
		} else {
			status = StatusNotApplicable
			message = "Campo não informado e não obrigatório"
		}
	} else {
		status, message = e.applyOperator(rule, input.Value)
	}

	// Build result
	result := RuleEvaluationResult{
		RuleID:        rule.ID,
		RuleName:      rule.Name,
		Status:        status,
		Message:       message,
		Input:         input,
		ExpectedValue: rule.Value,
		ActualValue:   input.Value,
		Sources:       rule.Sources,
		EvaluatedAt:   evaluatedAt,
	}

	return result
}

// applyOperator applies the rule operator to the input value.
func (e *RuleEngine) applyOperator(rule RuleDefinition, actualValue string) (RuleStatus, string) {
	switch rule.Operator {
	case OperatorEquals:
		if actualValue == rule.Value {
			return rule.Status, "Valor corresponde ao esperado"
		}
		return StatusFail, "Valor não corresponde ao esperado"

	case OperatorNotEquals:
		if actualValue != rule.Value {
			return rule.Status, "Valor é diferente do esperado"
		}
		return StatusFail, "Valor é igual ao valor proibido"

	case OperatorGreaterThan:
		return e.compareNumbers(actualValue, rule.Value, func(a, b float64) bool { return a > b }, "maior que")

	case OperatorGreaterOrEqual:
		return e.compareNumbers(actualValue, rule.Value, func(a, b float64) bool { return a >= b }, "maior ou igual a")

	case OperatorLessThan:
		return e.compareNumbers(actualValue, rule.Value, func(a, b float64) bool { return a < b }, "menor que")

	case OperatorLessOrEqual:
		return e.compareNumbers(actualValue, rule.Value, func(a, b float64) bool { return a <= b }, "menor ou igual a")

	case OperatorContains:
		if len(rule.Value) > 0 && len(actualValue) > 0 {
			// Simple string contains
			if contains(actualValue, rule.Value) {
				return rule.Status, "Contém o valor esperado"
			}
		}
		return StatusFail, "Não contém o valor esperado"

	case OperatorNotContains:
		if len(rule.Value) > 0 && len(actualValue) > 0 {
			if !contains(actualValue, rule.Value) {
				return rule.Status, "Não contém o valor proibido"
			}
		}
		return StatusFail, "Contém o valor proibido"

	case OperatorExists:
		return StatusPass, "Campo existe"

	case OperatorNotExists:
		return StatusFail, "Campo não deveria existir"

	case OperatorIn:
		// Value should be comma-separated list
		allowed := splitAndTrim(rule.Value, ",")
		for _, v := range allowed {
			if actualValue == v {
				return rule.Status, "Valor está na lista permitida"
			}
		}
		return StatusFail, "Valor não está na lista permitida"

	case OperatorNotIn:
		allowed := splitAndTrim(rule.Value, ",")
		for _, v := range allowed {
			if actualValue == v {
				return StatusFail, "Valor está na lista proibida"
			}
		}
		return rule.Status, "Valor não está na lista proibida"

	default:
		return StatusNeedsReview, "Operador não reconhecido"
	}
}

// compareNumbers compares two numeric values.
func (e *RuleEngine) compareNumbers(actualStr, expectedStr string, cmp func(float64, float64) bool, opDesc string) (RuleStatus, string) {
	actual, err1 := parseNumber(actualStr)
	expected, err2 := parseNumber(expectedStr)

	if err1 != nil || err2 != nil {
		return StatusInsufficientEvidence, "Valores não são numéricos válidos"
	}

	if cmp(actual, expected) {
		return StatusPass, "Valor é " + opDesc + " o limite"
	}
	return StatusFail, "Valor não é " + opDesc + " o limite"
}

// parseNumber parses a string as a number (handles currency format).
func parseNumber(s string) (float64, error) {
	// Remove currency symbols and formatting
	cleaned := ""
	for _, r := range s {
		if (r >= '0' && r <= '9') || r == '.' || r == ',' {
			cleaned += string(r)
		}
	}

	// Handle Brazilian number format: 1.000,00 -> 1000.00
	// If there are both dots and commas, the last comma is decimal separator
	// and dots are thousand separators
	lastComma := strings.LastIndex(cleaned, ",")
	lastDot := strings.LastIndex(cleaned, ".")

	if lastComma > lastDot {
		// Comma is decimal separator, dots are thousand separators
		cleaned = strings.ReplaceAll(cleaned, ".", "")
		cleaned = strings.Replace(cleaned, ",", ".", 1)
	} else if lastDot > lastComma {
		// Dot is decimal separator, commas are thousand separators (or just use as-is if US format)
		cleaned = strings.ReplaceAll(cleaned, ",", "")
	} else {
		// No decimal separator or only one type - replace comma with dot
		cleaned = strings.ReplaceAll(cleaned, ",", ".")
	}

	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return 0, nil
	}
	return strconv.ParseFloat(cleaned, 64)
}

// contains checks if a string contains a substring (case-insensitive).
func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// splitAndTrim splits a string by separator and trims whitespace.
func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// RuleRegistry manages rule definitions.
type RuleRegistry struct {
	rules map[string]RuleDefinition
}

// NewRuleRegistry creates a new rule registry.
func NewRuleRegistry() *RuleRegistry {
	return &RuleRegistry{rules: make(map[string]RuleDefinition)}
}

// Register adds a rule to the registry.
func (r *RuleRegistry) Register(rule RuleDefinition) error {
	if _, exists := r.rules[rule.ID]; exists {
		return fmt.Errorf("rule with ID %s already exists", rule.ID)
	}
	r.rules[rule.ID] = rule
	return nil
}

// Get retrieves a rule by ID.
func (r *RuleRegistry) Get(id string) (RuleDefinition, bool) {
	rule, ok := r.rules[id]
	return rule, ok
}

// ListActive returns all active rules.
func (r *RuleRegistry) ListActive() []RuleDefinition {
	var active []RuleDefinition
	for _, r := range r.rules {
		if r.Active {
			active = append(active, r)
		}
	}
	return active
}

// GetByField returns rules that apply to a specific field.
func (r *RuleRegistry) GetByField(field string) []RuleDefinition {
	var result []RuleDefinition
	for _, rule := range r.rules {
		if rule.Active && rule.Field == field {
			result = append(result, rule)
		}
	}
	return result
}

// EvaluationSummary summarizes the overall evaluation result.
type EvaluationSummary struct {
	TaskID           string
	OverallStatus    RuleStatus
	Results          []RuleEvaluationResult
	PassCount        int
	FailCount        int
	NeedsReviewCount int
	InsufficientCount int
	NotApplicableCount int
	EvaluatedAt      string
}

// Summarize creates a summary from rule evaluation results.
func Summarize(taskID string, results []RuleEvaluationResult) EvaluationSummary {
	summary := EvaluationSummary{
		TaskID:        taskID,
		Results:       results,
		EvaluatedAt:   time.Now().UTC().Format(time.RFC3339),
	}

	overallPriority := map[RuleStatus]int{
		StatusFail:               1,
		StatusNeedsReview:        2,
		StatusInsufficientEvidence: 3,
		StatusPass:               4,
		StatusNotApplicable:      5,
	}

	for _, r := range results {
		switch r.Status {
		case StatusPass:
			summary.PassCount++
		case StatusFail:
			summary.FailCount++
		case StatusNeedsReview:
			summary.NeedsReviewCount++
		case StatusInsufficientEvidence:
			summary.InsufficientCount++
		case StatusNotApplicable:
			summary.NotApplicableCount++
		}

		// Determine overall status by priority
		if priority, ok := overallPriority[r.Status]; ok {
			if currentPriority, ok := overallPriority[summary.OverallStatus]; !ok || priority < currentPriority {
				summary.OverallStatus = r.Status
			}
		}
	}

	// Default to PASS if no rules evaluated
	if len(results) == 0 {
		summary.OverallStatus = StatusPass
	}

	return summary
}