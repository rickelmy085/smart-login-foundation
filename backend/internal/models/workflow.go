package models

// TaskStatus represents the state of a workflow task.
type TaskStatus string

const (
	StatusDetected       TaskStatus = "detected"
	StatusCollectingData TaskStatus = "collecting_data"
	StatusValidating     TaskStatus = "validating"
	StatusReadyToGenerate TaskStatus = "ready_to_generate"
	StatusGenerating     TaskStatus = "generating"
	StatusGenerated      TaskStatus = "generated"
	StatusNeedsReview    TaskStatus = "needs_review"
	StatusCompleted      TaskStatus = "completed"
	StatusBlocked        TaskStatus = "blocked"
)

// DocumentRunStatus represents the status of a generated document.
type DocumentRunStatus string

const (
	RunStatusGenerating DocumentRunStatus = "generating"
	RunStatusGenerated  DocumentRunStatus = "generated"
	RunStatusPending    DocumentRunStatus = "pending_review"
	RunStatusApproved   DocumentRunStatus = "approved"
	RunStatusRejected   DocumentRunStatus = "rejected"
)

// FieldType represents the type of a template field.
type FieldType string

const (
	FieldTypeText      FieldType = "text"
	FieldTypeNumber    FieldType = "number"
	FieldTypeCurrency  FieldType = "currency"
	FieldTypeDate      FieldType = "date"
	FieldTypeBoolean   FieldType = "boolean"
	FieldTypeSelect    FieldType = "select"
	FieldTypeTextarea  FieldType = "textarea"
	FieldTypeEmployee  FieldType = "employee"
	FieldTypeDepartment FieldType = "department"
	FieldTypeCompany   FieldType = "company"
	FieldTypeSupplier  FieldType = "supplier"
)

// Intent represents the classified intent of a user request.
type Intent string

const (
	IntentKnowledgeQuery      Intent = "knowledge_query"
	IntentProcedureQuery      Intent = "procedure_query"
	IntentDocumentGeneration  Intent = "document_generation"
	IntentFormCompletion      Intent = "form_completion"
	IntentApprovalCheck       Intent = "approval_check"
	IntentRequirementCheck    Intent = "requirement_check"
	IntentWorkflowExecution  Intent = "workflow_execution"
)

// WorkflowTask represents a workflow task in the system.
type WorkflowTask struct {
	ID            string     `json:"id" db:"id"`
	EmployeeID    string     `json:"employeeId" db:"employee_id"`
	Intent        Intent     `json:"intent" db:"intent"`
	Procedure     string     `json:"procedure" db:"procedure"`
	Status        TaskStatus `json:"status" db:"status"`
	OriginalRequest string    `json:"originalRequest" db:"original_request"`
	TemplateID    string     `json:"templateId" db:"template_id"`
	CreatedAt     string     `json:"createdAt" db:"created_at"`
	UpdatedAt     string     `json:"updatedAt" db:"updated_at"`
}

// WorkflowRequirement represents a requirement extracted from normatives.
type WorkflowRequirement struct {
	ID             string `json:"id" db:"id"`
	TaskID         string `json:"taskId" db:"task_id"`
	Name           string `json:"name" db:"name"`
	Label          string `json:"label" db:"label"`
	Required       bool   `json:"required" db:"required"`
	SourceDocument string `json:"sourceDocument" db:"source_document"`
	SourceChunkID *int64  `json:"sourceChunkId" db:"source_chunk_id"`
	SourceSnippet  string `json:"sourceSnippet" db:"source_snippet"`
	Rank           int    `json:"rank" db:"rank"`
	CreatedAt      string `json:"createdAt" db:"created_at"`
}

// WorkflowData represents collected data for a task.
type WorkflowData struct {
	ID         string `json:"id" db:"id"`
	TaskID     string `json:"taskId" db:"task_id"`
	FieldName  string `json:"fieldName" db:"field_name"`
	Value      string `json:"value" db:"value"`
	CreatedAt  string `json:"createdAt" db:"created_at"`
	UpdatedAt  string `json:"updatedAt" db:"updated_at"`
}

// DocumentTemplate represents a document template.
type DocumentTemplate struct {
	ID          string `json:"id" db:"id"`
	Name        string `json:"name" db:"name"`
	Description string `json:"description" db:"description"`
	DocumentType string `json:"documentType" db:"document_type"`
	Version     string `json:"version" db:"version"`
	Active      bool   `json:"active" db:"active"`
	TemplatePath string `json:"templatePath" db:"template_path"`
	CreatedAt   string `json:"createdAt" db:"created_at"`
	UpdatedAt   string `json:"updatedAt" db:"updated_at"`
}

// TemplateField represents a field in a template.
type TemplateField struct {
	ID               string     `json:"id" db:"id"`
	TemplateID       string     `json:"templateId" db:"template_id"`
	FieldName        string     `json:"fieldName" db:"field_name"`
	Label            string     `json:"label" db:"label"`
	Type             FieldType  `json:"type" db:"type"`
	Required         bool       `json:"required" db:"required"`
	ValidationRule   string     `json:"validationRule" db:"validation_rule"`
	SourceRequirement string    `json:"sourceRequirement" db:"source_requirement"`
	NormativeDocument string    `json:"normativeDocument" db:"normative_document"`
	NormativeChunkID *int64    `json:"normativeChunkId" db:"normative_chunk_id"`
	CreatedAt        string     `json:"createdAt" db:"created_at"`
}

// DocumentRun represents a generated document.
type DocumentRun struct {
	ID            string           `json:"id" db:"id"`
	TaskID        string           `json:"taskId" db:"task_id"`
	EmployeeID    string           `json:"employeeId" db:"employee_id"`
	TemplateID    string           `json:"templateId" db:"template_id"`
	TemplateVersion string         `json:"templateVersion" db:"template_version"`
	Status        DocumentRunStatus `json:"status" db:"status"`
	DocxPath      string           `json:"docxPath" db:"docx_path"`
	PdfPath       string           `json:"pdfPath" db:"pdf_path"`
	CreatedAt     string           `json:"createdAt" db:"created_at"`
	UpdatedAt     string           `json:"updatedAt" db:"updated_at"`
}

// DocumentSource represents a normative source used in document generation.
type DocumentSource struct {
	ID               string `json:"id" db:"id"`
	DocumentRunID    string `json:"documentRunId" db:"document_run_id"`
	NormativeDocument string `json:"normativeDocument" db:"normative_document"`
	NormativeChunkID *int64 `json:"normativeChunkId" db:"normative_chunk_id"`
	NormativeSnippet  string `json:"normativeSnippet" db:"normative_snippet"`
	Requirement      string `json:"requirement" db:"requirement"`
	CreatedAt        string `json:"createdAt" db:"created_at"`
}

// IntentClassification is the LLM response for intent classification.
type IntentClassification struct {
	Intent      string  `json:"intent"`
	Confidence  float64 `json:"confidence"`
	Procedure   string  `json:"procedure,omitempty"`
	TemplateKey string  `json:"template_key,omitempty"`
}

// RequirementExtraction is the LLM response for requirements.
type RequirementExtraction struct {
	Procedure     string                  `json:"procedure"`
	Summary       string                  `json:"summary"`
	Requirements  []ExtractedRequirement  `json:"requirements"`
}

// ExtractedRequirement is a single requirement extracted by the LLM.
type ExtractedRequirement struct {
	Name           string             `json:"name"`
	Label          string             `json:"label"`
	Required       bool               `json:"required"`
	Type           string             `json:"type,omitempty"`
	SourceDocument string             `json:"source_document"`
	SourceSnippet  string             `json:"source_snippet"`
}

// RuleEvaluation represents the result of evaluating a deterministic rule.
type RuleEvaluation struct {
	RuleName    string `json:"rule_name"`
	Passed      bool   `json:"passed"`
	Description string `json:"description"`
	Source      string `json:"source"`
}
