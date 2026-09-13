package requirements

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/bsmart/abis/internal/groq"
	"github.com/bsmart/abis/internal/knowledge"
	"github.com/bsmart/abis/internal/models"
)

const systemPrompt = `Você é o ABIS, assistente interno de inteligência operacional da Organização Bradesco.

Sua tarefa é analisar os trechos de normativos fornecidos e extrair os requisitos operacionais para: %s

Campos do template esperados (use EXATAMENTE estes nomes técnicos):
%s

Para cada requisito identificado, retorne:
- name: identificador técnico (DEVE ser um dos campos do template acima, ex: "fornecedor", "valor", "justificativa", "aprovacao_cade")
- label: texto legível (ex: "Fornecedor", "Valor da aquisição")
- required: true se obrigatório
- type: text|number|currency|date|boolean|select|textarea
- source_document: título do documento de origem
- source_snippet: trecho relevante (máx 500 chars)

Formato JSON obrigatório:
{"procedure": "...", "summary": "...", "requirements": [...]}

NÃO invente requisitos. Se um requisito não estiver nos trechos fornecidos, não inclua.
Só retorne JSON válido.`

const extractDataPrompt = `Extraia os valores dos seguintes campos da mensagem do usuário.
Campos possíveis: %s

Mensagem: "%s"

Retorne JSON no formato: {"campo": "valor"}
Se um campo não estiver presente na mensagem, não o inclua.
Preencha apenas com informações explícitas na mensagem.`

type Extractor struct {
	client *groq.Client
}

func NewExtractor(client *groq.Client) *Extractor {
	return &Extractor{client: client}
}

func (e *Extractor) Extract(ctx context.Context, procedure string, hits []knowledge.SearchHit, templateFields []models.TemplateField) (*models.RequirementExtraction, error) {
	if e.client == nil || e.client.Empty() {
		return nil, errors.New("groq client not configured")
	}

	// Build template field names string for the prompt
	var fieldNames []string
	for _, f := range templateFields {
		fieldNames = append(fieldNames, f.FieldName)
	}
	fieldNamesStr := "Nenhum template associado"
	if len(fieldNames) > 0 {
		fieldNamesStr = strings.Join(fieldNames, ", ")
	}

	var contextBuilder strings.Builder
	contextBuilder.WriteString("NORMATIVOS ENCONTRADOS:\n\n")
	maxContentLen := 1000
	for i, h := range hits {
		if i >= 3 {
			break
		}
		content := h.Chunk.Content
		if len(content) > maxContentLen {
			content = content[:maxContentLen] + "..."
		}
		contextBuilder.WriteString(fmt.Sprintf("Documento: %s\n", h.Document.Title))
		contextBuilder.WriteString(fmt.Sprintf("Trecho:\n%s\n\n", content))
	}

	resp, err := e.client.ChatWithMaxTokens(ctx, []groq.Message{
		{Role: "system", Content: fmt.Sprintf(systemPrompt, procedure, fieldNamesStr)},
		{Role: "user", Content: contextBuilder.String()},
	}, 4096)
	if err != nil {
		return nil, fmt.Errorf("extract requirements: %w", err)
	}

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

	var extraction models.RequirementExtraction
	if err := json.Unmarshal([]byte(resp), &extraction); err != nil {
		idx := strings.Index(resp, "{")
		if idx >= 0 {
			jsonStr := resp[idx:]
			if err := json.Unmarshal([]byte(jsonStr), &extraction); err != nil {
				return nil, fmt.Errorf("parse requirements response: %w", err)
			}
		} else {
			return nil, fmt.Errorf("parse requirements response: %w", err)
		}
	}

	return &extraction, nil
}

func (e *Extractor) ExtractData(ctx context.Context, message string, requirements []models.WorkflowRequirement) (map[string]string, error) {
	if e.client == nil || e.client.Empty() {
		return nil, errors.New("groq client not configured")
	}

	var reqNames []string
	for _, r := range requirements {
		reqNames = append(reqNames, r.Name)
	}

	prompt := fmt.Sprintf(extractDataPrompt, strings.Join(reqNames, ", "), message)

	resp, err := e.client.Chat(ctx, []groq.Message{
		{Role: "system", Content: "Você é um assistente que extrai dados estruturados de mensagens."},
		{Role: "user", Content: prompt},
	})
	if err != nil {
		return nil, fmt.Errorf("extract data LLM call: %w", err)
	}

	var result map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(resp)), &result); err != nil {
		return nil, fmt.Errorf("parse data extraction: %w", err)
	}

	return result, nil
}