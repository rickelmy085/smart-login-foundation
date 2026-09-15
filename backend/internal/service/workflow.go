package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/bsmart/abis/internal/documentengine"
	"github.com/bsmart/abis/internal/groq"
	"github.com/bsmart/abis/internal/knowledge"
	"github.com/bsmart/abis/internal/metrics"
	"github.com/bsmart/abis/internal/models"
	"github.com/bsmart/abis/internal/repository"

	"github.com/bsmart/abis/internal/rules"
	"github.com/bsmart/abis/internal/service/docgen"
	"github.com/bsmart/abis/internal/service/intent"
	"github.com/bsmart/abis/internal/service/requirements"

	"github.com/google/uuid"
)

var (
	ErrTaskNotFound     = errors.New("task not found")
	ErrTemplateNotFound = errors.New("template not found")
	ErrMissingData      = errors.New("missing required data")
	ErrInvalidIntent    = errors.New("invalid intent")
	ErrValidationFailed = errors.New("validation failed")
	ErrNoNormativeBase  = errors.New("no normative base found")
)

// AuditEvent represents a structured audit log entry.
type AuditEvent struct {
	EventType   string         `json:"event_type"`
	TaskID      string         `json:"task_id,omitempty"`
	EmployeeID  string         `json:"employee_id,omitempty"`
	TemplateKey string         `json:"template_key,omitempty"`
	FromStatus  string         `json:"from_status,omitempty"`
	ToStatus    string         `json:"to_status,omitempty"`
	StepKey     string         `json:"step_key,omitempty"`
	StepStatus  string         `json:"step_status,omitempty"`
	RequestID   string         `json:"request_id,omitempty"`
	Message     string         `json:"message,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	Timestamp   string         `json:"timestamp"`
}

// auditLog logs a structured audit event.
func (s *WorkflowService) auditLog(ctx context.Context, event AuditEvent) {
	event.Timestamp = time.Now().UTC().Format(time.RFC3339)
	slog.Info("audit", "event", event)
}

// WorkflowService orchestrates the document generation workflow.
type WorkflowService struct {
	repo     *repository.WorkflowRepo
	searcher *knowledge.Searcher

	intentClassifier  *intent.Classifier
	reqExtractor      *requirements.Extractor
	docGenerator      *docgen.Generator
	rulesEngine       *rules.RuleEngine
	docEngineEnabled  bool
	docEngineFallback bool
}

func NewWorkflowService(repo *repository.WorkflowRepo, searcher *knowledge.Searcher, g *groq.Client) *WorkflowService {
	svc := &WorkflowService{
		repo:     repo,
		searcher: searcher,

		intentClassifier: intent.NewClassifier(g),
		reqExtractor:     requirements.NewExtractor(g),
		docGenerator:     docgen.NewGenerator(nil), // repo will be set via WithDocumentEngine
		rulesEngine:      rules.NewRuleEngine(defaultRules()),
	}

	return svc
}

// WithDocumentEngine configures the optional Python Document Engine integration.
func (s *WorkflowService) WithDocumentEngine(client *documentengine.Client, enabled, fallback bool) {
	s.docGenerator = docgen.NewGenerator(s.repo).WithDocumentEngine(client, true, fallback)
	s.docEngineEnabled = true
	s.docEngineFallback = fallback
}

// WithEmployeeRepo configures the optional employee repository.
func (s *WorkflowService) WithEmployeeRepo(repo *repository.EmployeeRepo) {
	s.docGenerator = s.docGenerator.WithEmployeeRepo(repo)
}

// WithRulesEngine configures a custom rules engine (for testing or customization).
func (s *WorkflowService) WithRulesEngine(engine *rules.RuleEngine) {
	s.rulesEngine = engine
}

// RulesEngine returns the rules engine for testing.
func (s *WorkflowService) RulesEngine() *rules.RuleEngine {
	return s.rulesEngine
}

// UpdateTaskStatus updates the task status with state machine validation.
func (s *WorkflowService) updateTaskStatus(ctx context.Context, taskID string, newStatus models.TaskStatus, templateID string) error {
	currentTask, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		return err
	}

	if !models.IsValidTransition(currentTask.Status, newStatus) {
		return fmt.Errorf("invalid state transition: %s -> %s", currentTask.Status, newStatus)
	}

	if err := s.repo.UpdateTaskStatus(ctx, taskID, newStatus, templateID); err != nil {
		return err
	}

	s.auditLog(ctx, AuditEvent{
		EventType:  "task_status_changed",
		TaskID:     taskID,
		FromStatus: string(currentTask.Status),
		ToStatus:   string(newStatus),
		Message:    "Task status transition",
		Metadata: map[string]any{
			"template_id": templateID,
		},
	})

	return nil
}

// GetStandardWorkflowSteps returns the standard workflow steps for a template key.
func (s *WorkflowService) GetStandardWorkflowSteps(templateKey string) []models.WorkflowStep {
	switch templateKey {
	case "solicitacao_aquisicao_ti":
		return []models.WorkflowStep{
			{StepKey: "collect_data", Name: "Coletar dados", Order: 1},
			{StepKey: "validate_requirements", Name: "Validar requisitos", Order: 2},
			{StepKey: "evaluate_rules", Name: "Avaliar regras", Order: 3},
			{StepKey: "generate_document", Name: "Gerar documento", Order: 4},
			{StepKey: "review", Name: "Revisão", Order: 5},
		}
	case "memorando_interno":
		return []models.WorkflowStep{
			{StepKey: "collect_data", Name: "Coletar dados", Order: 1},
			{StepKey: "validate_requirements", Name: "Validar requisitos", Order: 2},
			{StepKey: "evaluate_rules", Name: "Avaliar regras", Order: 3},
			{StepKey: "generate_document", Name: "Gerar documento", Order: 4},
		}
	case "relatorio_operacional":
		return []models.WorkflowStep{
			{StepKey: "collect_data", Name: "Coletar dados", Order: 1},
			{StepKey: "validate_requirements", Name: "Validar requisitos", Order: 2},
			{StepKey: "evaluate_rules", Name: "Avaliar regras", Order: 3},
			{StepKey: "generate_document", Name: "Gerar documento", Order: 4},
		}
	default:
		return []models.WorkflowStep{
			{StepKey: "collect_data", Name: "Coletar dados", Order: 1},
			{StepKey: "validate_requirements", Name: "Validar requisitos", Order: 2},
			{StepKey: "evaluate_rules", Name: "Avaliar regras", Order: 3},
			{StepKey: "generate_document", Name: "Gerar documento", Order: 4},
		}
	}
}

// initializeWorkflowSteps creates the standard workflow steps for a task.
func (s *WorkflowService) initializeWorkflowSteps(ctx context.Context, taskID, templateKey string) error {
	steps := s.GetStandardWorkflowSteps(templateKey)
	now := time.Now().UTC().Format(time.RFC3339)

	for i, step := range steps {
		wfStep := models.WorkflowStep{
			ID:        "ws-" + uuid.NewString(),
			TaskID:    taskID,
			StepKey:   step.StepKey,
			Name:      step.Name,
			Status:    "pending",
			Order:     i + 1,
			CreatedAt: now,
		}
		if i == 0 {
			wfStep.Status = "in_progress"
			wfStep.StartedAt = &now
		}
		if err := s.repo.CreateWorkflowStep(ctx, wfStep); err != nil {
			return fmt.Errorf("create workflow step %s: %w", step.StepKey, err)
		}
	}
	return nil
}

// defaultRules returns the default set of rules for the ABIS system.
func defaultRules() []rules.RuleDefinition {
	return []rules.RuleDefinition{
		{
			ID:          "rule-valor-informado",
			Name:        "Valor informado",
			Description: "Verifica se o campo valor foi preenchido",
			Field:       "valor",
			Operator:    rules.OperatorExists,
			Value:       "",
			ValueType:   rules.ValueTypeString,
			Status:      rules.StatusPass,
			Sources:     []rules.RuleSource{},
			Active:      true,
			CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		},
		{
			ID:          "rule-justificativa-informada",
			Name:        "Justificativa informada",
			Description: "Verifica se a justificativa foi preenchida",
			Field:       "justificativa",
			Operator:    rules.OperatorExists,
			Value:       "",
			ValueType:   rules.ValueTypeString,
			Status:      rules.StatusPass,
			Sources:     []rules.RuleSource{},
			Active:      true,
			CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		},
		{
			ID:          "rule-fornecedor-informado",
			Name:        "Fornecedor informado",
			Description: "Verifica se o fornecedor foi informado",
			Field:       "fornecedor",
			Operator:    rules.OperatorExists,
			Value:       "",
			ValueType:   rules.ValueTypeString,
			Status:      rules.StatusPass,
			Sources:     []rules.RuleSource{},
			Active:      true,
			CreatedAt:   time.Now().UTC().Format(time.RFC3339),
		},
	}
}

// --- Intent Classification ---

// ClassifyIntent uses the LLM to classify the intent of a user request.
func (s *WorkflowService) ClassifyIntent(ctx context.Context, question string) (*models.IntentClassification, error) {
	return s.intentClassifier.Classify(ctx, question)
}

// --- Requirement Extraction ---

// ExtractRequirements uses the LLM to extract requirements from RAG search results.
func (s *WorkflowService) ExtractRequirements(ctx context.Context, procedure string, hits []knowledge.SearchHit, templateFields []models.TemplateField) (*models.RequirementExtraction, error) {
	return s.reqExtractor.Extract(ctx, procedure, hits, templateFields)
}

// ExtractData extracts structured data from a user message using the requirements extractor.
func (s *WorkflowService) ExtractData(ctx context.Context, message string, requirements []models.WorkflowRequirement) (map[string]string, error) {
	return s.reqExtractor.ExtractData(ctx, message, requirements)
}

// --- Workflow Management ---

// CreateTask creates a new workflow task from a user question.
func (s *WorkflowService) CreateTask(ctx context.Context, employeeID, question string, deadline *string, priority string, origin string) (string, error) {
	classification, err := s.ClassifyIntent(ctx, question)
	if err != nil {
		return "", err
	}

	taskID := uuid.NewString()
	task := models.WorkflowTask{
		ID:              taskID,
		EmployeeID:      employeeID,
		Intent:          models.Intent(classification.Intent),
		Procedure:       classification.Procedure,
		Status:          models.StatusDetected,
		OriginalRequest: question,
		TemplateKey:     classification.TemplateKey,
		Deadline:        deadline,
		Priority:        priority,
		Origin:          origin,
	}

	if err := s.repo.CreateTask(ctx, task); err != nil {
		return "", fmt.Errorf("create task: %w", err)
	}

<<<<<<< HEAD
	s.auditLog(ctx, AuditEvent{
		EventType:   "task_created",
		TaskID:      taskID,
		EmployeeID:  employeeID,
		TemplateKey: classification.TemplateKey,
		ToStatus:    string(models.StatusDetected),
		Message:     "Task created from user request",
	})

	slog.Info("task created", "id", taskID, "intent", classification.Intent, "template_key", classification.TemplateKey)
=======
	slog.Info("task created", "id", taskID, "intent", classification.Intent, "template_key", classification.TemplateKey, "deadline", deadline, "priority", priority, "origin", origin)
>>>>>>> a424274 (feat(fullstack): enhance workflow management with priority and deadlines)

	// Record metrics
	metrics.GlobalMetrics.RecordWorkflowTaskCreated()

	return taskID, nil
}

// GetTask retrieves a task with all its details (requirements, data, sources).
func (s *WorkflowService) GetTask(ctx context.Context, taskID string) (*TaskDetail, error) {
	task, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	requirements, err := s.repo.GetRequirements(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get requirements: %w", err)
	}

	data, err := s.repo.GetData(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get data: %w", err)
	}

	detail := &TaskDetail{
		Task:         task,
		Requirements: requirements,
		Data:         data,
	}

	if task.TemplateID != "" {
		tpl, _ := s.repo.GetTemplate(ctx, task.TemplateID)
		detail.Template = &tpl
	}

	return detail, nil
}

// UpdateTaskStatus updates the status of a task (e.g., to mark as completed).
func (s *WorkflowService) UpdateTaskStatus(ctx context.Context, taskID string, status models.TaskStatus) error {
	if _, err := s.repo.GetTask(ctx, taskID); err != nil {
		if err == sql.ErrNoRows {
			return ErrTaskNotFound
		}
		return err
	}

	return s.repo.UpdateTaskStatus(ctx, taskID, status, "")
}

// TaskDetail is the full view of a task for the frontend.
type TaskDetail struct {
	Task          models.WorkflowTask
	Requirements  []models.WorkflowRequirement
	Data          map[string]string
	Template      *models.DocumentTemplate
	MissingFields []MissingField
}

// MissingField describes a field that is missing for task progression.
type MissingField struct {
	FieldName string `json:"fieldName"`
	Label     string `json:"label"`
	Required  bool   `json:"required"`
}

// GetMissingFields returns the list of required fields that haven't been filled.
func (s *WorkflowService) GetMissingFields(ctx context.Context, taskID string) ([]MissingField, error) {
	if _, err := s.repo.GetTask(ctx, taskID); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	requirements, err := s.repo.GetRequirements(ctx, taskID)
	if err != nil {
		return nil, err
	}

	data, err := s.repo.GetData(ctx, taskID)
	if err != nil {
		return nil, err
	}

	var missing []MissingField
	for _, req := range requirements {
		if !req.Required {
			continue
		}
		value, exists := data[req.Name]
		if !exists || strings.TrimSpace(value) == "" {
			missing = append(missing, MissingField{
				FieldName: req.Name,
				Label:     req.Label,
				Required:  true,
			})
		}
	}

	return missing, nil
}

// SetData sets a field value for a task.
func (s *WorkflowService) SetData(ctx context.Context, taskID, field, value string) error {
	if _, err := s.repo.GetTask(ctx, taskID); err != nil {
		return ErrTaskNotFound
	}

	return s.repo.SetData(ctx, taskID, field, value)
}

// StartProcessing begins processing a task: search norms, extract requirements,
// select template, and move to collecting_data state.
func (s *WorkflowService) StartProcessing(ctx context.Context, taskID string) (*ProcessingResult, error) {
	task, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	result := &ProcessingResult{
		TaskID:   taskID,
		Answered: false,
	}

	// Resolve template first (using template_key from classifier or task)
	template, err := s.resolveTemplate(ctx, task)
	if err != nil && err != sql.ErrNoRows {
		slog.Warn("template lookup failed", "error", err.Error())
	}
	var templateFields []models.TemplateField
	if template.ID != "" {
		templateFields, _ = s.repo.GetTemplateFields(ctx, template.ID)
		result.Template = &template
		if err := s.repo.UpdateTaskTemplate(ctx, taskID, template.ID); err != nil {
			slog.Warn("failed to update task template", "error", err.Error())
		}
	}

	proc := task.Procedure
	if proc == "" {
		proc = task.OriginalRequest
	}

	hits, err := s.searcher.Search(ctx, proc, knowledge.SearchOptions{Limit: 20})
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	if len(hits) == 0 {
		return &ProcessingResult{
			TaskID:   taskID,
			Answered: true,
			Message:  "Os normativos disponíveis não trazem informação suficiente para concluir esta etapa com segurança.",
			Mode:     "knowledge_query",
		}, nil
	}

	// Pass template field names to extractor for better alignment
	extraction, err := s.ExtractRequirements(ctx, proc, hits, templateFields)
	if err != nil {
		slog.Warn("extract requirements failed, falling back to search-only", "error", err.Error())
		result.Answered = true
		result.Message = "Encontrei informações relevantes, mas não consegui estruturar os requisitos. Consulte os trechos abaixo:"
		result.Sources = hits
		return result, nil
	}

	for i, req := range extraction.Requirements {
		r := models.WorkflowRequirement{
			ID:             "wr-" + uuid.NewString(),
			TaskID:         taskID,
			Name:           req.Name,
			Label:          req.Label,
			Required:       req.Required,
			SourceDocument: req.SourceDocument,
			SourceSnippet:  req.SourceSnippet,
			Rank:           i,
		}
		if err := s.repo.CreateRequirement(ctx, r); err != nil {
			slog.Warn("failed to create requirement", "error", err.Error())
		}
		result.Requirements = append(result.Requirements, r)
	}

	if task.Intent == models.IntentDocumentGeneration && template.ID != "" {
		if err := s.updateTaskStatus(ctx, taskID, models.StatusCollectingData, template.ID); err != nil {
			return nil, err
		}
		// Initialize workflow steps
		if err := s.initializeWorkflowSteps(ctx, taskID, template.TemplateKey); err != nil {
			slog.Warn("failed to initialize workflow steps", "error", err.Error())
		}
	} else if task.Intent == models.IntentDocumentGeneration {
		if err := s.updateTaskStatus(ctx, taskID, models.StatusBlocked, ""); err != nil {
			return nil, err
		}
		result.Blocked = true
		result.Message = "Não foi possível identificar um template aplicável. Consulte os normativos ou entre em contato com a área responsável."
	} else {
		if err := s.updateTaskStatus(ctx, taskID, models.StatusCompleted, ""); err != nil {
			return nil, err
		}
		result.Answered = true
		result.Message = extraction.Summary
	}

	result.Sources = hits
	return result, nil
}

func (s *WorkflowService) resolveTemplate(ctx context.Context, task models.WorkflowTask) (models.DocumentTemplate, error) {
	resolver := docgen.NewTemplateResolver(s.repo)
	return resolver.Resolve(ctx, task)
}

// ProcessMessage handles a message sent to an existing task.
func (s *WorkflowService) ProcessMessage(ctx context.Context, taskID, message string) (*MessageResult, error) {
	task, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	if task.Status == models.StatusCollectingData || task.Status == models.StatusValidating {
		requirements, err := s.repo.GetRequirements(ctx, taskID)
		if err != nil {
			return nil, fmt.Errorf("get requirements: %w", err)
		}

		extractedData, err := s.reqExtractor.ExtractData(ctx, message, requirements)
		if err != nil {
			slog.Warn("extract data failed", "error", err.Error())
			return &MessageResult{
				Answer: "Não consegui entender os dados informados. Pode reformular?",
			}, nil
		}

		for field, value := range extractedData {
			if value != "" {
				if err := s.repo.SetData(ctx, taskID, field, value); err != nil {
					slog.Warn("failed to set data", "field", field, "error", err.Error())
				}
			}
		}

		missing, err := s.GetMissingFields(ctx, taskID)
		if err != nil {
			return nil, err
		}

		if len(missing) > 0 {
			return &MessageResult{
				Answer: fmt.Sprintf("Recebi os dados. Ainda preciso de: %s", formatMissingFieldNames(missing)),
			}, nil
		}

		// Update workflow step: validate_requirements
		if err := s.updateWorkflowStepStatus(ctx, taskID, "validate_requirements", "in_progress", ""); err != nil {
			slog.Warn("failed to update workflow step", "step", "validate_requirements", "error", err.Error())
		}

		s.updateTaskStatus(ctx, taskID, models.StatusValidating, task.TemplateID)
		return s.ValidateAndProceed(ctx, taskID)
	}

	return &MessageResult{Answer: message}, nil
}

func (s *WorkflowService) updateWorkflowStepStatus(ctx context.Context, taskID, stepKey, status, errorMsg string) error {
	steps, err := s.repo.GetWorkflowStepsByTask(ctx, taskID)
	if err != nil {
		return err
	}
	for _, step := range steps {
		if step.StepKey == stepKey {
			now := time.Now().UTC().Format(time.RFC3339)
			step.Status = status
			if status == "in_progress" && step.StartedAt == nil {
				step.StartedAt = &now
			}
			if status == "completed" || status == "failed" {
				step.CompletedAt = &now
			}
			step.Error = errorMsg
			return s.repo.UpdateWorkflowStep(ctx, step)
		}
	}
	return nil
}

// GetWorkflowSteps returns the workflow steps for a task.
func (s *WorkflowService) GetWorkflowSteps(ctx context.Context, taskID string) ([]models.WorkflowStep, error) {
	return s.repo.GetWorkflowStepsByTask(ctx, taskID)
}

// GetNextAction returns the next action for a task based on its current status and workflow steps.
func (s *WorkflowService) GetNextAction(ctx context.Context, taskID string) (string, error) {
	task, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		return "", err
	}

	steps, err := s.repo.GetWorkflowStepsByTask(ctx, taskID)
	if err != nil {
		return "", err
	}

	// Find first incomplete step
	for _, step := range steps {
		if step.Status == "pending" || step.Status == "in_progress" {
			return step.StepKey, nil
		}
		if step.Status == "failed" {
			return "retry_" + step.StepKey, nil
		}
	}

	// All steps completed, check task status
	switch task.Status {
	case models.StatusReadyToGenerate:
		return "generate_document", nil
	case models.StatusGenerating:
		return "waiting_generation", nil
	case models.StatusGenerated:
		return "review", nil
	case models.StatusNeedsReview:
		return "awaiting_review", nil
	case models.StatusCompleted:
		return "completed", nil
	case models.StatusBlocked:
		return "blocked", nil
	default:
		return "unknown", nil
	}
}

// ValidateAndProceed validates the task data and either generates the document or reports issues.
func (s *WorkflowService) ValidateAndProceed(ctx context.Context, taskID string) (*MessageResult, error) {
	task, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}

	missing, err := s.GetMissingFields(ctx, taskID)
	if err != nil {
		return nil, err
	}

	if len(missing) > 0 {
		return &MessageResult{
			Answer: fmt.Sprintf("Os seguintes campos são obrigatórios e estão faltando: %s", formatMissingFieldNames(missing)),
		}, nil
	}

	taskData, err := s.repo.GetData(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// Evaluate rules using the rules engine
	requirements, err := s.repo.GetRequirements(ctx, taskID)
	if err != nil {
		return nil, err
	}

	// Get template for context
	template := &models.DocumentTemplate{}
	if task.TemplateID != "" {
		tpl, err := s.repo.GetTemplate(ctx, task.TemplateID)
		if err == nil {
			template = &tpl
		}
	}

	evalContext := rules.EvaluationContext{
		TaskID:       taskID,
		TaskData:     taskData,
		Requirements: requirements,
		TemplateKey:  template.TemplateKey,
	}

	evalResults := s.rulesEngine.Evaluate(ctx, evalContext)

	// Persist rule evaluations (convert rules types to models types)
	for _, result := range evalResults {
		modelResult := models.RuleEvaluationResult{
			RuleID:   result.RuleID,
			RuleName: result.RuleName,
			Status:   models.RuleStatus(result.Status),
			Message:  result.Message,
			Input: models.RuleInput{
				FieldName: result.Input.FieldName,
				Value:     result.Input.Value,
				ValueType: string(result.Input.ValueType),
				Required:  result.Input.Required,
				Missing:   result.Input.Missing,
			},
			ExpectedValue: result.ExpectedValue,
			ActualValue:   result.ActualValue,
			Sources:       convertRuleSources(result.Sources),
			EvaluatedAt:   result.EvaluatedAt,
		}
		if err := s.repo.CreateRuleEvaluation(ctx, taskID, modelResult); err != nil {
			slog.Warn("failed to persist rule evaluation", "rule_id", result.RuleID, "error", err.Error())
		}
	}

	// Check overall status
	summary := rules.Summarize(taskID, evalResults)

	slog.Info("rule evaluation complete",
		"task_id", taskID,
		"overall_status", summary.OverallStatus,
		"pass", summary.PassCount,
		"fail", summary.FailCount,
		"needs_review", summary.NeedsReviewCount,
		"insufficient", summary.InsufficientCount,
	)

	// Handle evaluation results
	switch summary.OverallStatus {
	case rules.StatusFail:
		// Find first failing rule for the error message
		for _, r := range evalResults {
			if r.Status == rules.StatusFail {
				return &MessageResult{
					Answer: fmt.Sprintf("Validação falhou: %s", r.Message),
				}, nil
			}
		}
		return &MessageResult{
			Answer: "Validação falhou: regra não atendida",
		}, nil

	case rules.StatusNeedsReview:
		// Update workflow step
		s.updateWorkflowStepStatus(ctx, taskID, "evaluate_rules", "completed", "")
		s.updateWorkflowStepStatus(ctx, taskID, "review", "in_progress", "")

		if err := s.updateTaskStatus(ctx, taskID, models.StatusNeedsReview, task.TemplateID); err != nil {
			return nil, err
		}
		return &MessageResult{
			Answer:          "Esta solicitação requer revisão manual antes de prosseguir.",
			ReadyToGenerate: false,
		}, nil

	case rules.StatusInsufficientEvidence:
		// Find missing required fields
		missingFields := []string{}
		for _, r := range evalResults {
			if r.Status == rules.StatusInsufficientEvidence && r.Input.Required {
				missingFields = append(missingFields, r.Input.FieldName)
			}
		}
		if len(missingFields) > 0 {
			return &MessageResult{
				Answer: fmt.Sprintf("Informações insuficientes para validação. Faltam: %s", strings.Join(missingFields, ", ")),
			}, nil
		}
		return &MessageResult{
			Answer: "Informações insuficientes para completar a validação automática.",
		}, nil

	case rules.StatusNotApplicable:
		// All rules not applicable - continue
		fallthrough

	default:
		// PASS or default
		// Update workflow step
		s.updateWorkflowStepStatus(ctx, taskID, "evaluate_rules", "completed", "")

		if err := s.updateTaskStatus(ctx, taskID, models.StatusReadyToGenerate, task.TemplateID); err != nil {
			return nil, err
		}
		return &MessageResult{
			Answer:          "Todos os dados foram validados. O documento está pronto para geração.",
			ReadyToGenerate: true,
		}, nil
	}
}

// GenerateDocument generates DOCX and PDF from the template.
func (s *WorkflowService) GenerateDocument(ctx context.Context, taskID string) (*models.DocumentRun, error) {
	// Idempotency check: see if a document run already exists for this task
	existingRun, err := s.repo.GetDocumentRunByTask(ctx, taskID)
	if err == nil && existingRun.ID != "" {
		slog.Info("document already generated for task, returning existing", "task_id", taskID, "run_id", existingRun.ID)
		return &existingRun, nil
	}

	// Get current task to verify status
	currentTask, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	slog.Info("GenerateDocument starting", "task_id", taskID, "current_status", currentTask.Status)

	// Update workflow step: generate_document
	if err := s.updateWorkflowStepStatus(ctx, taskID, "generate_document", "in_progress", ""); err != nil {
		slog.Warn("failed to update workflow step", "error", err.Error())
	}

	// Transition to generating state
	if err := s.updateTaskStatus(ctx, taskID, models.StatusGenerating, ""); err != nil {
		slog.Error("failed to transition to generating", "error", err.Error(), "current_status", currentTask.Status)
		return nil, fmt.Errorf("failed to transition to generating: %w", err)
	}

	run, err := s.docGenerator.Generate(ctx, taskID)
	if err != nil {
		s.updateWorkflowStepStatus(ctx, taskID, "generate_document", "failed", err.Error())
		s.updateTaskStatus(ctx, taskID, models.StatusBlocked, "")
		return nil, err
	}

	// Generator already updates status to Generated on success
	// Update workflow steps
	if err := s.updateWorkflowStepStatus(ctx, taskID, "generate_document", "completed", ""); err != nil {
		slog.Warn("failed to update workflow step", "error", err.Error())
	}

	if err := s.updateWorkflowStepStatus(ctx, taskID, "review", "in_progress", ""); err != nil {
		slog.Warn("failed to update workflow step", "error", err.Error())
	}

	return run, nil
}

// GetDocumentsByEmployee returns all document runs for an employee.
func (s *WorkflowService) GetDocumentsByEmployee(ctx context.Context, employeeID string) ([]models.DocumentRun, error) {
	return s.repo.GetDocumentRunByEmployee(ctx, employeeID)
}

// GetTaskSources returns the search hits that were used as sources for a task.
func (s *WorkflowService) GetTaskSources(ctx context.Context, taskID string) ([]knowledge.SearchHit, error) {
	task, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}

	proc := task.Procedure
	if proc == "" {
		proc = task.OriginalRequest
	}

	return s.searcher.Search(ctx, proc, knowledge.SearchOptions{Limit: 20})
}

// GetDocumentSources returns the normative sources recorded for a document run.
func (s *WorkflowService) GetDocumentSources(ctx context.Context, runID string) ([]models.DocumentSource, error) {
	return s.repo.GetDocumentSources(ctx, runID)
}

// GetDocumentRun returns the full DocumentRun by ID.
func (s *WorkflowService) GetDocumentRun(ctx context.Context, runID string) (models.DocumentRun, error) {
	return s.repo.GetDocumentRun(ctx, runID)
}

// GetDocumentRunPath returns the local file path for a document run.
func (s *WorkflowService) GetDocumentRunPath(ctx context.Context, runID string) (string, error) {
	run, err := s.repo.GetDocumentRun(ctx, runID)
	if err != nil {
		return "", err
	}
	if run.DocxPath == "" {
		return "", fmt.Errorf("document run has no docx path")
	}
	return run.DocxPath, nil
}

// GetHistory returns a unified history view for the authenticated employee.
func (s *WorkflowService) GetHistory(ctx context.Context, employeeID string) ([]models.HistoryItem, error) {
	return s.repo.GetHistory(ctx, employeeID)
}

// ListTemplates returns all document templates.
func (s *WorkflowService) ListTemplates(ctx context.Context) ([]models.DocumentTemplate, error) {
	return s.repo.ListTemplates(ctx)
}

// DeleteTask deletes a task by ID (ownership must be verified before calling).
func (s *WorkflowService) DeleteTask(ctx context.Context, taskID string) error {
	if _, err := s.repo.GetTask(ctx, taskID); err != nil {
		if err == sql.ErrNoRows {
			return ErrTaskNotFound
		}
		return err
	}
	return s.repo.DeleteTask(ctx, taskID)
}

// HandleDocumentRequest is the unified document generation pipeline.
func (s *WorkflowService) HandleDocumentRequest(ctx context.Context, employeeID, question string) (*DocumentRequestResult, error) {
	taskID, err := s.CreateTask(ctx, employeeID, question, nil, "", "workflow")
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	procResult, err := s.StartProcessing(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("start processing: %w", err)
	}

	resp := &DocumentRequestResult{
		TaskID: taskID,
	}

	if procResult.Blocked || procResult.Template == nil {
		resp.Status = string(models.StatusBlocked)
		resp.Blocked = true
		resp.Message = "Não foi possível identificar um template aplicável. Consulte os normativos ou entre em contato com a área responsável."
		return resp, nil
	}

	resp.Template = procResult.Template
	resp.Requirements = procResult.Requirements

	missing, err := s.GetMissingFields(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get missing fields: %w", err)
	}

	if len(missing) > 0 {
		resp.Status = string(models.StatusCollectingData)
		resp.MissingFields = missing
		resp.Message = fmt.Sprintf("Para gerar o documento, são necessárias as seguintes informações: %s", formatMissingFieldNames(missing))
		return resp, nil
	}

	run, err := s.GenerateDocument(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("generate document: %w", err)
	}

	resp.Status = string(run.Status)
	resp.DocumentRun = run
	resp.DocxURL = "/api/documents/" + run.ID + "/docx"
	resp.PdfURL = "/api/documents/" + run.ID + "/pdf"
	resp.Message = "Documento gerado com sucesso"

	return resp, nil
}

// DocumentRequestResult is the result of handling a document request.
type DocumentRequestResult struct {
	TaskID        string
	Status        string
	Message       string
	Blocked       bool
	MissingFields []MissingField
	DocumentRun   *models.DocumentRun
	DocxURL       string
	PdfURL        string
	Requirements  []models.WorkflowRequirement
	Template      *models.DocumentTemplate
}

// ProcessingResult is the result of processing a task.
type ProcessingResult struct {
	TaskID       string
	Requirements []models.WorkflowRequirement
	Template     *models.DocumentTemplate
	Answered     bool
	Blocked      bool
	Message      string
	Sources      []knowledge.SearchHit
	Mode         string
}

// MessageResult is the result of processing a message in a task.
type MessageResult struct {
	Answer          string
	ReadyToGenerate bool
	MissingFields   []MissingField
}

func formatMissingFieldNames(missing []MissingField) string {
	names := make([]string, len(missing))
	for i, m := range missing {
		if m.Label != "" {
			names[i] = m.Label
		} else {
			names[i] = m.FieldName
		}
	}
	return strings.Join(names, ", ")
}

func convertRuleSources(sources []rules.RuleSource) []models.RuleSource {
	result := make([]models.RuleSource, len(sources))
	for i, s := range sources {
		result[i] = models.RuleSource{
			Document:    s.Document,
			ChunkID:     s.ChunkID,
			Snippet:     s.Snippet,
			Requirement: s.Requirement,
		}
	}
	return result
}
