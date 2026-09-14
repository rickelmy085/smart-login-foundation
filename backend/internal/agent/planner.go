package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bsmart/abis/internal/groq"
	"github.com/bsmart/abis/internal/tools"
)

// Planner creates execution plans for user goals.
type Planner struct {
	client      *groq.Client
	toolRegistry *tools.ToolRegistry
}

func NewPlanner(client *groq.Client, registry *tools.ToolRegistry) *Planner {
	return &Planner{
		client:       client,
		toolRegistry: registry,
	}
}

// Plan represents an execution plan for a user goal.
type Plan struct {
	Goal        string                 `json:"goal"`
	Steps       []PlanStep             `json:"steps"`
	RequiredTools []string             `json:"required_tools"`
	Metadata    map[string]any         `json:"metadata,omitempty"`
}

// PlanStep represents a single step in the execution plan.
type PlanStep struct {
	StepNumber    int                    `json:"step_number"`
	Tool          string                 `json:"tool"`
	Description   string                 `json:"description"`
	Arguments     map[string]any         `json:"arguments,omitempty"`
	ExpectedOutput string                `json:"expected_output"`
	DependsOn     []int                  `json:"depends_on,omitempty"` // step numbers this depends on
	Condition     string                 `json:"condition,omitempty"` // optional condition to execute
}

// PlanningResult contains the plan and any initial tool calls.
type PlanningResult struct {
	Plan        Plan                 `json:"plan"`
	InitialCalls []ToolCall          `json:"initial_calls,omitempty"`
	Explanation string               `json:"explanation"`
}

// ToolCall represents a tool call to execute.
type ToolCall struct {
	Tool      string         `json:"tool"`
	Arguments map[string]any `json:"arguments"`
}

// Plan creates an execution plan for a user goal.
func (p *Planner) Plan(ctx context.Context, goal string, context map[string]any) (*PlanningResult, error) {
	if p.client == nil || p.client.Empty() {
		return nil, fmt.Errorf("groq client not configured")
	}

	systemPrompt := fmt.Sprintf(`Você é o ABIS Planner. Sua tarefa é criar um plano de execução para alcançar o objetivo do usuário.

FERRAMENTAS DISPONÍVEIS:
%s

REGRAS:
1. Decomponha o objetivo em etapas lógicas e sequenciais
2. Use APENAS ferramentas da lista acima
3. Cada etapa deve ter uma ferramenta, descrição e saída esperada
4. Se uma etapa depende de outra, use "depends_on" com o número da etapa
5. Para condições, use "condition" (ex: "se ready_to_generate == true")
6. Retorne APENAS JSON válido no formato especificado

FORMATO JSON OBRIGATÓRIO:
{
  "plan": {
    "goal": "string",
    "steps": [
      {
        "step_number": 1,
        "tool": "nome_da_ferramenta",
        "description": "descrição do que esta etapa faz",
        "arguments": {"param": "valor"},
        "expected_output": "o que se espera obter",
        "depends_on": [],
        "condition": ""
      }
    ],
    "required_tools": ["tool1", "tool2"],
    "metadata": {}
  },
  "initial_calls": [
    {"tool": "tool_name", "arguments": {"param": "valor"}}
  ],
  "explanation": "explicação breve do plano"
}

EXEMPLOS DE PLANOS:

Para "Quero gerar uma solicitação de aquisição de TI":
- search_normatives (buscar normas de compras)
- criar task via workflow (já existe endpoint)
- set_task_data para cada campo
- validate_task
- generate_document

Para "Qual o limite de compras?":
- search_normatives
- (responder direto)

NÃO invente ferramentas. Use apenas as listadas.`, p.buildToolDescriptions())

	userPrompt := fmt.Sprintf(`OBJETIVO DO USUÁRIO: "%s"

CONTEXTO ADICIONAL: %s

Crie um plano de execução.`, goal, p.formatContext(context))

	resp, err := p.client.ChatWithMaxTokens(ctx, []groq.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}, 4096)
	if err != nil {
		return nil, fmt.Errorf("planner LLM call: %w", err)
	}

	return p.parsePlanResponse(resp)
}

func (p *Planner) buildToolDescriptions() string {
	var b strings.Builder
	schemas := p.toolRegistry.GetSchema()
	for name, schema := range schemas {
		if schemaMap, ok := schema.(map[string]any); ok {
			desc := ""
			if d, ok := schemaMap["description"].(string); ok {
				desc = d
			}
			b.WriteString(fmt.Sprintf("- %s: %s\n", name, desc))
			if props, ok := schemaMap["properties"].(map[string]any); ok {
				b.WriteString("  Parâmetros:\n")
				for paramName, paramSchema := range props {
					if ps, ok := paramSchema.(map[string]any); ok {
						pDesc := ""
						if pd, ok := ps["description"].(string); ok {
							pDesc = pd
						}
						b.WriteString(fmt.Sprintf("    - %s: %s\n", paramName, pDesc))
					}
				}
			}
		}
	}
	return b.String()
}

func (p *Planner) formatContext(ctx map[string]any) string {
	if ctx == nil || len(ctx) == 0 {
		return "Nenhum contexto adicional."
	}
	b, _ := json.MarshalIndent(ctx, "", "  ")
	return string(b)
}

func (p *Planner) parsePlanResponse(resp string) (*PlanningResult, error) {
	resp = strings.TrimSpace(resp)
	if strings.HasPrefix(resp, "```json") {
		resp = strings.TrimPrefix(resp, "```json")
		resp = strings.TrimSuffix(resp, "```")
		resp = strings.TrimSpace(resp)
	} else if strings.HasPrefix(resp, "```") {
		resp = strings.TrimPrefix(resp, "```")
		resp = strings.TrimSuffix(resp, "```")
		resp = strings.TrimSpace(resp)
	}

	var result PlanningResult
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		idx := strings.Index(resp, "{")
		if idx >= 0 {
			jsonStr := resp[idx:]
			if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
				return nil, fmt.Errorf("parse plan response: %w", err)
			}
		} else {
			return nil, fmt.Errorf("parse plan response: %w", err)
		}
	}

	// Validate required tools exist
	for _, toolName := range result.Plan.RequiredTools {
		if _, ok := p.toolRegistry.Get(toolName); !ok {
			return nil, fmt.Errorf("planner requested unknown tool: %s", toolName)
		}
	}

	return &result, nil
}

// RefinePlan allows the planner to adjust a plan based on execution results.
func (p *Planner) RefinePlan(ctx context.Context, originalPlan Plan, results []ToolResult, feedback string) (*PlanningResult, error) {
	// For now, just re-plan with feedback as context
	context := map[string]any{
		"previous_plan": originalPlan,
		"results":       results,
		"feedback":      feedback,
	}
	return p.Plan(ctx, originalPlan.Goal, context)
}

// ToolResult represents the result of a tool execution.
type ToolResult struct {
	Tool      string `json:"tool"`
	Success   bool   `json:"success"`
	Output    any    `json:"output,omitempty"`
	Error     string `json:"error,omitempty"`
}