package documentengine

type HealthResponse struct {
	Status          string   `json:"status"`
	Service         string   `json:"service"`
	Version         string   `json:"version"`
	TemplatesLoaded int      `json:"templates_loaded"`
	TemplateKeys    []string `json:"template_keys"`
}

type GenerateResponse struct {
	Status       string             `json:"status"`
	SpecID       string             `json:"spec_id"`
	RunID        string             `json:"run_id"`
	TemplateUsed TemplateInfo       `json:"template_used"`
	Files        []FileOutput       `json:"files"`
	Warnings     []GenerationWarning `json:"warnings"`
}

type TemplateInfo struct {
	Key     string `json:"key"`
	Version string `json:"version"`
	Path    string `json:"path"`
}

type FileOutput struct {
	Format     string `json:"format"`
	Filename   string `json:"filename"`
	SizeBytes  int64  `json:"size_bytes"`
	SHA256     string `json:"sha256"`
	DataBase64 string `json:"data_base64"`
}

type GenerationWarning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ErrorDetail struct {
	Status    string `json:"status"`
	ErrorCode string `json:"error_code"`
	Message   string `json:"message"`
	SpecID    string `json:"spec_id"`
	RunID     string `json:"run_id"`
}
