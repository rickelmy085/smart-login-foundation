package docspec

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var (
	uuidRE     = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	safeKeyRE  = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	versionRE  = regexp.MustCompile(`^[0-9]+(\.[0-9]+)*$`)
	validFormats = map[string]bool{"docx": true, "pdf": true}
)

func Validate(spec *DocumentSpec) error {
	if !uuidRE.MatchString(spec.SpecID) {
		return fmt.Errorf("invalid spec_id format: %q", spec.SpecID)
	}
	if !uuidRE.MatchString(spec.RunID) {
		return fmt.Errorf("invalid run_id format: %q", spec.RunID)
	}
	if !uuidRE.MatchString(spec.TaskID) {
		return fmt.Errorf("invalid task_id format: %q", spec.TaskID)
	}
	if spec.DocumentType == "" || !safeKeyRE.MatchString(spec.DocumentType) {
		return fmt.Errorf("invalid document_type: %q", spec.DocumentType)
	}
	if spec.TemplateKey == "" || !safeKeyRE.MatchString(spec.TemplateKey) {
		return fmt.Errorf("invalid template_key: %q", spec.TemplateKey)
	}
	if spec.TemplateVersion == "" || !versionRE.MatchString(spec.TemplateVersion) {
		return fmt.Errorf("invalid template_version: %q", spec.TemplateVersion)
	}
	if len(spec.Output.Formats) == 0 {
		return fmt.Errorf("at least one output format is required")
	}
	for _, f := range spec.Output.Formats {
		if !validFormats[strings.ToLower(f)] {
			return fmt.Errorf("unsupported output format: %q", f)
		}
	}
	if spec.Output.FilenamePrefix == "" {
		spec.Output.FilenamePrefix = "document"
	}
	if spec.Metadata.GeneratedAt == "" {
		spec.Metadata.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	}
	return nil
}
