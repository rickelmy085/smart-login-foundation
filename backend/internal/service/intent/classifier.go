package intent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/bsmart/abis/internal/groq"
	"github.com/bsmart/abis/internal/models"
)

const systemPrompt = `Você é o ABIS, assistente interno de inteligência operacional da Organização Bradesco.

Sua tarefa é classificar a solicitação do funcionário em uma única intenção e devolver JSON estritamente válido.

Intenções possíveis:
- knowledge_query: pergunta sobre informação (ex: "qual o limite de compras?")
- procedure_query: pergunta sobre procedimento (ex: "como faço uma aquisição?")
- document_generation: solicita criar um documento ou solicitação (ex: "preciso de uma solicitação de compra")
- form_completion: solicita preencher um formulário existente
- approval_check: pergunta sobre aprovação necessária
- requirement_check: pergunta sobre requisitos
- workflow_execution: solicita executar um processo completo
- capability_query: pergunta sobre capacidade do sistema (ex: "você consegue gerar documentos?", "quais documentos você pode criar?")

Templates disponíveis:
- solicitacao_aquisicao_ti: Solicitação de Aquisição de TI (campos: fornecedor, valor, justificativa, categoria_produto, etc.)
- memorando_interno: Memorando Interno
- relatorio_operacional: Relatório Operacional

Analise:
- Se é apenas uma pergunta = knowledge_query ou procedure_query
- Se menciona "preciso", "quero gerar", "criar", "elaborar", "solicitação" = document_generation
- "aquisição", "compra", "contratação", "contratar" com valor ou dados concretos = document_generation
- Se pergunta sobre capacidade/capacidade do ABIS = capability_query (ex: "você consegue", "você pode", "o ABIS gera", "quais documentos")

Para document_generation, extraia um "procedure" com palavras-chave de busca relevantes para encontrar os normativos aplicáveis (ex: "política de compras aquisição fornecedor valor justificativa").
Para template_key, use um dos templates acima se a solicitação claramente corresponder.

Formato JSON obrigatório:
{"intent": "...", "confidence": 0.x, "procedure": "...", "template_key": "..."}

Nunca invente procedure ou template_key se não estiver claro. Deixe vazio se incerto.

IMPORTANTE: Use EXATAMENTE os nomes das intenções acima. Para perguntas de capacidade, use "capability_query" (não "capacity_query").`

// Classifier handles intent classification using LLM.
type Classifier struct {
	client *groq.Client
}

func NewClassifier(client *groq.Client) *Classifier {
	return &Classifier{client: client}
}

func (c *Classifier) Classify(ctx context.Context, question string) (*models.IntentClassification, error) {
	if c.client == nil || c.client.Empty() {
		return nil, errors.New("groq client not configured")
	}

	resp, err := c.client.Chat(ctx, []groq.Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: fmt.Sprintf("Classifique a seguinte solicitação:\n\n\"%s\"", question)},
	})
	if err != nil {
		return nil, fmt.Errorf("classify intent: %w", err)
	}

	var classification models.IntentClassification
	if err := json.Unmarshal([]byte(strings.TrimSpace(resp)), &classification); err != nil {
		return nil, fmt.Errorf("parse intent response: %w", err)
	}

	if classification.Intent == "" {
		return nil, errors.New("intent vazio")
	}

	return &classification, nil
}