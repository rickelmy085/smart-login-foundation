package agent

import (
	"encoding/json"
	"net/http"
)

type AgentHandler struct {
	agentService *AgentService
}

func NewAgentHandler(agentService *AgentService) *AgentHandler {
	return &AgentHandler{agentService: agentService}
}

func (h *AgentHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/agent/process", h.Process)
	mux.HandleFunc("/api/agent/status/", h.GetStatus)
	mux.HandleFunc("/api/agent/tools", h.ListTools)
	mux.HandleFunc("/api/agent/tools/", h.GetToolSchema)
	mux.HandleFunc("/api/agent/resume", h.Resume)
	mux.HandleFunc("/api/agent/human-input", h.HumanInput)
}

func getEmployeeID(r *http.Request) (string, bool) {
	id, ok := r.Context().Value("employee_id").(string)
	return id, ok
}

func (h *AgentHandler) Process(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req AgentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	employeeID, ok := getEmployeeID(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	resp, err := h.agentService.Process(r.Context(), req, employeeID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, resp)
}

func (h *AgentHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	planID := r.PathValue("planID")
	if planID == "" {
		h.writeError(w, http.StatusBadRequest, "plan ID required")
		return
	}

	employeeID, ok := getEmployeeID(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	resp, err := h.agentService.GetExecutionStatus(planID, employeeID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, resp)
}

func (h *AgentHandler) ListTools(w http.ResponseWriter, r *http.Request) {
	_, ok := getEmployeeID(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	tools := h.agentService.ListTools()
	h.writeJSON(w, http.StatusOK, map[string]any{"tools": tools})
}

func (h *AgentHandler) GetToolSchema(w http.ResponseWriter, r *http.Request) {
	_, ok := getEmployeeID(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	toolName := r.PathValue("toolName")
	if toolName == "" {
		h.writeError(w, http.StatusBadRequest, "tool name required")
		return
	}

	schema, ok := h.agentService.GetToolSchema(toolName)
	if !ok {
		h.writeError(w, http.StatusNotFound, "tool not found")
		return
	}

	h.writeJSON(w, http.StatusOK, schema)
}

func (h *AgentHandler) Resume(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		PlanID string `json:"plan_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	employeeID, ok := getEmployeeID(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	agentReq := AgentRequest{
		ResumePlanID: req.PlanID,
		Context:      map[string]any{"employee_id": employeeID},
	}

	resp, err := h.agentService.Process(r.Context(), agentReq, employeeID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, resp)
}

func (h *AgentHandler) HumanInput(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		PlanID   string `json:"plan_id"`
		Response string `json:"response"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	employeeID, ok := getEmployeeID(r)
	if !ok {
		h.writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	agentReq := AgentRequest{
		HumanInput: req.Response,
		PlanID:     req.PlanID,
	}

	resp, err := h.agentService.Process(r.Context(), agentReq, employeeID)
	if err != nil {
		h.writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.writeJSON(w, http.StatusOK, resp)
}

func (h *AgentHandler) writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *AgentHandler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]string{"error": message})
}
