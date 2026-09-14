package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bsmart/abis/internal/tools"
)

type Executor struct {
	toolRegistry *tools.ToolRegistry
	mu           sync.Mutex
	executions   map[string]*ExecutionState
}

type ExecutionState struct {
	mu           sync.RWMutex
	PlanID       string
	EmployeeID   string
	Plan         Plan
	CurrentStep  int
	Results      []ToolResult
	Context      map[string]any
	Status       string
	Error        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	HumanInput   chan HumanInputRequest
	humanPending bool
}

func (s *ExecutionState) GetStatus() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Status
}

func (s *ExecutionState) GetCurrentStep() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.CurrentStep
}

func (s *ExecutionState) GetResults() []ToolResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make([]ToolResult, len(s.Results))
	copy(cp, s.Results)
	return cp
}

func (s *ExecutionState) GetError() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Error
}

func (s *ExecutionState) GetPlan() Plan {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Plan
}

type HumanInputRequest struct {
	Question string
	Options  []string
	Response chan string
	Context  map[string]any
}

func NewExecutor(registry *tools.ToolRegistry) *Executor {
	return &Executor{
		toolRegistry: registry,
		executions:   make(map[string]*ExecutionState),
	}
}

func (e *Executor) ExecutePlan(ctx context.Context, plan Plan, employeeID string, initialContext map[string]any) (*ExecutionState, error) {
	planID := fmt.Sprintf("exec-%d", time.Now().UnixNano())

	state := &ExecutionState{
		PlanID:     planID,
		EmployeeID: employeeID,
		Plan:       plan,
		Results:    make([]ToolResult, 0),
		Context:    initialContext,
		Status:     "running",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		HumanInput: make(chan HumanInputRequest, 1),
	}

	e.mu.Lock()
	e.executions[planID] = state
	e.mu.Unlock()

	go e.runExecution(ctx, state)

	return state, nil
}

func (e *Executor) runExecution(ctx context.Context, state *ExecutionState) {
	defer func() {
		if r := recover(); r != nil {
			state.mu.Lock()
			state.Status = "failed"
			state.Error = fmt.Sprintf("panic: %v", r)
			state.UpdatedAt = time.Now()
			state.mu.Unlock()
		}
	}()

	for {
		state.mu.RLock()
		cur := state.CurrentStep
		total := len(state.Plan.Steps)
		state.mu.RUnlock()

		if cur >= total {
			break
		}

		step := state.Plan.Steps[cur]

		if step.Condition != "" {
			state.mu.RLock()
			ctxCopy := make(map[string]any, len(state.Context))
			for k, v := range state.Context {
				ctxCopy[k] = v
			}
			state.mu.RUnlock()

			if !e.evaluateCondition(step.Condition, ctxCopy) {
				state.mu.Lock()
				state.Results = append(state.Results, ToolResult{
					Tool:    step.Tool,
					Success: true,
					Output:  map[string]any{"skipped": true, "reason": "condition not met"},
				})
				state.CurrentStep++
				state.UpdatedAt = time.Now()
				state.mu.Unlock()
				continue
			}
		}

		if !e.checkDependencies(step, state) {
			state.mu.Lock()
			state.Results = append(state.Results, ToolResult{
				Tool:    step.Tool,
				Success: false,
				Error:   "dependencies not met",
			})
			state.Status = "failed"
			state.Error = "dependencies not met"
			state.UpdatedAt = time.Now()
			state.mu.Unlock()
			return
		}

		state.mu.RLock()
		ctxCopy := make(map[string]any, len(state.Context))
		for k, v := range state.Context {
			ctxCopy[k] = v
		}
		employeeID := state.EmployeeID
		state.mu.RUnlock()

		args := e.substituteArguments(step.Arguments, ctxCopy)
		if args == nil {
			args = make(map[string]any)
		}
		args["_employee_id"] = employeeID

		result := e.executeTool(ctx, step.Tool, args)

		state.mu.Lock()
		state.Results = append(state.Results, result)
		state.UpdatedAt = time.Now()

		if !result.Success {
			state.Status = "failed"
			state.Error = fmt.Sprintf("step %d failed: %s", step.StepNumber, result.Error)
			state.mu.Unlock()
			return
		}

		state.Context[fmt.Sprintf("step_%d_result", step.StepNumber)] = result.Output
		state.Context[fmt.Sprintf("step_%d_tool", step.StepNumber)] = step.Tool
		state.mu.Unlock()

		if e.needsHumanInput(step, result) {
			input := e.requestHumanInput(ctx, state, step, result)
			state.mu.Lock()
			if input == "" {
				state.Status = "failed"
				state.Error = "human input cancelled"
				state.UpdatedAt = time.Now()
				state.mu.Unlock()
				return
			}
			state.Context["human_input"] = input
			state.CurrentStep++
			state.mu.Unlock()
		} else {
			state.mu.Lock()
			state.CurrentStep++
			state.mu.Unlock()
		}
	}

	state.mu.Lock()
	state.Status = "completed"
	state.UpdatedAt = time.Now()
	state.mu.Unlock()
}

func (e *Executor) executeTool(ctx context.Context, toolName string, args map[string]any) ToolResult {
	tool, ok := e.toolRegistry.Get(toolName)
	if !ok {
		return ToolResult{
			Tool:    toolName,
			Success: false,
			Error:   fmt.Sprintf("tool not found: %s", toolName),
		}
	}

	output, err := tool.Execute(ctx, args)
	if err != nil {
		return ToolResult{
			Tool:    toolName,
			Success: false,
			Error:   err.Error(),
		}
	}

	return ToolResult{
		Tool:    toolName,
		Success: true,
		Output:  output,
	}
}

func (e *Executor) substituteArguments(args map[string]any, context map[string]any) map[string]any {
	if args == nil {
		return nil
	}

	result := make(map[string]any)
	for k, v := range args {
		if str, ok := v.(string); ok {
			result[k] = e.substituteString(str, context)
		} else {
			result[k] = v
		}
	}
	return result
}

func (e *Executor) substituteString(str string, context map[string]any) string {
	for key, value := range context {
		placeholder := fmt.Sprintf("{{%s}}", key)
		if str == placeholder {
			return fmt.Sprintf("%v", value)
		}
		str = strings.ReplaceAll(str, placeholder, fmt.Sprintf("%v", value))
	}
	return str
}

func (e *Executor) checkDependencies(step PlanStep, state *ExecutionState) bool {
	state.mu.RLock()
	defer state.mu.RUnlock()
	for _, dep := range step.DependsOn {
		if dep > len(state.Results) || dep <= 0 {
			return false
		}
		if !state.Results[dep-1].Success {
			return false
		}
	}
	return true
}

func (e *Executor) evaluateCondition(condition string, context map[string]any) bool {
	condition = strings.TrimSpace(condition)

	if strings.Contains(condition, "==") {
		parts := strings.Split(condition, "==")
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			expected := strings.TrimSpace(parts[1])
			expected = strings.Trim(expected, `'"`)

			if val, ok := context[key]; ok {
				return fmt.Sprintf("%v", val) == expected
			}
		}
	}

	if val, ok := context[condition]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
		if s, ok := val.(string); ok {
			return s == "true" || s == "1"
		}
	}

	return false
}

func (e *Executor) needsHumanInput(step PlanStep, result ToolResult) bool {
	if result.Output != nil {
		if outMap, ok := result.Output.(map[string]any); ok {
			if needsHuman, ok := outMap["needs_human"].(bool); ok && needsHuman {
				return true
			}
			if status, ok := outMap["status"].(string); ok && status == "needs_review" {
				return true
			}
		}
	}
	return false
}

func (e *Executor) requestHumanInput(ctx context.Context, state *ExecutionState, step PlanStep, result ToolResult) string {
	question := fmt.Sprintf("A etapa '%s' requer sua intervenção. ", step.Description)

	if result.Output != nil {
		if outMap, ok := result.Output.(map[string]any); ok {
			if msg, ok := outMap["answer"].(string); ok {
				question += msg
			}
		}
	}

	req := HumanInputRequest{
		Question: question,
		Options:  []string{"continuar", "cancelar", "fornecer mais info"},
		Response: make(chan string, 1),
	}

	state.mu.Lock()
	state.Status = "awaiting_human"
	state.UpdatedAt = time.Now()
	state.humanPending = true
	state.mu.Unlock()

	select {
	case state.HumanInput <- req:
	case <-ctx.Done():
		state.mu.Lock()
		state.Status = "failed"
		state.Error = "context cancelled while awaiting human input"
		state.humanPending = false
		state.UpdatedAt = time.Now()
		state.mu.Unlock()
		return ""
	}

	select {
	case response := <-req.Response:
		state.mu.Lock()
		state.humanPending = false
		state.Status = "running"
		state.UpdatedAt = time.Now()
		state.mu.Unlock()
		return response
	case <-ctx.Done():
		state.mu.Lock()
		state.Status = "failed"
		state.Error = "context cancelled while awaiting human response"
		state.humanPending = false
		state.UpdatedAt = time.Now()
		state.mu.Unlock()
		return ""
	}
}

func (e *Executor) GetExecutionState(planID string) (*ExecutionState, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	state, ok := e.executions[planID]
	return state, ok
}

func (e *Executor) ProvideHumanInput(planID, response string) error {
	e.mu.Lock()
	state, ok := e.executions[planID]
	e.mu.Unlock()

	if !ok {
		return fmt.Errorf("execution not found: %s", planID)
	}

	deadline := time.After(5 * time.Second)
	for {
		state.mu.RLock()
		pending := state.humanPending
		state.mu.RUnlock()

		if pending {
			select {
			case req := <-state.HumanInput:
				req.Response <- response
				return nil
			default:
				time.Sleep(50 * time.Millisecond)
				continue
			}
		}

		select {
		case <-deadline:
			return fmt.Errorf("no pending human input request for plan %s", planID)
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}
}

func (e *Executor) ListExecutions() []*ExecutionState {
	e.mu.Lock()
	defer e.mu.Unlock()

	result := make([]*ExecutionState, 0, len(e.executions))
	for _, state := range e.executions {
		result = append(result, state)
	}
	return result
}

func (e *Executor) ListExecutionsByEmployee(employeeID string) []*ExecutionState {
	e.mu.Lock()
	defer e.mu.Unlock()

	result := make([]*ExecutionState, 0)
	for _, state := range e.executions {
		if state.EmployeeID == employeeID {
			result = append(result, state)
		}
	}
	return result
}
