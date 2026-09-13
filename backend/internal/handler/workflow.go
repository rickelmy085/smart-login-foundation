package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bsmart/abis/internal/models"
	"github.com/bsmart/abis/internal/repository"
	"github.com/bsmart/abis/internal/service"
)

// WorkflowHandler exposes task/workflow endpoints.
type WorkflowHandler struct {
	svc          *service.WorkflowService
	repo         *repository.WorkflowRepo
	documentsDir string
}

func NewWorkflowHandler(s *service.WorkflowService, repo *repository.WorkflowRepo) *WorkflowHandler {
	return &WorkflowHandler{svc: s, repo: repo, documentsDir: "data/documents"}
}

// Request bodies

type CreateTaskRequest struct {
	Question string `json:"question"`
}

type ProcessMessageRequest struct {
	Message string `json:"message"`
}

type SetDataRequest struct {
	FieldName string `json:"fieldName"`
	Value     string `json:"value"`
}

// Response types

type TaskResponse struct {
	ID            string                             `json:"id"`
	Intent        string                             `json:"intent"`
	Procedure     string                             `json:"procedure"`
	Status        string                             `json:"status"`
	OriginalRequest string                          `json:"originalRequest"`
	TemplateID    string                             `json:"templateId"`
	Requirements  []RequirementResponse              `json:"requirements"`
	Data          map[string]string                  `json:"data"`
	Template      *TemplateResponse                  `json:"template,omitempty"`
	MissingFields []service.MissingField             `json:"missingFields,omitempty"`
	CreatedAt     string                             `json:"createdAt"`
	UpdatedAt     string                             `json:"updatedAt"`
}

type RequirementResponse struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Label          string  `json:"label"`
	Required       bool    `json:"required"`
	SourceDocument string  `json:"sourceDocument,omitempty"`
	SourceSnippet  string  `json:"sourceSnippet,omitempty"`
}

type TemplateResponse struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	DocumentType string                 `json:"documentType"`
	Version      string                 `json:"version"`
	Fields       []TemplateFieldResponse `json:"fields"`
}

type TemplateFieldResponse struct {
	FieldName        string  `json:"fieldName"`
	Label            string  `json:"label"`
	Type             string  `json:"type"`
	Required         bool    `json:"required"`
	NormativeDocument string `json:"normativeDocument,omitempty"`
}

type CreateTaskResponse struct {
	TaskID  string `json:"taskId"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type ProcessMessageResponse struct {
	Answer      string                 `json:"answer"`
	TaskStatus  string                 `json:"taskStatus"`
	ReadyToGen  bool                   `json:"readyToGenerate"`
	MissingFields []service.MissingField `json:"missingFields,omitempty"`
}

type DocumentRunResponse struct {
	ID            string `json:"id"`
	TemplateID    string `json:"templateId"`
	TemplateVersion string `json:"templateVersion"`
	Status        string `json:"status"`
	DocxPath      string `json:"docxPath,omitempty"`
	PdfPath       string `json:"pdfPath,omitempty"`
	CreatedAt     string `json:"createdAt"`
}

// POST /api/tasks
func (h *WorkflowHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	fmt.Println("[WORKFLOW] CreateTask iniciada")
	var body CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		fmt.Printf("[WORKFLOW] CreateTask erro decode JSON: %v\n", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	body.Question = strings.TrimSpace(body.Question)
	if body.Question == "" {
		fmt.Println("[WORKFLOW] CreateTask question vazia")
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "question is required"})
		return
	}

	employeeID := r.Context().Value("employee_id").(string)
	fmt.Printf("[WORKFLOW] CreateTask employeeID=%s question=%s\n", employeeID, body.Question)

	taskID, err := h.svc.CreateTask(r.Context(), employeeID, body.Question)
	if err != nil {
		fmt.Printf("[WORKFLOW] CreateTask erro no service: %v\n", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	fmt.Printf("[WORKFLOW] CreateTask sucesso taskID=%s\n", taskID)

	writeJSON(w, http.StatusOK, CreateTaskResponse{
		TaskID:  taskID,
		Status:  string(models.StatusDetected),
		Message: "Tarefa criada. Use POST /api/tasks/{id}/process para iniciar o processamento.",
	})
}

// GET /api/tasks/{id}
func (h *WorkflowHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	fmt.Printf("[WORKFLOW] GetTask taskID=%s\n", taskID)
	if taskID == "" {
		fmt.Println("[WORKFLOW] GetTask taskID vazio")
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "task id is required"})
		return
	}

	employeeID := r.Context().Value("employee_id").(string)
	task, err := h.svc.GetTask(r.Context(), taskID)
	if err != nil {
		fmt.Printf("[WORKFLOW] GetTask erro: %v\n", err)
		if err == service.ErrTaskNotFound {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Ownership check
	if task.Task.EmployeeID != employeeID {
		fmt.Printf("[WORKFLOW] GetTask ownership denied: task employee=%s, request employee=%s\n", task.Task.EmployeeID, employeeID)
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
		return
	}

	fmt.Printf("[WORKFLOW] GetTask sucesso taskID=%s status=%s\n", taskID, task.Task.Status)

	resp := buildTaskResponse(task)
	writeJSON(w, http.StatusOK, resp)
}

// GET /api/tasks
func (h *WorkflowHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	employeeID := r.Context().Value("employee_id").(string)
	fmt.Printf("[WORKFLOW] ListTasks employeeID=%s\n", employeeID)

	tasks, err := h.repo.GetTasksByEmployee(r.Context(), employeeID)
	if err != nil {
		fmt.Printf("[WORKFLOW] ListTasks erro: %v\n", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	fmt.Printf("[WORKFLOW] ListTasks sucesso: %d tarefas encontradas\n", len(tasks))

	resp := make([]TaskResponse, 0, len(tasks))
	for _, t := range tasks {
		// Build minimal response
		resp = append(resp, TaskResponse{
			ID:            t.ID,
			Intent:        string(t.Intent),
			Procedure:     t.Procedure,
			Status:        string(t.Status),
			OriginalRequest: t.OriginalRequest,
			TemplateID:    t.TemplateID,
			CreatedAt:     t.CreatedAt,
			UpdatedAt:     t.UpdatedAt,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"tasks": resp})
}

// POST /api/tasks/{id}/process
func (h *WorkflowHandler) ProcessTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	fmt.Printf("[WORKFLOW] ProcessTask taskID=%s\n", taskID)
	if taskID == "" {
		fmt.Println("[WORKFLOW] ProcessTask taskID vazio")
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "task id is required"})
		return
	}

	employeeID := r.Context().Value("employee_id").(string)
	task, err := h.svc.GetTask(r.Context(), taskID)
	if err != nil {
		fmt.Printf("[WORKFLOW] ProcessTask erro get task: %v\n", err)
		if err == service.ErrTaskNotFound {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if task.Task.EmployeeID != employeeID {
		fmt.Printf("[WORKFLOW] ProcessTask ownership denied: task employee=%s, request employee=%s\n", task.Task.EmployeeID, employeeID)
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
		return
	}

	result, err := h.svc.StartProcessing(r.Context(), taskID)
	if err != nil {
		fmt.Printf("[WORKFLOW] ProcessTask erro: %v\n", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	fmt.Printf("[WORKFLOW] ProcessTask sucesso taskID=%s answered=%v blocked=%v\n", taskID, result.Answered, result.Blocked)

	resp := map[string]any{
		"taskId":         taskID,
		"requirements":   result.Requirements,
		"readyToGenerate": false,
	}

	if result.Answered {
		resp["message"] = result.Message
		resp["answered"] = true
	}
	if result.Blocked {
		resp["blocked"] = true
		resp["message"] = result.Message
	}
	if result.Template != nil {
		resp["template"] = buildTemplateResponse(*result.Template, nil)
	}

	// Convert sources to simple response
	sources := make([]map[string]any, 0, len(result.Sources))
	for _, hit := range result.Sources {
		sources = append(sources, map[string]any{
			"title":       hit.Document.Title,
			"snippet":     hit.Snippet,
			"chunkOrd":    hit.Chunk.Ord,
			"score":       hit.Score,
			"source":      hit.Document.SourcePath,
		})
	}
	resp["sources"] = sources

	writeJSON(w, http.StatusOK, resp)
}

// POST /api/tasks/{id}/message
func (h *WorkflowHandler) ProcessMessage(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	fmt.Printf("[WORKFLOW] ProcessMessage taskID=%s\n", taskID)
	if taskID == "" {
		fmt.Println("[WORKFLOW] ProcessMessage taskID vazio")
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "task id is required"})
		return
	}

	employeeID := r.Context().Value("employee_id").(string)
	task, err := h.svc.GetTask(r.Context(), taskID)
	if err != nil {
		if err == service.ErrTaskNotFound {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if task.Task.EmployeeID != employeeID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
		return
	}

	var body ProcessMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		fmt.Printf("[WORKFLOW] ProcessMessage erro decode JSON: %v\n", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	body.Message = strings.TrimSpace(body.Message)
	if body.Message == "" {
		fmt.Println("[WORKFLOW] ProcessMessage message vazia")
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "message is required"})
		return
	}
	fmt.Printf("[WORKFLOW] ProcessMessage taskID=%s message=%s\n", taskID, body.Message)

	result, err := h.svc.ProcessMessage(r.Context(), taskID, body.Message)
	if err != nil {
		fmt.Printf("[WORKFLOW] ProcessMessage erro: %v\n", err)
		status := http.StatusInternalServerError
		if err == service.ErrTaskNotFound {
			status = http.StatusNotFound
		} else if err == service.ErrMissingData {
			status = http.StatusBadRequest
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	fmt.Printf("[WORKFLOW] ProcessMessage sucesso readyToGenerate=%v\n", result.ReadyToGenerate)

	// Get updated missing fields
	missing, _ := h.svc.GetMissingFields(r.Context(), taskID)

	writeJSON(w, http.StatusOK, ProcessMessageResponse{
		Answer:       result.Answer,
		TaskStatus:   "",
		ReadyToGen:   result.ReadyToGenerate,
		MissingFields: missing,
	})
}

// POST /api/tasks/{id}/data
func (h *WorkflowHandler) SetData(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	fmt.Printf("[WORKFLOW] SetData taskID=%s\n", taskID)
	if taskID == "" {
		fmt.Println("[WORKFLOW] SetData taskID vazio")
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "task id is required"})
		return
	}

	employeeID := r.Context().Value("employee_id").(string)
	task, err := h.svc.GetTask(r.Context(), taskID)
	if err != nil {
		if err == service.ErrTaskNotFound {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if task.Task.EmployeeID != employeeID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
		return
	}

	var body SetDataRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		fmt.Printf("[WORKFLOW] SetData erro decode JSON: %v\n", err)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	fmt.Printf("[WORKFLOW] SetData taskID=%s field=%s value=%s\n", taskID, body.FieldName, body.Value)

	if err := h.svc.SetData(r.Context(), taskID, body.FieldName, body.Value); err != nil {
		fmt.Printf("[WORKFLOW] SetData erro: %v\n", err)
		status := http.StatusInternalServerError
		if err == service.ErrTaskNotFound {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	fmt.Println("[WORKFLOW] SetData sucesso")

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// POST /api/tasks/{id}/validate
func (h *WorkflowHandler) ValidateTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	fmt.Printf("[WORKFLOW] ValidateTask taskID=%s\n", taskID)
	if taskID == "" {
		fmt.Println("[WORKFLOW] ValidateTask taskID vazio")
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "task id is required"})
		return
	}

	employeeID := r.Context().Value("employee_id").(string)
	task, err := h.svc.GetTask(r.Context(), taskID)
	if err != nil {
		if err == service.ErrTaskNotFound {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if task.Task.EmployeeID != employeeID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
		return
	}

	result, err := h.svc.ValidateAndProceed(r.Context(), taskID)
	if err != nil {
		fmt.Printf("[WORKFLOW] ValidateTask erro: %v\n", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	fmt.Printf("[WORKFLOW] ValidateTask sucesso readyToGenerate=%v\n", result.ReadyToGenerate)

	writeJSON(w, http.StatusOK, ProcessMessageResponse{
		Answer:       result.Answer,
		ReadyToGen:  result.ReadyToGenerate,
		MissingFields: result.MissingFields,
	})
}

// POST /api/tasks/{id}/generate
func (h *WorkflowHandler) GenerateDocument(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	fmt.Printf("[WORKFLOW] GenerateDocument taskID=%s\n", taskID)
	if taskID == "" {
		fmt.Println("[WORKFLOW] GenerateDocument taskID vazio")
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "task id is required"})
		return
	}

	employeeID := r.Context().Value("employee_id").(string)
	task, err := h.svc.GetTask(r.Context(), taskID)
	if err != nil {
		if err == service.ErrTaskNotFound {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if task.Task.EmployeeID != employeeID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
		return
	}

	run, err := h.svc.GenerateDocument(r.Context(), taskID)
	if err != nil {
		fmt.Printf("[WORKFLOW] GenerateDocument erro: %v\n", err)
		status := http.StatusInternalServerError
		if err == service.ErrTaskNotFound || err == service.ErrTemplateNotFound {
			status = http.StatusNotFound
		} else if err == service.ErrMissingData || err == service.ErrValidationFailed {
			status = http.StatusBadRequest
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}
	fmt.Printf("[WORKFLOW] GenerateDocument sucesso runID=%s status=%s docx=%s\n", run.ID, run.Status, run.DocxPath)

	var docxURL string
	if run.DocxPath != "" {
		docxURL = "/api/documents/" + run.ID + "/docx"
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"documentRunId": run.ID,
		"status":        string(run.Status),
		"docxPath":      docxURL,
		"pdfPath":       "",
	})
}

// GET /api/tasks/{id}/sources
func (h *WorkflowHandler) GetTaskSources(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	fmt.Printf("[WORKFLOW] GetTaskSources taskID=%s\n", taskID)
	if taskID == "" {
		fmt.Println("[WORKFLOW] GetTaskSources taskID vazio")
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "task id is required"})
		return
	}

	employeeID := r.Context().Value("employee_id").(string)
	task, err := h.svc.GetTask(r.Context(), taskID)
	if err != nil {
		if err == service.ErrTaskNotFound {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if task.Task.EmployeeID != employeeID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
		return
	}

	hits, err := h.svc.GetTaskSources(r.Context(), taskID)
	if err != nil {
		fmt.Printf("[WORKFLOW] GetTaskSources erro: %v\n", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	fmt.Printf("[WORKFLOW] GetTaskSources sucesso: %d fontes encontradas\n", len(hits))

	sources := make([]map[string]any, 0, len(hits))
	for _, hit := range hits {
		sources = append(sources, map[string]any{
			"title":       hit.Document.Title,
			"snippet":     hit.Snippet,
			"chunkOrd":    hit.Chunk.Ord,
			"score":       hit.Score,
			"source":      hit.Document.SourcePath,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"sources": sources})
}

// GET /api/documents
func (h *WorkflowHandler) ListDocuments(w http.ResponseWriter, r *http.Request) {
	employeeID := r.Context().Value("employee_id").(string)
	fmt.Printf("[WORKFLOW] ListDocuments employeeID=%s\n", employeeID)

	runs, err := h.svc.GetDocumentsByEmployee(r.Context(), employeeID)
	if err != nil {
		fmt.Printf("[WORKFLOW] ListDocuments erro: %v\n", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	fmt.Printf("[WORKFLOW] ListDocuments sucesso: %d documentos encontrados\n", len(runs))

	resp := make([]DocumentRunResponse, 0, len(runs))
	for _, run := range runs {
		resp = append(resp, DocumentRunResponse{
			ID:              run.ID,
			TemplateID:      run.TemplateID,
			TemplateVersion: run.TemplateVersion,
			Status:          string(run.Status),
			DocxPath:        run.DocxPath,
			PdfPath:         run.PdfPath,
			CreatedAt:       run.CreatedAt,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"documents": resp})
}

// GET /api/documents/{id}/sources
func (h *WorkflowHandler) GetDocumentSources(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	fmt.Printf("[WORKFLOW] GetDocumentSources runID=%s\n", runID)
	if runID == "" {
		fmt.Println("[WORKFLOW] GetDocumentSources runID vazio")
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "document id is required"})
		return
	}

	sources, err := h.svc.GetDocumentSources(r.Context(), runID)
	if err != nil {
		fmt.Printf("[WORKFLOW] GetDocumentSources erro: %v\n", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	fmt.Printf("[WORKFLOW] GetDocumentSources sucesso: %d fontes encontradas\n", len(sources))

	resp := make([]map[string]any, 0, len(sources))
	for _, ds := range sources {
		resp = append(resp, map[string]any{
			"normativeDocument": ds.NormativeDocument,
			"snippet":           ds.NormativeSnippet,
			"requirement":       ds.Requirement,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"sources": resp})
}

// GET /api/documents/{id}/docx
func (h *WorkflowHandler) DownloadDocx(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	if runID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "document id is required"})
		return
	}

	employeeID, ok := r.Context().Value("employee_id").(string)
	if !ok || employeeID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	run, err := h.svc.GetDocumentRun(r.Context(), runID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "document not found"})
		return
	}

	if run.EmployeeID != employeeID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "document not found"})
		return
	}

	data, err := validateAndReadFile(run.DocxPath, h.documentsDir)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to read document"})
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="ABIS-documento-%s.docx"`, runID))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// GET /api/documents/{id}/pdf
func (h *WorkflowHandler) DownloadPdf(w http.ResponseWriter, r *http.Request) {
	runID := r.PathValue("id")
	if runID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "document id is required"})
		return
	}

	employeeID, ok := r.Context().Value("employee_id").(string)
	if !ok || employeeID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	run, err := h.svc.GetDocumentRun(r.Context(), runID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "document not found"})
		return
	}

	if run.EmployeeID != employeeID {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "document not found"})
		return
	}

	var pdfPath string
	if run.PdfPath != "" {
		pdfPath = run.PdfPath
	} else {
		docxPath := run.DocxPath
		pdfPath = strings.TrimSuffix(docxPath, ".docx") + ".pdf"
	}

	data, err := validateAndReadFile(pdfPath, h.documentsDir)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to read pdf"})
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="ABIS-documento-%s.pdf"`, runID))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func validateAndReadFile(path, baseDir string) ([]byte, error) {
	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		return nil, fmt.Errorf("invalid path: path traversal detected")
	}
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return nil, err
	}
	absPath = filepath.Clean(absPath)
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, err
	}
	absBase = filepath.Clean(absBase)
	if !strings.HasPrefix(absPath, absBase+string(filepath.Separator)) && absPath != absBase {
		return nil, fmt.Errorf("path outside allowed directory")
	}
	return os.ReadFile(absPath)
}

func buildTaskResponse(task *service.TaskDetail) TaskResponse {
	reqs := make([]RequirementResponse, 0, len(task.Requirements))
	for _, r := range task.Requirements {
		reqs = append(reqs, RequirementResponse{
			ID:             r.ID,
			Name:           r.Name,
			Label:          r.Label,
			Required:       r.Required,
			SourceDocument: r.SourceDocument,
			SourceSnippet:  r.SourceSnippet,
		})
	}

	var tpl *TemplateResponse
	if task.Template != nil {
		// Need to get fields too
		tpl = &TemplateResponse{
			ID:           task.Template.ID,
			Name:         task.Template.Name,
			Description:  task.Template.Description,
			DocumentType: task.Template.DocumentType,
			Version:      task.Template.Version,
		}
	}

	return TaskResponse{
		ID:            task.Task.ID,
		Intent:        string(task.Task.Intent),
		Procedure:     task.Task.Procedure,
		Status:        string(task.Task.Status),
		OriginalRequest: task.Task.OriginalRequest,
		TemplateID:    task.Task.TemplateID,
		Requirements:  reqs,
		Data:          task.Data,
		Template:      tpl,
		MissingFields: task.MissingFields,
		CreatedAt:     task.Task.CreatedAt,
		UpdatedAt:     task.Task.UpdatedAt,
	}
}

func buildTemplateResponse(tmpl models.DocumentTemplate, fields []models.TemplateField) TemplateResponse {
	fieldResponses := make([]TemplateFieldResponse, 0, len(fields))
	for _, f := range fields {
		fieldResponses = append(fieldResponses, TemplateFieldResponse{
			FieldName:        f.FieldName,
			Label:            f.Label,
			Type:             string(f.Type),
			Required:         f.Required,
			NormativeDocument: f.NormativeDocument,
		})
	}
	return TemplateResponse{
		ID:           tmpl.ID,
		Name:         tmpl.Name,
		Description:  tmpl.Description,
		DocumentType: tmpl.DocumentType,
		Version:      tmpl.Version,
		Fields:       fieldResponses,
	}
}

// GET /api/history
func (h *WorkflowHandler) ListHistory(w http.ResponseWriter, r *http.Request) {
	employeeID := r.Context().Value("employee_id").(string)
	fmt.Printf("[WORKFLOW] ListHistory employeeID=%s\n", employeeID)

	items, err := h.svc.GetHistory(r.Context(), employeeID)
	if err != nil {
		fmt.Printf("[WORKFLOW] ListHistory erro: %v\n", err)
		writeJSON(w, http.StatusOK, map[string]any{"history": []models.HistoryItem{}})
		return
	}
	fmt.Printf("[WORKFLOW] ListHistory sucesso: %d items\n", len(items))

	writeJSON(w, http.StatusOK, map[string]any{"history": items})
}
