package tools

import (
	"context"
	"errors"
	"fmt"

	"github.com/bsmart/abis/internal/knowledge"
	"github.com/bsmart/abis/internal/models"
	"github.com/bsmart/abis/internal/service"
	"github.com/bsmart/abis/internal/service/intent"
	"github.com/bsmart/abis/internal/service/requirements"
)

var errTaskAccessDenied = errors.New("task not found")

func verifyTaskOwnership(svc *service.WorkflowService, ctx context.Context, taskID string, args map[string]any) error {
	employeeID, _ := args["_employee_id"].(string)
	if employeeID == "" {
		return errTaskAccessDenied
	}
	task, err := svc.GetTask(ctx, taskID)
	if err != nil {
		return errTaskAccessDenied
	}
	if task.Task.EmployeeID != employeeID {
		return errTaskAccessDenied
	}
	return nil
}

// Tool defines the interface for all agent tools.
type Tool interface {
	Name() string
	Description() string
	Schema() map[string]any
	Execute(ctx context.Context, args map[string]any) (any, error)
}

// ToolRegistry manages available tools for the agent.
type ToolRegistry struct {
	tools map[string]Tool
}

// NewToolRegistry creates a new tool registry with all available tools.
func NewToolRegistry(
	searcher *knowledge.Searcher,
	intentClassifier *intent.Classifier,
	requirementsExtractor *requirements.Extractor,
	workflowService *service.WorkflowService,
) *ToolRegistry {
	registry := &ToolRegistry{tools: make(map[string]Tool)}

	// Register all tools
	registry.Register(NewSearchNormativesTool(searcher))
	registry.Register(NewGetTaskTool(workflowService))
	registry.Register(NewGetRequirementsTool(workflowService))
	registry.Register(NewSetTaskDataTool(workflowService))
	registry.Register(NewValidateTaskTool(workflowService))
	registry.Register(NewGenerateDocumentTool(workflowService))
	registry.Register(NewGetWorkflowStepsTool(workflowService))
	registry.Register(NewGetNextActionTool(workflowService))
	registry.Register(NewGetTaskSourcesTool(workflowService))
	registry.Register(NewExtractDataTool(requirementsExtractor))

	return registry
}

// Register adds a tool to the registry.
func (r *ToolRegistry) Register(tool Tool) {
	r.tools[tool.Name()] = tool
}

// Get retrieves a tool by name.
func (r *ToolRegistry) Get(name string) (Tool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

// List returns all registered tool names.
func (r *ToolRegistry) List() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// GetSchema returns the JSON schema for all tools.
func (r *ToolRegistry) GetSchema() map[string]any {
	schemas := make(map[string]any)
	for name, tool := range r.tools {
		schemas[name] = tool.Schema()
	}
	return schemas
}

// ExecuteTool executes a tool by name with given arguments.
func (r *ToolRegistry) ExecuteTool(ctx context.Context, name string, args map[string]any) (any, error) {
	tool, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", name)
	}
	return tool.Execute(ctx, args)
}

// --- Tool Definitions ---

// SearchNormativesTool searches for relevant normative documents.
type SearchNormativesTool struct {
	searcher *knowledge.Searcher
}

func NewSearchNormativesTool(searcher *knowledge.Searcher) *SearchNormativesTool {
	return &SearchNormativesTool{searcher: searcher}
}

func (t *SearchNormativesTool) Name() string {
	return "search_normatives"
}

func (t *SearchNormativesTool) Description() string {
	return "Busca documentos normativos relevantes para uma consulta. Use para encontrar normas, políticas, procedimentos ou regulamentos aplicáveis a uma solicitação."
}

func (t *SearchNormativesTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Consulta de busca (ex: 'política de compras aquisição fornecedor valor')",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "Número máximo de resultados (padrão: 10)",
			},
			"min_score": map[string]any{
				"type":        "number",
				"description": "Limiar mínimo de relevância BM25 (menor = mais relevante, padrão: 15.0)",
			},
		},
		"required": []string{"query"},
	}
}

func (t *SearchNormativesTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	query, ok := args["query"].(string)
	if !ok || query == "" {
		return nil, fmt.Errorf("query is required")
	}

	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	minScore := 15.0
	if ms, ok := args["min_score"].(float64); ok {
		minScore = ms
	}

	hits, err := t.searcher.Search(ctx, query, knowledge.SearchOptions{
		Limit:    limit,
		MinScore: minScore,
	})
	if err != nil {
		return nil, err
	}

	results := make([]map[string]any, 0, len(hits))
	for _, h := range hits {
		results = append(results, map[string]any{
			"document_id":  h.Document.ID,
			"title":        h.Document.Title,
			"source_path":  h.Document.SourcePath,
			"chunk_ord":    h.Chunk.Ord,
			"snippet":      h.Snippet,
			"score":        h.Score,
			"content":      h.Chunk.Content,
		})
	}

	return map[string]any{
		"results": results,
		"count":   len(results),
	}, nil
}

// GetTaskTool retrieves a task by ID.
type GetTaskTool struct {
	service *service.WorkflowService
}

func NewGetTaskTool(s *service.WorkflowService) *GetTaskTool {
	return &GetTaskTool{service: s}
}

func (t *GetTaskTool) Name() string {
	return "get_task"
}

func (t *GetTaskTool) Description() string {
	return "Recupera uma task existente pelo ID, incluindo status, dados, requisitos e passos do workflow."
}

func (t *GetTaskTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task_id": map[string]any{
				"type":        "string",
				"description": "ID da task (formato UUID)",
			},
		},
		"required": []string{"task_id"},
	}
}

func (t *GetTaskTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	taskID, ok := args["task_id"].(string)
	if !ok || taskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}

	if err := verifyTaskOwnership(t.service, ctx, taskID, args); err != nil {
		return nil, err
	}

	taskDetail, err := t.service.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}

	return taskDetail, nil
}

// GetRequirementsTool retrieves requirements for a task.
type GetRequirementsTool struct {
	service *service.WorkflowService
}

func NewGetRequirementsTool(s *service.WorkflowService) *GetRequirementsTool {
	return &GetRequirementsTool{service: s}
}

func (t *GetRequirementsTool) Name() string {
	return "get_requirements"
}

func (t *GetRequirementsTool) Description() string {
	return "Lista os requisitos identificados para uma task, incluindo quais estão preenchidos e quais faltam."
}

func (t *GetRequirementsTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task_id": map[string]any{
				"type":        "string",
				"description": "ID da task",
			},
		},
		"required": []string{"task_id"},
	}
}

func (t *GetRequirementsTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	taskID, ok := args["task_id"].(string)
	if !ok || taskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}

	if err := verifyTaskOwnership(t.service, ctx, taskID, args); err != nil {
		return nil, err
	}

	taskDetail, err := t.service.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}

	requirements := make([]map[string]any, 0, len(taskDetail.Requirements))
	for _, r := range taskDetail.Requirements {
		hasValue := false
		if val, ok := taskDetail.Data[r.Name]; ok && val != "" {
			hasValue = true
		}
		requirements = append(requirements, map[string]any{
			"name":          r.Name,
			"label":         r.Label,
			"required":      r.Required,
			"type":          "text", // WorkflowRequirement doesn't have type; template fields do
			"has_value":     hasValue,
			"source_doc":    r.SourceDocument,
			"source_snippet": r.SourceSnippet,
		})
	}

	return map[string]any{
		"requirements": requirements,
	}, nil
}

// SetTaskDataTool sets a field value for a task.
type SetTaskDataTool struct {
	service *service.WorkflowService
}

func NewSetTaskDataTool(s *service.WorkflowService) *SetTaskDataTool {
	return &SetTaskDataTool{service: s}
}

func (t *SetTaskDataTool) Name() string {
	return "set_task_data"
}

func (t *SetTaskDataTool) Description() string {
	return "Define o valor de um campo para uma task. Use para preencher requisitos faltantes."
}

func (t *SetTaskDataTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task_id": map[string]any{
				"type":        "string",
				"description": "ID da task",
			},
			"field_name": map[string]any{
				"type":        "string",
				"description": "Nome técnico do campo (ex: 'fornecedor', 'valor', 'justificativa')",
			},
			"value": map[string]any{
				"type":        "string",
				"description": "Valor a ser definido",
			},
		},
		"required": []string{"task_id", "field_name", "value"},
	}
}

func (t *SetTaskDataTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	taskID, _ := args["task_id"].(string)
	fieldName, _ := args["field_name"].(string)
	value, _ := args["value"].(string)

	if taskID == "" || fieldName == "" || value == "" {
		return nil, fmt.Errorf("task_id, field_name and value are required")
	}

	if err := verifyTaskOwnership(t.service, ctx, taskID, args); err != nil {
		return nil, err
	}

	err := t.service.SetData(ctx, taskID, fieldName, value)
	if err != nil {
		return nil, err
	}

	return map[string]any{"success": true}, nil
}

// ValidateTaskTool validates a task (runs rule engine).
type ValidateTaskTool struct {
	service *service.WorkflowService
}

func NewValidateTaskTool(s *service.WorkflowService) *ValidateTaskTool {
	return &ValidateTaskTool{service: s}
}

func (t *ValidateTaskTool) Name() string {
	return "validate_task"
}

func (t *ValidateTaskTool) Description() string {
	return "Executa a validação dos dados da task contra as regras do Rule Engine. Retorna PASS/FAIL/NEEDS_REVIEW e próximo passo."
}

func (t *ValidateTaskTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task_id": map[string]any{
				"type":        "string",
				"description": "ID da task",
			},
		},
		"required": []string{"task_id"},
	}
}

func (t *ValidateTaskTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	taskID, ok := args["task_id"].(string)
	if !ok || taskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}

	if err := verifyTaskOwnership(t.service, ctx, taskID, args); err != nil {
		return nil, err
	}

	result, err := t.service.ValidateAndProceed(ctx, taskID)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"answer":           result.Answer,
		"ready_to_generate": result.ReadyToGenerate,
		"missing_fields":   result.MissingFields,
	}, nil
}

// GenerateDocumentTool generates the document for a task.
type GenerateDocumentTool struct {
	service *service.WorkflowService
}

func NewGenerateDocumentTool(s *service.WorkflowService) *GenerateDocumentTool {
	return &GenerateDocumentTool{service: s}
}

func (t *GenerateDocumentTool) Name() string {
	return "generate_document"
}

func (t *GenerateDocumentTool) Description() string {
	return "Gera o documento (DOCX/PDF) para a task. Só funciona após validação com PASS. Retorna document_run_id e URLs de download."
}

func (t *GenerateDocumentTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task_id": map[string]any{
				"type":        "string",
				"description": "ID da task",
			},
		},
		"required": []string{"task_id"},
	}
}

func (t *GenerateDocumentTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	taskID, ok := args["task_id"].(string)
	if !ok || taskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}

	if err := verifyTaskOwnership(t.service, ctx, taskID, args); err != nil {
		return nil, err
	}

	run, err := t.service.GenerateDocument(ctx, taskID)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"document_run_id": run.ID,
		"docx_url":        run.DocxPath,
		"pdf_url":         run.PdfPath,
		"status":          run.Status,
	}, nil
}

// GetWorkflowStepsTool retrieves workflow steps for a task.
type GetWorkflowStepsTool struct {
	service *service.WorkflowService
}

func NewGetWorkflowStepsTool(s *service.WorkflowService) *GetWorkflowStepsTool {
	return &GetWorkflowStepsTool{service: s}
}

func (t *GetWorkflowStepsTool) Name() string {
	return "get_workflow_steps"
}

func (t *GetWorkflowStepsTool) Description() string {
	return "Retorna os passos do workflow para uma task, com status de cada etapa."
}

func (t *GetWorkflowStepsTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task_id": map[string]any{
				"type":        "string",
				"description": "ID da task",
			},
		},
		"required": []string{"task_id"},
	}
}

func (t *GetWorkflowStepsTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	taskID, ok := args["task_id"].(string)
	if !ok || taskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}

	if err := verifyTaskOwnership(t.service, ctx, taskID, args); err != nil {
		return nil, err
	}

	steps, err := t.service.GetWorkflowSteps(ctx, taskID)
	if err != nil {
		return nil, err
	}

	result := make([]map[string]any, 0, len(steps))
	for _, s := range steps {
		result = append(result, map[string]any{
			"id":           s.ID,
			"step_key":     s.StepKey,
			"name":         s.Name,
			"status":       s.Status,
			"order":        s.Order,
			"started_at":   s.StartedAt,
			"completed_at": s.CompletedAt,
			"error":        s.Error,
		})
	}

	return map[string]any{"steps": result}, nil
}

// GetNextActionTool retrieves the next action for a task.
type GetNextActionTool struct {
	service *service.WorkflowService
}

func NewGetNextActionTool(s *service.WorkflowService) *GetNextActionTool {
	return &GetNextActionTool{service: s}
}

func (t *GetNextActionTool) Name() string {
	return "get_next_action"
}

func (t *GetNextActionTool) Description() string {
	return "Retorna a próxima ação recomendada para a task baseada no estado atual."
}

func (t *GetNextActionTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task_id": map[string]any{
				"type":        "string",
				"description": "ID da task",
			},
		},
		"required": []string{"task_id"},
	}
}

func (t *GetNextActionTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	taskID, ok := args["task_id"].(string)
	if !ok || taskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}

	if err := verifyTaskOwnership(t.service, ctx, taskID, args); err != nil {
		return nil, err
	}

	nextAction, err := t.service.GetNextAction(ctx, taskID)
	if err != nil {
		return nil, err
	}

	return map[string]any{"next_action": nextAction}, nil
}

// GetTaskSourcesTool retrieves normative sources used for a task.
type GetTaskSourcesTool struct {
	service *service.WorkflowService
}

func NewGetTaskSourcesTool(s *service.WorkflowService) *GetTaskSourcesTool {
	return &GetTaskSourcesTool{service: s}
}

func (t *GetTaskSourcesTool) Name() string {
	return "get_task_sources"
}

func (t *GetTaskSourcesTool) Description() string {
	return "Retorna as fontes normativas (documentos/chunks) que foram usadas como evidência para a task."
}

func (t *GetTaskSourcesTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"task_id": map[string]any{
				"type":        "string",
				"description": "ID da task",
			},
		},
		"required": []string{"task_id"},
	}
}

func (t *GetTaskSourcesTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	taskID, ok := args["task_id"].(string)
	if !ok || taskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}

	if err := verifyTaskOwnership(t.service, ctx, taskID, args); err != nil {
		return nil, err
	}

	sources, err := t.service.GetTaskSources(ctx, taskID)
	if err != nil {
		return nil, err
	}

	result := make([]map[string]any, 0, len(sources))
	for _, s := range sources {
		result = append(result, map[string]any{
			"document_id": s.Document.ID,
			"title":       s.Document.Title,
			"snippet":     s.Snippet,
			"score":       s.Score,
		})
	}

	return map[string]any{"sources": result}, nil
}

// ExtractDataTool extracts structured data from user message.
type ExtractDataTool struct {
	extractor *requirements.Extractor
}

func NewExtractDataTool(e *requirements.Extractor) *ExtractDataTool {
	return &ExtractDataTool{extractor: e}
}

func (t *ExtractDataTool) Name() string {
	return "extract_data"
}

func (t *ExtractDataTool) Description() string {
	return "Extrai dados estruturados (campos/valores) de uma mensagem do usuário. Útil para preencher múltiplos campos de uma vez."
}

func (t *ExtractDataTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"message": map[string]any{
				"type":        "string",
				"description": "Mensagem do usuário contendo os dados",
			},
			"requirement_names": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Nomes dos requisitos/campos a extrair",
			},
		},
		"required": []string{"message", "requirement_names"},
	}
}

func (t *ExtractDataTool) Execute(ctx context.Context, args map[string]any) (any, error) {
	message, _ := args["message"].(string)
	reqNamesInterface, _ := args["requirement_names"].([]any)

	if message == "" {
		return nil, fmt.Errorf("message is required")
	}

	reqNames := make([]string, 0, len(reqNamesInterface))
	for _, n := range reqNamesInterface {
		if s, ok := n.(string); ok {
			reqNames = append(reqNames, s)
		}
	}

	// Build minimal requirements for extraction
	requirements := make([]models.WorkflowRequirement, 0, len(reqNames))
	for _, n := range reqNames {
		requirements = append(requirements, models.WorkflowRequirement{Name: n})
	}

	data, err := t.extractor.ExtractData(ctx, message, requirements)
	if err != nil {
		return nil, err
	}

	return data, nil
}