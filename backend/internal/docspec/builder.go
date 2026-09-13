package docspec

import (
	"strings"
	"time"

	"github.com/bsmart/abis/internal/models"
	"github.com/google/uuid"
)

type BuildInput struct {
	Task         models.WorkflowTask
	Template     models.DocumentTemplate
	Data         map[string]string
	Requirements []models.WorkflowRequirement
	Employee     *models.Employee
	RunID        string
}

func Build(input BuildInput) *DocumentSpec {
	runID := input.RunID
	if runID == "" {
		runID = uuid.NewString()
	}

	spec := &DocumentSpec{
		SpecID:          uuid.NewString(),
		RunID:           runID,
		TaskID:          input.Task.ID,
		DocumentType:    input.Template.DocumentType,
		TemplateKey:     resolveTemplateKey(input.Template),
		TemplateVersion: input.Template.Version,
		Title:           input.Template.Name,
		Metadata: DocumentMetadata{
			GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		},
		Fields:   copyFields(input.Data),
		Sections: []DocumentSection{},
		Rules:    []DocumentRule{},
		Sources:  []DocumentSource{},
		Output: DocumentOutput{
			Formats:        []string{"docx", "pdf"},
			FilenamePrefix: sanitizeFilename(input.Task.ID),
		},
	}

	if input.Employee != nil {
		spec.Metadata.EmployeeID = input.Employee.ID
		spec.Metadata.EmployeeName = input.Employee.Name
		spec.Metadata.EmployeeRE = input.Employee.RE
		spec.Metadata.Department = input.Employee.Role
	}

	for _, req := range input.Requirements {
		src := DocumentSource{
			Document:    req.SourceDocument,
			Snippet:     req.SourceSnippet,
			Requirement: req.Name,
		}
		if req.SourceChunkID != nil {
			src.ChunkID = *req.SourceChunkID
		}
		spec.Sources = append(spec.Sources, src)
	}

	return spec
}

func resolveTemplateKey(tpl models.DocumentTemplate) string {
	// Use template_key if available (matches document engine template keys)
	if tpl.TemplateKey != "" {
		return tpl.TemplateKey
	}
	// Fallback to document_type for backward compatibility
	if tpl.DocumentType != "" {
		return tpl.DocumentType
	}
	key := strings.ToLower(tpl.Name)
	key = strings.ReplaceAll(key, " ", "_")
	return key
}

func copyFields(data map[string]string) map[string]string {
	out := make(map[string]string, len(data))
	for k, v := range data {
		out[k] = v
	}
	return out
}

func sanitizeFilename(s string) string {
	var b strings.Builder
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_' || c == '.' {
			b.WriteRune(c)
		}
	}
	result := b.String()
	if result == "" {
		return "document"
	}
	if len(result) > 200 {
		result = result[:200]
	}
	return result
}
