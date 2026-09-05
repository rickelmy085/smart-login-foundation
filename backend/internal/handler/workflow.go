package handler

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/bsmart/abis/internal/models"
	"github.com/bsmart/abis/internal/repository"
	"github.com/bsmart/abis/internal/service"
)

// WorkflowHandler exposes task/workflow endpoints.
type WorkflowHandler struct {
	svc  *service.WorkflowService
	repo *repository.WorkflowRepo
}

func NewWorkflowHandler(s *service.WorkflowService, repo *repository.WorkflowRepo) *WorkflowHandler {
	return &WorkflowHandler{svc: s, repo: repo}
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

type GenerateDocumentResponse struct {
	DocumentRunID string `json:"documentRunId"`
	Status        string `json:"status"`
	DocxPath      string `json:"docxPath"`
	PdfPath       string `json:"pdfPath"`
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
	var body CreateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	body.Question = strings.TrimSpace(body.Question)
	if body.Question == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "question is required"})
		return
	}

	employeeID := r.Context().Value("employee_id").(string)

	taskID, err := h.svc.CreateTask(r.Context(), employeeID, body.Question)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, CreateTaskResponse{
		TaskID:  taskID,
		Status:  string(models.StatusDetected),
		Message: "Tarefa criada. Use POST /api/tasks/{id}/process para iniciar o processamento.",
	})
}

// GET /api/tasks/{id}
func (h *WorkflowHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "task id is required"})
		return
	}

	task, err := h.svc.GetTask(r.Context(), taskID)
	if err != nil {
		if err == service.ErrTaskNotFound {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "task not found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	resp := buildTaskResponse(task)
	writeJSON(w, http.StatusOK, resp)
}

// GET /api/tasks
func (h *WorkflowHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	employeeID := r.Context().Value("employee_id").(string)

	tasks, err := h.repo.GetTasksByEmployee(r.Context(), employeeID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

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
	if taskID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "task id is required"})
		return
	}

	result, err := h.svc.StartProcessing(r.Context(), taskID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

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
	if taskID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "task id is required"})
		return
	}

	var body ProcessMessageRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	body.Message = strings.TrimSpace(body.Message)
	if body.Message == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "message is required"})
		return
	}

	result, err := h.svc.ProcessMessage(r.Context(), taskID, body.Message)
	if err != nil {
		status := http.StatusInternalServerError
		if err == service.ErrTaskNotFound {
			status = http.StatusNotFound
		} else if err == service.ErrMissingData {
			status = http.StatusBadRequest
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

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
	if taskID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "task id is required"})
		return
	}

	var body SetDataRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	if err := h.svc.SetData(r.Context(), taskID, body.FieldName, body.Value); err != nil {
		status := http.StatusInternalServerError
		if err == service.ErrTaskNotFound {
			status = http.StatusNotFound
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// POST /api/tasks/{id}/validate
func (h *WorkflowHandler) ValidateTask(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "task id is required"})
		return
	}

	result, err := h.svc.ValidateAndProceed(r.Context(), taskID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, ProcessMessageResponse{
		Answer:       result.Answer,
		ReadyToGen:  result.ReadyToGenerate,
		MissingFields: result.MissingFields,
	})
}

// POST /api/tasks/{id}/generate
func (h *WorkflowHandler) GenerateDocument(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "task id is required"})
		return
	}

	run, err := h.svc.GenerateDocument(r.Context(), taskID)
	if err != nil {
		status := http.StatusInternalServerError
		if err == service.ErrTaskNotFound || err == service.ErrTemplateNotFound {
			status = http.StatusNotFound
		} else if err == service.ErrMissingData || err == service.ErrValidationFailed {
			status = http.StatusBadRequest
		}
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	var docxURL string
	if run.DocxPath != "" {
		docxURL = "/api/documents/" + filepath.Base(run.DocxPath)
	}

	writeJSON(w, http.StatusOK, GenerateDocumentResponse{
		DocumentRunID: run.ID,
		Status:        string(run.Status),
		DocxPath:      docxURL,
		PdfPath:       "",
	})
}

// GET /api/tasks/{id}/sources
func (h *WorkflowHandler) GetTaskSources(w http.ResponseWriter, r *http.Request) {
	taskID := r.PathValue("id")
	if taskID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "task id is required"})
		return
	}

	hits, err := h.svc.GetTaskSources(r.Context(), taskID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

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

	runs, err := h.svc.GetDocumentsByEmployee(r.Context(), employeeID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

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
	if runID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "document id is required"})
		return
	}

	sources, err := h.svc.GetDocumentSources(r.Context(), runID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

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

// Helper functions

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
