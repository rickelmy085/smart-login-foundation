package agent

import (
	"context"
	"fmt"

	"github.com/bsmart/abis/internal/groq"
	"github.com/bsmart/abis/internal/knowledge"
	"github.com/bsmart/abis/internal/service"
	"github.com/bsmart/abis/internal/service/intent"
	"github.com/bsmart/abis/internal/service/requirements"
	"github.com/bsmart/abis/internal/tools"
)

type AgentService struct {
	groqClient            *groq.Client
	searcher              *knowledge.Searcher
	intentClassifier      *intent.Classifier
	requirementsExtractor *requirements.Extractor
	workflowService       *service.WorkflowService

	toolRegistry *tools.ToolRegistry
	planner      *Planner
	executor     *Executor
}

func NewAgentService(
	groqClient *groq.Client,
	searcher *knowledge.Searcher,
	intentClassifier *intent.Classifier,
	requirementsExtractor *requirements.Extractor,
	workflowService *service.WorkflowService,
) *AgentService {
	toolRegistry := tools.NewToolRegistry(
		searcher,
		intentClassifier,
		requirementsExtractor,
		workflowService,
	)

	planner := NewPlanner(groqClient, toolRegistry)
	executor := NewExecutor(toolRegistry)

	return &AgentService{
		groqClient:            groqClient,
		searcher:              searcher,
		intentClassifier:      intentClassifier,
		requirementsExtractor: requirementsExtractor,
		workflowService:       workflowService,
		toolRegistry:          toolRegistry,
		planner:               planner,
		executor:              executor,
	}
}

type AgentRequest struct {
	Goal         string         `json:"goal"`
	Context      map[string]any `json:"context,omitempty"`
	TaskID       string         `json:"task_id,omitempty"`
	ResumePlanID string         `json:"resume_plan_id,omitempty"`
	HumanInput   string         `json:"human_input,omitempty"`
	PlanID       string         `json:"plan_id,omitempty"`
}

type AgentResponse struct {
	Success       bool         `json:"success"`
	Message       string       `json:"message"`
	PlanID        string       `json:"plan_id,omitempty"`
	Plan          *Plan        `json:"plan,omitempty"`
	Results       []ToolResult `json:"results,omitempty"`
	CurrentStep   int          `json:"current_step,omitempty"`
	Status        string       `json:"status"`
	NextAction    string       `json:"next_action,omitempty"`
	RequiresHuman bool         `json:"requires_human"`
	HumanQuestion string       `json:"human_question,omitempty"`
	Error         string       `json:"error,omitempty"`
}

func (a *AgentService) Process(ctx context.Context, req AgentRequest, employeeID string) (*AgentResponse, error) {
	if req.ResumePlanID != "" {
		return a.resumeExecution(ctx, req, employeeID)
	}

	if req.HumanInput != "" && req.PlanID != "" {
		return a.handleHumanInput(ctx, req, employeeID)
	}

	var initialContext map[string]any
	if req.TaskID != "" {
		taskDetail, err := a.workflowService.GetTask(ctx, req.TaskID)
		if err != nil {
			return nil, fmt.Errorf("task not found: %w", err)
		}
		if taskDetail.Task.EmployeeID != employeeID {
			return nil, fmt.Errorf("task not found")
		}
		initialContext = map[string]any{
			"task_id":      taskDetail.Task.ID,
			"task_status":  taskDetail.Task.Status,
			"template_key": taskDetail.Task.TemplateKey,
		}
	}

	if req.Context != nil {
		if initialContext == nil {
			initialContext = make(map[string]any)
		}
		for k, v := range req.Context {
			initialContext[k] = v
		}
	}
	if initialContext == nil {
		initialContext = make(map[string]any)
	}
	initialContext["employee_id"] = employeeID

	planResult, err := a.planner.Plan(ctx, req.Goal, initialContext)
	if err != nil {
		return &AgentResponse{
			Success: false,
			Error:   fmt.Sprintf("erro ao criar plano: %v", err),
		}, nil
	}

	execState, err := a.executor.ExecutePlan(ctx, planResult.Plan, employeeID, initialContext)
	if err != nil {
		return &AgentResponse{
			Success: false,
			Error:   fmt.Sprintf("erro ao executar plano: %v", err),
		}, nil
	}

	return a.buildRunningResponse(execState, planResult), nil
}

func (a *AgentService) handleHumanInput(ctx context.Context, req AgentRequest, employeeID string) (*AgentResponse, error) {
	state, ok := a.executor.GetExecutionState(req.PlanID)
	if !ok {
		return &AgentResponse{
			Success: false,
			Error:   "execução não encontrada",
		}, nil
	}

	if state.EmployeeID != employeeID {
		return &AgentResponse{
			Success: false,
			Error:   "execução não encontrada",
		}, nil
	}

	if err := a.executor.ProvideHumanInput(req.PlanID, req.HumanInput); err != nil {
		return &AgentResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &AgentResponse{
		Success: true,
		Message: "Resposta recebida. Execução retomada.",
		PlanID:  req.PlanID,
		Status:  "running",
	}, nil
}

func (a *AgentService) resumeExecution(ctx context.Context, req AgentRequest, employeeID string) (*AgentResponse, error) {
	state, ok := a.executor.GetExecutionState(req.ResumePlanID)
	if !ok {
		return &AgentResponse{
			Success: false,
			Error:   "plano não encontrado",
		}, nil
	}

	if state.EmployeeID != employeeID {
		return &AgentResponse{
			Success: false,
			Error:   "plano não encontrado",
		}, nil
	}

	return &AgentResponse{
		Success:     true,
		Message:     "Execução em andamento",
		PlanID:      req.ResumePlanID,
		Status:      state.GetStatus(),
		CurrentStep: state.GetCurrentStep(),
		Results:     state.GetResults(),
	}, nil
}

func (a *AgentService) buildRunningResponse(execState *ExecutionState, planResult *PlanningResult) *AgentResponse {
	return &AgentResponse{
		Success:     true,
		Message:     planResult.Explanation,
		PlanID:      execState.PlanID,
		Plan:        &planResult.Plan,
		Status:      execState.GetStatus(),
		CurrentStep: execState.GetCurrentStep(),
		Results:     execState.GetResults(),
	}
}

func (a *AgentService) GetExecutionStatus(planID string, employeeID string) (*AgentResponse, error) {
	state, ok := a.executor.GetExecutionState(planID)
	if !ok {
		return &AgentResponse{
			Success: false,
			Error:   "execução não encontrada",
		}, nil
	}

	if state.EmployeeID != employeeID {
		return &AgentResponse{
			Success: false,
			Error:   "execução não encontrada",
		}, nil
	}

	results := state.GetResults()
	status := state.GetStatus()
	humanQuestion := ""
	requiresHuman := status == "awaiting_human"

	if requiresHuman {
		select {
		case req := <-state.HumanInput:
			humanQuestion = req.Question
			state.HumanInput <- req
		default:
		}
	}

	return &AgentResponse{
		Success:       true,
		PlanID:        planID,
		Status:        status,
		CurrentStep:   state.GetCurrentStep(),
		Results:       results,
		Plan:          ptrPlan(state.GetPlan()),
		RequiresHuman: requiresHuman,
		HumanQuestion: humanQuestion,
		Error:         state.GetError(),
	}, nil
}

func ptrPlan(p Plan) *Plan {
	return &p
}

func (a *AgentService) ListTools() []string {
	return a.toolRegistry.List()
}

func (a *AgentService) GetToolSchema(name string) (map[string]any, bool) {
	tool, ok := a.toolRegistry.Get(name)
	if !ok {
		return nil, false
	}
	return tool.Schema(), true
}
