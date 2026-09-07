package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bsmart/abis/internal/document"
	"github.com/bsmart/abis/internal/groq"
	"github.com/bsmart/abis/internal/knowledge"
	"github.com/bsmart/abis/internal/models"
	"github.com/bsmart/abis/internal/repository"
	"github.com/google/uuid"
)

var (
	ErrTaskNotFound      = errors.New("task not found")
	ErrTemplateNotFound  = errors.New("template not found")
	ErrMissingData       = errors.New("missing required data")
	ErrInvalidIntent     = errors.New("invalid intent")
	ErrValidationFailed  = errors.New("validation failed")
	ErrNoNormativeBase   = errors.New("no normative base found")
)

// WorkflowService orchestrates the document generation workflow.
type WorkflowService struct {
	repo     *repository.WorkflowRepo
	searcher *knowledge.Searcher
	groq     *groq.Client
}

func NewWorkflowService(repo *repository.WorkflowRepo, searcher *knowledge.Searcher, g *groq.Client) *WorkflowService {
	return &WorkflowService{
		repo:     repo,
		searcher: searcher,
		groq:     g,
	}
}

// --- Intent Classification ---

const intentSystemPrompt = `Você é o ABIS, assistente interno de inteligência operacional da Organização Bradesco.

Sua tarefa é classificar a solicitação do funcionário em uma única intenção e devolver JSON estritamente válido.

Intenções possíveis:
- knowledge_query: pergunta sobre informação (ex: "qual o limite de compras?")
- procedure_query: pergunta sobre procedimento (ex: "como faço uma aquisição?")
- document_generation: solicita criar um documento ou solicitação (ex: "preciso de uma solicitação de compra")
- form_completion: solicita preencher um formulário existente
- approval_check: pergunta sobre aprovação necessária
- requirement_check: pergunta sobre requisitos
- workflow_execution: solicita executar um processo completo

Analise:
- Se é apenas uma pergunta = knowledge_query ou procedure_query
- Se menciona "preciso", "quero gerar", "criar", "elaborar", "solicitação" = document_generation
- "aquisição", "compra", "contratação", "contratar" com valor ou dados concretos = document_generation

Formato JSON obrigatório:
{"intent": "...", "confidence": 0.x, "procedure": "...", "template_key": "..."}

Nunca invente procedure ou template_key se não estiver claro. Deixe vazio se incerto.`

// ClassifyIntent uses the LLM to classify the intent of a user request.
func (s *WorkflowService) ClassifyIntent(ctx context.Context, question string) (*models.IntentClassification, error) {
	fmt.Printf("[WORKFLOW] ClassifyIntent iniciado question=%s\n", question)
	if s.groq == nil || s.groq.Empty() {
		fmt.Println("[WORKFLOW] ClassifyIntent groq client nao configurado")
		return nil, errors.New("groq client not configured")
	}

	resp, err := s.groq.Chat(ctx, []groq.Message{
		{Role: "system", Content: intentSystemPrompt},
		{Role: "user", Content: fmt.Sprintf("Classifique a seguinte solicitação:\n\n\"%s\"", question)},
	})
	if err != nil {
		fmt.Printf("[WORKFLOW] ClassifyIntent erro groq: %v\n", err)
		return nil, fmt.Errorf("classify intent: %w", err)
	}
	fmt.Printf("[WORKFLOW] ClassifyIntent resposta LLM: %s\n", resp)

	var classification models.IntentClassification
	if err := json.Unmarshal([]byte(strings.TrimSpace(resp)), &classification); err != nil {
		fmt.Printf("[WORKFLOW] ClassifyIntent erro parse JSON: %v\n", err)
		return nil, fmt.Errorf("parse intent response: %w", err)
	}

	if classification.Intent == "" {
		fmt.Println("[WORKFLOW] ClassifyIntent intent vazio")
		return nil, ErrInvalidIntent
	}

	slog.Info("intent classified", "intent", classification.Intent, "confidence", classification.Confidence)
	fmt.Printf("[WORKFLOW] ClassifyIntent sucesso intent=%s confidence=%.2f procedure=%s\n",
		classification.Intent, classification.Confidence, classification.Procedure)
	return &classification, nil
}

// --- Requirement Extraction ---

const requirementSystemPrompt = `Você é o ABIS, assistente interno de inteligência operacional da Organização Bradesco.

Sua tarefa é analisar os trechos de normativos fornecidos e extrair os requisitos operacionais para: %s

Para cada requisito identificado, retorne:
- name: identificador técnico (ex: "fornecedor", "valor", "justificativa", "aprovacao_cade")
- label: texto legível (ex: "Fornecedor", "Valor da aquisição")
- required: true se obrigatório
- type: text|number|currency|date|boolean|select|textarea
- source_document: título do documento de origem
- source_snippet: trecho relevante (máx 500 chars)

Formato JSON obrigatório:
{"procedure": "...", "summary": "...", "requirements": [...]}

NÃO invente requisitos. Se um requisito não estiver nos trechos fornecidos, não inclua.
Só retorne JSON válido.`

// ExtractRequirements uses the LLM to extract requirements from RAG search results.
func (s *WorkflowService) ExtractRequirements(ctx context.Context, procedure string, hits []knowledge.SearchHit) (*models.RequirementExtraction, error) {
	fmt.Printf("[WORKFLOW] ExtractRequirements procedure=%s hits=%d\n", procedure, len(hits))
	if s.groq == nil || s.groq.Empty() {
		fmt.Println("[WORKFLOW] ExtractRequirements groq client nao configurado")
		return nil, errors.New("groq client not configured")
	}

	// Build context from search hits (truncate aggressively to manage token budget)
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

	resp, err := s.groq.ChatWithMaxTokens(ctx, []groq.Message{
		{Role: "system", Content: fmt.Sprintf(requirementSystemPrompt, procedure)},
		{Role: "user", Content: contextBuilder.String()},
	}, 4096)
	if err != nil {
		fmt.Printf("[WORKFLOW] ExtractRequirements erro groq: %v\n", err)
		return nil, fmt.Errorf("extract requirements: %w", err)
	}
	fmt.Printf("[WORKFLOW] ExtractRequirements resposta LLM: %s\n", resp)

	// Try to parse JSON; if it fails, try extracting JSON from markdown
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
		// Try to find JSON in the response
		idx := strings.Index(resp, "{")
		if idx >= 0 {
			jsonStr := resp[idx:]
			if err := json.Unmarshal([]byte(jsonStr), &extraction); err != nil {
				fmt.Printf("[WORKFLOW] ExtractRequirements erro parse JSON: %v\n", err)
				return nil, fmt.Errorf("parse requirements response: %w", err)
			}
		} else {
			fmt.Printf("[WORKFLOW] ExtractRequirements erro parse JSON: %v\n", err)
			return nil, fmt.Errorf("parse requirements response: %w", err)
		}
	}

	slog.Info("requirements extracted", "count", len(extraction.Requirements), "procedure", extraction.Procedure)
	fmt.Printf("[WORKFLOW] ExtractRequirements sucesso: %d requisitos procedure=%s\n", len(extraction.Requirements), extraction.Procedure)
	return &extraction, nil
}

// --- Workflow Management ---

// CreateTask creates a new workflow task from a user question.
func (s *WorkflowService) CreateTask(ctx context.Context, employeeID, question string) (string, error) {
	fmt.Printf("[WORKFLOW] CreateTask employeeID=%s question=%s\n", employeeID, question)
	classification, err := s.ClassifyIntent(ctx, question)
	if err != nil {
		fmt.Printf("[WORKFLOW] CreateTask erro classify: %v\n", err)
		return "", err
	}

	taskID := uuid.NewString()
	task := models.WorkflowTask{
		ID:             taskID,
		EmployeeID:     employeeID,
		Intent:         models.Intent(classification.Intent),
		Procedure:      classification.Procedure,
		Status:         models.StatusDetected,
		OriginalRequest: question,
	}

	if err := s.repo.CreateTask(ctx, task); err != nil {
		fmt.Printf("[WORKFLOW] CreateTask erro repo: %v\n", err)
		return "", fmt.Errorf("create task: %w", err)
	}

	slog.Info("task created", "id", taskID, "intent", classification.Intent)
	fmt.Printf("[WORKFLOW] CreateTask sucesso taskID=%s intent=%s\n", taskID, classification.Intent)
	return taskID, nil
}

// GetTask retrieves a task with all its details (requirements, data, sources).
func (s *WorkflowService) GetTask(ctx context.Context, taskID string) (*TaskDetail, error) {
	fmt.Printf("[WORKFLOW] GetTask taskID=%s\n", taskID)
	task, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Printf("[WORKFLOW] GetTask task nao encontrada taskID=%s\n", taskID)
			return nil, ErrTaskNotFound
		}
		fmt.Printf("[WORKFLOW] GetTask erro repo: %v\n", err)
		return nil, err
	}
	fmt.Printf("[WORKFLOW] GetTask sucesso taskID=%s status=%s\n", taskID, task.Status)

	requirements, err := s.repo.GetRequirements(ctx, taskID)
	if err != nil {
		fmt.Printf("[WORKFLOW] GetTask erro get requirements: %v\n", err)
		return nil, fmt.Errorf("get requirements: %w", err)
	}

	data, err := s.repo.GetData(ctx, taskID)
	if err != nil {
		fmt.Printf("[WORKFLOW] GetTask erro get data: %v\n", err)
		return nil, fmt.Errorf("get data: %w", err)
	}
	fmt.Printf("[WORKFLOW] GetTask requirements=%d data_fields=%d\n", len(requirements), len(data))

	detail := &TaskDetail{
		Task:         task,
		Requirements: requirements,
		Data:         data,
	}

	// If status is document_generation with template, get template info
	if task.TemplateID != "" {
		tpl, _ := s.repo.GetTemplate(ctx, task.TemplateID)
		detail.Template = &tpl
		fmt.Printf("[WORKFLOW] GetTask template carregado templateID=%s\n", task.TemplateID)
	}

	return detail, nil
}

// TaskDetail is the full view of a task for the frontend.
type TaskDetail struct {
	Task         models.WorkflowTask
	Requirements []models.WorkflowRequirement
	Data         map[string]string
	Template     *models.DocumentTemplate
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
	fmt.Printf("[WORKFLOW] StartProcessing taskID=%s\n", taskID)
	task, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Printf("[WORKFLOW] StartProcessing task nao encontrada taskID=%s\n", taskID)
			return nil, ErrTaskNotFound
		}
		fmt.Printf("[WORKFLOW] StartProcessing erro repo: %v\n", err)
		return nil, err
	}
	fmt.Printf("[WORKFLOW] StartProcessing task encontrada status=%s intent=%s\n", task.Status, task.Intent)

	// If it's a knowledge_query/procedure_query, just do a search
	result := &ProcessingResult{
		TaskID:   taskID,
		Answered: false,
	}

	// For document generation, we need to search norms and extract requirements
	proc := task.Procedure
	if proc == "" {
		proc = task.OriginalRequest
	}
	fmt.Printf("[WORKFLOW] StartProcessing procedure=%s\n", proc)

	// Search for relevant chunks
	hits, err := s.searcher.Search(ctx, proc, knowledge.SearchOptions{Limit: 20})
	if err != nil {
		fmt.Printf("[WORKFLOW] StartProcessing erro search: %v\n", err)
		return nil, fmt.Errorf("search: %w", err)
	}
	fmt.Printf("[WORKFLOW] StartProcessing search sucesso: %d hits\n", len(hits))

	if len(hits) == 0 {
		fmt.Println("[WORKFLOW] StartProcessing nenhum hit encontrado")
		return &ProcessingResult{
			TaskID:    taskID,
			Answered:  true,
			Message:   "Os normativos disponíveis não trazem informação suficiente para concluir esta etapa com segurança.",
			Mode:      "knowledge_query",
		}, nil
	}

	// Extract requirements using LLM
	extraction, err := s.ExtractRequirements(ctx, proc, hits)
	if err != nil {
		fmt.Printf("[WORKFLOW] StartProcessing erro extract requirements: %v\n", err)
		slog.Warn("extract requirements failed, falling back to search-only", "error", err. Error())
		result.Answered = true
		result.Message = "Encontrei informações relevantes, mas não consegui estruturar os requisitos. Consulte os trechos abaixo:"
		result.Sources = hits
		return result, nil
	}
	fmt.Printf("[WORKFLOW] StartProcessing requirements extraidas: %d\n", len(extraction.Requirements))

	// Persist requirements
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
			fmt.Printf("[WORKFLOW] StartProcessing erro create requirement: %v\n", err)
			slog.Warn("failed to create requirement", "error", err.Error())
		}
		result.Requirements = append(result.Requirements, r)
	}

	// Try to find a matching template
	template, err := s.repo.GetTemplateByKey(ctx, proc)
	if err != nil && err != sql.ErrNoRows {
		fmt.Printf("[WORKFLOW] StartProcessing erro template lookup: %v\n", err)
		slog.Warn("template lookup failed", "error", err.Error())
	}
	if template.ID != "" {
		result.Template = &template
		if err := s.repo.UpdateTaskTemplate(ctx, taskID, template.ID); err != nil {
			fmt.Printf("[WORKFLOW] StartProcessing erro update template: %v\n", err)
			slog.Warn("failed to update task template", "error", err.Error())
		}
		fmt.Printf("[WORKFLOW] StartProcessing template encontrado templateID=%s\n", template.ID)
	} else {
		fmt.Println("[WORKFLOW] StartProcessing nenhum template encontrado")
	}

	// Update task status to collecting_data
	if task.Intent == models.IntentDocumentGeneration && template.ID != "" {
		s.repo.UpdateTaskStatus(ctx, taskID, models.StatusCollectingData, template.ID)
		fmt.Println("[WORKFLOW] StartProcessing status atualizado para collecting_data")
	} else if task.Intent == models.IntentDocumentGeneration {
		s.repo.UpdateTaskStatus(ctx, taskID, models.StatusBlocked, "")
		result.Blocked = true
		result.Message = "Não foi possível identificar um template aplicável. Consulte os normativos ou entre em contato com a área responsável."
		fmt.Println("[WORKFLOW] StartProcessing bloqueado: template nao encontrado")
	} else {
		s.repo.UpdateTaskStatus(ctx, taskID, models.StatusCompleted, "")
		result.Answered = true
		result.Message = extraction.Summary
		fmt.Println("[WORKFLOW] StartProcessing status atualizado para completed")
	}

	// Store the found sources
	result.Sources = hits

	return result, nil
}

// ProcessingResult is the result of processing a task.
type ProcessingResult struct {
	TaskID        string
	Requirements  []models.WorkflowRequirement
	Template      *models.DocumentTemplate
	Answered      bool
	Blocked       bool
	Message       string
	Sources       []knowledge.SearchHit
	Mode          string
}

// MessageResult is the result of processing a message in a task.
type MessageResult struct {
	Answer           string
	ReadyToGenerate  bool
	MissingFields    []MissingField
}

// ProcessMessage handles a message sent to an existing task.
// It stores the data and checks if validation/generation can proceed.
func (s *WorkflowService) ProcessMessage(ctx context.Context, taskID, message string) (*MessageResult, error) {
	fmt.Printf("[WORKFLOW] ProcessMessage taskID=%s message=%s\n", taskID, message)
	task, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Printf("[WORKFLOW] ProcessMessage task nao encontrada taskID=%s\n", taskID)
			return nil, ErrTaskNotFound
		}
		fmt.Printf("[WORKFLOW] ProcessMessage erro repo: %v\n", err)
		return nil, err
	}
	fmt.Printf("[WORKFLOW] ProcessMessage task encontrada status=%s\n", task.Status)

	// If we're in collecting_data, try to parse key-value data from the message
	if task.Status == models.StatusCollectingData || task.Status == models.StatusValidating {
		fmt.Printf("[WORKFLOW] ProcessMessage coletando dados taskID=%s\n", taskID)
		requirements, err := s.repo.GetRequirements(ctx, taskID)
		if err != nil {
			fmt.Printf("[WORKFLOW] ProcessMessage erro get requirements: %v\n", err)
			return nil, fmt.Errorf("get requirements: %w", err)
		}
		fmt.Printf("[WORKFLOW] ProcessMessage requirements encontradas: %d\n", len(requirements))

		// Try to extract data fields from the message using LLM
		extractedData, err := s.ExtractData(ctx, message, requirements)
		if err != nil {
			fmt.Printf("[WORKFLOW] ProcessMessage erro extract data: %v\n", err)
			slog.Warn("extract data failed", "error", err.Error())
			return &MessageResult{
				Answer: "Não consegui entender os dados informados. Pode reformular?",
			}, nil
		}
		fmt.Printf("[WORKFLOW] ProcessMessage dados extraidos: %v\n", extractedData)

		for field, value := range extractedData {
			if value != "" {
				if err := s.repo.SetData(ctx, taskID, field, value); err != nil {
					fmt.Printf("[WORKFLOW] ProcessMessage erro set data field=%s: %v\n", field, err)
					slog.Warn("failed to set data", "field", field, "error", err.Error())
				} else {
					fmt.Printf("[WORKFLOW] ProcessMessage set data field=%s value=%s\n", field, value)
				}
			}
		}

		// Check for missing fields
		missing, err := s.GetMissingFields(ctx, taskID)
		if err != nil {
			fmt.Printf("[WORKFLOW] ProcessMessage erro get missing fields: %v\n", err)
			return nil, err
		}
		fmt.Printf("[WORKFLOW] ProcessMessage missing fields: %d\n", len(missing))

		if len(missing) > 0 {
			result := &MessageResult{
				Answer: fmt.Sprintf("Recebi os dados. Ainda preciso de: %s", formatMissingFieldNames(missing)),
			}
			return result, nil
		}

		// All required fields are filled, move to validation
		s.repo.UpdateTaskStatus(ctx, taskID, models.StatusValidating, task.TemplateID)
		fmt.Println("[WORKFLOW] ProcessMessage todos campos preenchidos, indo para validacao")
		return s.ValidateAndProceed(ctx, taskID)
	}

	// Default: return the message as-is
	fmt.Println("[WORKFLOW] ProcessMessage retornando mensagem como esta")
	return &MessageResult{Answer: message}, nil
}

// ExtractData uses the LLM to extract field values from a natural language message.
func (s *WorkflowService) ExtractData(ctx context.Context, message string, requirements []models.WorkflowRequirement) (map[string]string, error) {
	if s.groq == nil || s.groq.Empty() {
		return nil, errors.New("groq client not configured")
	}

	var reqNames []string
	for _, r := range requirements {
		reqNames = append(reqNames, r.Name)
	}

	prompt := fmt.Sprintf(`
Extraia os valores dos seguintes campos da mensagem do usuário.
Campos possíveis: %s

Mensagem: "%s"

Retorne JSON no formato: {"campo": "valor"}
Se um campo não estiver presente na mensagem, não o inclua.
Preencha apenas com informações explícitas na mensagem.`, strings.Join(reqNames, ", "), message)

	resp, err := s.groq.Chat(ctx, []groq.Message{
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

// ValidateAndProceed validates the task data and either generates the document or reports issues.
func (s *WorkflowService) ValidateAndProceed(ctx context.Context, taskID string) (*MessageResult, error) {
	fmt.Printf("[WORKFLOW] ValidateAndProceed taskID=%s\n", taskID)
	task, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		fmt.Printf("[WORKFLOW] ValidateAndProceed erro get task: %v\n", err)
		return nil, err
	}
	fmt.Printf("[WORKFLOW] ValidateAndProceed task encontrada status=%s\n", task.Status)

	// Validate all required fields are present
	missing, err := s.GetMissingFields(ctx, taskID)
	if err != nil {
		fmt.Printf("[WORKFLOW] ValidateAndProceed erro get missing: %v\n", err)
		return nil, err
	}
	fmt.Printf("[WORKFLOW] ValidateAndProceed missing fields: %d\n", len(missing))

	if len(missing) > 0 {
		fmt.Printf("[WORKFLOW] ValidateAndProceed campos faltando: %v\n", formatMissingFieldNames(missing))
		return &MessageResult{
			Answer: fmt.Sprintf("Os seguintes campos são obrigatórios e estão faltando: %s", formatMissingFieldNames(missing)),
		}, nil
	}

	// Check deterministic rules
	taskData, err := s.repo.GetData(ctx, taskID)
	if err != nil {
		fmt.Printf("[WORKFLOW] ValidateAndProceed erro get data: %v\n", err)
		return nil, err
	}
	fmt.Printf("[WORKFLOW] ValidateAndProceed data encontrada: %d campos\n", len(taskData))

	// Evaluate value-based rules
	rules := s.evaluateRules(taskData, task.Procedure)
	fmt.Printf("[WORKFLOW] ValidateAndProceed rules avaliadas: %d\n", len(rules))
	for _, rule := range rules {
		if !rule.Passed {
			fmt.Printf("[WORKFLOW] ValidateAndProceed regra falhou: %s\n", rule.Description)
			return &MessageResult{
				Answer: fmt.Sprintf("Validação falhou: %s", rule.Description),
			}, nil
		}
	}
	fmt.Println("[WORKFLOW] ValidateAndProceed todas regras passaram")

	// Move to ready_to_generate
	if err := s.repo.UpdateTaskStatus(ctx, taskID, models.StatusReadyToGenerate, task.TemplateID); err != nil {
		fmt.Printf("[WORKFLOW] ValidateAndProceed erro update status: %v\n", err)
		return nil, err
	}
	fmt.Println("[WORKFLOW] ValidateAndProceed status atualizado para ready_to_generate")

	return &MessageResult{
		Answer: "Todos os dados foram validados. O documento está pronto para geração.",
		ReadyToGenerate: true,
	}, nil
}

// evaluateRules applies deterministic rules from the normative base.
func (s *WorkflowService) evaluateRules(data map[string]string, procedure string) []models.RuleEvaluation {
	var rules []models.RuleEvaluation

	// Check for value field
	valStr := data["valor"]
	if valStr != "" {
		// Try to extract numeric value
		valStr = strings.ReplaceAll(valStr, "R$", "")
		valStr = strings.ReplaceAll(valStr, ".", "")
		valStr = strings.ReplaceAll(valStr, ",", ".")
		// Remove any remaining non-numeric chars except period
		var numStr strings.Builder
		for _, r := range valStr {
			if (r >= '0' && r <= '9') || r == '.' {
				numStr.WriteRune(r)
			}
		}
		cleaned := strings.TrimSpace(numStr.String())
		if cleaned == "" {
			return rules
		}
		val, err := strconv.ParseFloat(cleaned, 64)
		if err != nil {
			return rules
		}

		// Check for approval threshold rules from normatives
		// We search for "limite" or "teto" or "máximo" in the chunks
		// This is done via the RAG search - we check specific thresholds
		// that are explicitly stated in the normatives

		// For now, we just record what we found
		rules = append(rules, models.RuleEvaluation{
			RuleName:    "valor_informado",
			Passed:      true,
			Description: fmt.Sprintf("Valor de R$ %.2f foi informado", val),
			Source:      "dados do funcionário",
		})
	}

	return rules
}

// GenerateDocument generates DOCX and PDF from the template.
// Uses the shared GenerateDocumentFromTemplate to avoid code duplication.
func (s *WorkflowService) GenerateDocument(ctx context.Context, taskID string) (*models.DocumentRun, error) {
	fmt.Printf("[WORKFLOW] GenerateDocument taskID=%s\n", taskID)
	task, err := s.repo.GetTask(ctx, taskID)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Printf("[WORKFLOW] GenerateDocument task nao encontrada taskID=%s\n", taskID)
			return nil, ErrTaskNotFound
		}
		fmt.Printf("[WORKFLOW] GenerateDocument erro repo: %v\n", err)
		return nil, err
	}
	fmt.Printf("[WORKFLOW] GenerateDocument task encontrada status=%s templateID=%s\n", task.Status, task.TemplateID)

	if task.Status != models.StatusReadyToGenerate {
		fmt.Printf("[WORKFLOW] GenerateDocument task nao pronta status=%s\n", task.Status)
		return nil, fmt.Errorf("task not ready for generation: current status %s", task.Status)
	}

	if task.TemplateID == "" {
		fmt.Println("[WORKFLOW] GenerateDocument templateID vazio")
		return nil, ErrTemplateNotFound
	}

	taskData, err := s.repo.GetData(ctx, taskID)
	if err != nil {
		fmt.Printf("[WORKFLOW] GenerateDocument erro get data: %v\n", err)
		return nil, fmt.Errorf("get data: %w", err)
	}
	fmt.Printf("[WORKFLOW] GenerateDocument data encontrada: %d campos\n", len(taskData))

	// Validate template
	fields, err := s.repo.GetTemplateFields(ctx, task.TemplateID)
	if err != nil {
		fmt.Printf("[WORKFLOW] GenerateDocument erro get template fields: %v\n", err)
		return nil, fmt.Errorf("get template fields: %w", err)
	}
	fmt.Printf("[WORKFLOW] GenerateDocument template fields: %d\n", len(fields))

	// Check all required fields
	for _, f := range fields {
		if f.Required {
			if val, exists := taskData[f.FieldName]; !exists || strings.TrimSpace(val) == "" {
				fmt.Printf("[WORKFLOW] GenerateDocument campo obrigatorio faltando: %s\n", f.FieldName)
				return nil, fmt.Errorf("missing required field: %s", f.FieldName)
			}
		}
	}
	fmt.Println("[WORKFLOW] GenerateDocument todos campos obrigatorios preenchidos")

	// Get requirements for source tracking
	requirements, _ := s.repo.GetRequirements(ctx, taskID)
	fmt.Printf("[WORKFLOW] GenerateDocument requirements para source tracking: %d\n", len(requirements))

	// Use shared generation logic
	run, err := s.GenerateDocumentFromTemplate(ctx, taskID, task.TemplateID, task.EmployeeID, taskData, requirements)
	if err != nil {
		fmt.Printf("[WORKFLOW] GenerateDocument erro generate: %v\n", err)
		return nil, err
	}

	// Update task status
	if err := s.repo.UpdateTaskStatus(ctx, taskID, models.StatusGenerated, task.TemplateID); err != nil {
		fmt.Printf("[WORKFLOW] GenerateDocument erro update task status: %v\n", err)
		return nil, fmt.Errorf("update task status: %w", err)
	}
	fmt.Printf("[WORKFLOW] GenerateDocument sucesso runID=%s\n", run.ID)

	return run, nil
}

// GetDocumentsByEmployee returns all document runs for an employee.
func (s *WorkflowService) GetDocumentsByEmployee(ctx context.Context, employeeID string) ([]models.DocumentRun, error) {
	// We need to join with tasks to get all runs for an employee
	rows, err := s.repo.GetDocumentRunByEmployee(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// GetTaskSources returns the search hits that were used as sources for a task.
func (s *WorkflowService) GetTaskSources(ctx context.Context, taskID string) ([]knowledge.SearchHit, error) {
	// Re-run the search based on the task's procedure
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
	fmt.Printf("[WORKFLOW] GetHistory employeeID=%s\n", employeeID)
	items, err := s.repo.GetHistory(ctx, employeeID)
	if err != nil {
		fmt.Printf("[WORKFLOW] GetHistory erro: %v\n", err)
		return nil, err
	}
	fmt.Printf("[WORKFLOW] GetHistory sucesso count=%d\n", len(items))
	return items, nil
}

// ListTemplates returns all document templates.
func (s *WorkflowService) ListTemplates(ctx context.Context) ([]models.DocumentTemplate, error) {
	fmt.Printf("[WORKFLOW] ListTemplates\n")
	return s.repo.ListTemplates(ctx)
}

// HandleDocumentRequest is the unified document generation pipeline.
// It creates a task, classifies intent, searches norms, extracts requirements,
// selects a compatible template, checks for missing fields, and either
// generates the document or reports what is missing.
//
// This is the single entry point used by both the chat endpoint and the
// workflow task endpoint. It never picks an arbitrary template and never
// invents default values for missing fields.
func (s *WorkflowService) HandleDocumentRequest(ctx context.Context, employeeID, question string) (*DocumentRequestResult, error) {
	fmt.Printf("[WORKFLOW] HandleDocumentRequest employeeID=%s question=%s\n", employeeID, question)

	// 1. Create task and classify intent
	taskID, err := s.CreateTask(ctx, employeeID, question)
	if err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}

	// 2. Start processing: search norms, extract requirements, select template
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

	// 3. Check for missing required fields
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

	// 4. All required fields present — generate the document
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

// GenerateDocumentFromTemplate generates a document from a template with provided data.
// This is the shared generation logic used by both the chat and workflow paths.
// It validates that output files exist and have non-zero size before reporting success.
func (s *WorkflowService) GenerateDocumentFromTemplate(ctx context.Context, taskID, templateID, employeeID string, data map[string]string, requirements []models.WorkflowRequirement) (*models.DocumentRun, error) {
	fmt.Printf("[WORKFLOW] GenerateDocumentFromTemplate taskID=%s templateID=%s employeeID=%s\n", taskID, templateID, employeeID)

	template, err := s.repo.GetTemplate(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("template not found: %w", err)
	}

	// Read template
	raw, err := document.ReadTextTemplate(template.TemplatePath)
	if err != nil {
		return nil, fmt.Errorf("read template: %w", err)
	}

	// Apply placeholders
	filled := document.ApplyPlaceholders(raw, data)

	// Generate DOCX
	docxBytes, err := document.GenerateDOCX(template.Name, filled)
	if err != nil {
		return nil, fmt.Errorf("generate docx: %w", err)
	}

	// Generate PDF
	pdfBytes, err := document.GeneratePDF(template.Name, filled)
	if err != nil {
		return nil, fmt.Errorf("generate pdf: %w", err)
	}

	// Save files
	runID := "run-" + uuid.NewString()
	outputDir := filepath.Join("data", "documents", runID)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}

	docxPath := filepath.Join(outputDir, "document.docx")
	if err := os.WriteFile(docxPath, docxBytes, 0o644); err != nil {
		return nil, fmt.Errorf("write docx: %w", err)
	}

	pdfPath := filepath.Join(outputDir, "document.pdf")
	if err := os.WriteFile(pdfPath, pdfBytes, 0o644); err != nil {
		return nil, fmt.Errorf("write pdf: %w", err)
	}

	// Validate files exist and have non-zero size
	docxInfo, err := os.Stat(docxPath)
	if err != nil {
		return nil, fmt.Errorf("docx file not found: %w", err)
	}
	if docxInfo.Size() == 0 {
		return nil, fmt.Errorf("docx file is empty")
	}

	pdfInfo, err := os.Stat(pdfPath)
	if err != nil {
		return nil, fmt.Errorf("pdf file not found: %w", err)
	}
	if pdfInfo.Size() == 0 {
		return nil, fmt.Errorf("pdf file is empty")
	}

	// Create document run
	run := &models.DocumentRun{
		ID:              runID,
		TaskID:          taskID,
		EmployeeID:      employeeID,
		TemplateID:      template.ID,
		TemplateVersion: template.Version,
		Status:          models.RunStatusPending,
		DocxPath:        docxPath,
		PdfPath:         pdfPath,
	}

	if err := s.repo.CreateDocumentRun(ctx, *run); err != nil {
		return nil, fmt.Errorf("create document run: %w", err)
	}

	// Record sources
	for _, req := range requirements {
		ds := models.DocumentSource{
			ID:                "ds-" + uuid.NewString(),
			DocumentRunID:     runID,
			NormativeDocument: req.SourceDocument,
			NormativeSnippet:  req.SourceSnippet,
			Requirement:       req.Name,
		}
		if req.SourceChunkID != nil {
			ds.NormativeChunkID = req.SourceChunkID
		}
		s.repo.CreateDocumentSource(ctx, ds)
	}
	fmt.Printf("[WORKFLOW] GenerateDocumentFromTemplate fontes registradas: %d\n", len(requirements))

	fmt.Printf("[WORKFLOW] GenerateDocumentFromTemplate sucesso runID=%s\n", runID)
	return run, nil
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
