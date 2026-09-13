package docspec

type DocumentSpec struct {
	SpecID          string            `json:"spec_id"`
	RunID           string            `json:"run_id"`
	TaskID          string            `json:"task_id"`
	DocumentType    string            `json:"document_type"`
	TemplateKey     string            `json:"template_key"`
	TemplateVersion string            `json:"template_version"`
	Title           string            `json:"title"`
	Metadata        DocumentMetadata  `json:"metadata"`
	Fields          map[string]string `json:"fields"`
	Sections        []DocumentSection `json:"sections"`
	Rules           []DocumentRule    `json:"rules"`
	Sources         []DocumentSource  `json:"sources"`
	Output          DocumentOutput    `json:"output"`
}

type DocumentMetadata struct {
	GeneratedAt  string `json:"generated_at,omitempty"`
	EmployeeID   string `json:"employee_id,omitempty"`
	EmployeeName string `json:"employee_name,omitempty"`
	EmployeeRE   string `json:"employee_re,omitempty"`
	Department   string `json:"department,omitempty"`
}

type DocumentSection struct {
	ID      string         `json:"id"`
	Type   string         `json:"type"`
	Content map[string]any `json:"content,omitempty"`
}

type DocumentRule struct {
	RuleID   string     `json:"rule_id,omitempty"`
	Rule     string     `json:"rule"`
	Required bool       `json:"required"`
	Reason   string     `json:"reason,omitempty"`
	Source   *RuleSource `json:"source,omitempty"`
}

type RuleSource struct {
	Document string `json:"document,omitempty"`
	ChunkID  any    `json:"chunk_id,omitempty"`
	Excerpt  string `json:"excerpt,omitempty"`
}

type DocumentSource struct {
	Document    string `json:"document,omitempty"`
	ChunkID     any    `json:"chunk_id,omitempty"`
	Snippet     string `json:"snippet,omitempty"`
	Requirement string `json:"requirement,omitempty"`
}

type DocumentOutput struct {
	Formats        []string `json:"formats"`
	FilenamePrefix string   `json:"filename_prefix"`
}
