package docspec

import (
	"testing"

	"github.com/bsmart/abis/internal/models"
)

func TestBuild_BasicFields(t *testing.T) {
	task := models.WorkflowTask{
		ID:         "task-123",
		EmployeeID: "emp-456",
		Intent:     models.IntentDocumentGeneration,
	}

	template := models.DocumentTemplate{
		ID:           "tmpl-789",
		Name:         "Solicitação de Aquisição",
		DocumentType: "solicitacao_aquisicao_ti",
		Version:      "1.0",
	}

	data := map[string]string{
		"solicitante": "João Silva",
		"fornecedor":  "Dell",
		"valor":       "R$ 50.000,00",
	}

	input := BuildInput{
		Task:     task,
		Template: template,
		Data:     data,
		RunID:    "run-999",
	}

	spec := Build(input)

	if spec.SpecID == "" {
		t.Error("spec_id should be generated")
	}
	if spec.RunID != "run-999" {
		t.Errorf("run_id = %q, want %q", spec.RunID, "run-999")
	}
	if spec.TaskID != "task-123" {
		t.Errorf("task_id = %q, want %q", spec.TaskID, "task-123")
	}
	if spec.DocumentType != "solicitacao_aquisicao_ti" {
		t.Errorf("document_type = %q, want %q", spec.DocumentType, "solicitacao_aquisicao_ti")
	}
	if spec.TemplateKey != "solicitacao_aquisicao_ti" {
		t.Errorf("template_key = %q, want %q", spec.TemplateKey, "solicitacao_aquisicao_ti")
	}
	if spec.TemplateVersion != "1.0" {
		t.Errorf("template_version = %q, want %q", spec.TemplateVersion, "1.0")
	}
	if spec.Title != "Solicitação de Aquisição" {
		t.Errorf("title = %q, want %q", spec.Title, "Solicitação de Aquisição")
	}
}

func TestBuild_WithEmployee(t *testing.T) {
	task := models.WorkflowTask{
		ID:         "task-123",
		EmployeeID: "emp-456",
	}

	template := models.DocumentTemplate{
		ID:           "tmpl-789",
		DocumentType: "test_doc",
		Version:      "1.0",
	}

	employee := &models.Employee{
		ID:   "emp-456",
		Name: "Maria Santos",
		RE:   "123456",
		Role: "TI",
	}

	input := BuildInput{
		Task:     task,
		Template: template,
		Employee: employee,
	}

	spec := Build(input)

	if spec.Metadata.EmployeeID != "emp-456" {
		t.Errorf("metadata.employee_id = %q, want %q", spec.Metadata.EmployeeID, "emp-456")
	}
	if spec.Metadata.EmployeeName != "Maria Santos" {
		t.Errorf("metadata.employee_name = %q, want %q", spec.Metadata.EmployeeName, "Maria Santos")
	}
	if spec.Metadata.EmployeeRE != "123456" {
		t.Errorf("metadata.employee_re = %q, want %q", spec.Metadata.EmployeeRE, "123456")
	}
	if spec.Metadata.Department != "TI" {
		t.Errorf("metadata.department = %q, want %q", spec.Metadata.Department, "TI")
	}
}

func TestBuild_WithRequirements(t *testing.T) {
	task := models.WorkflowTask{ID: "task-123"}
	template := models.DocumentTemplate{
		DocumentType: "test",
		Version:      "1.0",
	}

	chunkID := int64(42)
	requirements := []models.WorkflowRequirement{
		{
			Name:           "fornecedor",
			SourceDocument: "Política de Compras",
			SourceSnippet:  "Fornecedores devem ser homologados",
			SourceChunkID:  &chunkID,
		},
		{
			Name:           "valor",
			SourceDocument: "Norma de Aquisições",
			SourceSnippet:  "Valores acima de R$ 100.000 requerem aprovação",
		},
	}

	input := BuildInput{
		Task:         task,
		Template:     template,
		Requirements: requirements,
	}

	spec := Build(input)

	if len(spec.Sources) != 2 {
		t.Fatalf("len(sources) = %d, want 2", len(spec.Sources))
	}

	if spec.Sources[0].Document != "Política de Compras" {
		t.Errorf("sources[0].document = %q, want %q", spec.Sources[0].Document, "Política de Compras")
	}
	if spec.Sources[0].Requirement != "fornecedor" {
		t.Errorf("sources[0].requirement = %q, want %q", spec.Sources[0].Requirement, "fornecedor")
	}
	if spec.Sources[0].ChunkID != int64(42) {
		t.Errorf("sources[0].chunk_id = %v, want 42", spec.Sources[0].ChunkID)
	}

	if spec.Sources[1].Document != "Norma de Aquisições" {
		t.Errorf("sources[1].document = %q, want %q", spec.Sources[1].Document, "Norma de Aquisições")
	}
}

func TestBuild_OutputFormats(t *testing.T) {
	task := models.WorkflowTask{ID: "task-123"}
	template := models.DocumentTemplate{
		DocumentType: "test",
		Version:      "1.0",
	}

	input := BuildInput{
		Task:     task,
		Template: template,
	}

	spec := Build(input)

	if len(spec.Output.Formats) != 2 {
		t.Fatalf("len(output.formats) = %d, want 2", len(spec.Output.Formats))
	}

	hasDocx := false
	hasPDF := false
	for _, f := range spec.Output.Formats {
		if f == "docx" {
			hasDocx = true
		}
		if f == "pdf" {
			hasPDF = true
		}
	}

	if !hasDocx {
		t.Error("output.formats should include docx")
	}
	if !hasPDF {
		t.Error("output.formats should include pdf")
	}
}

func TestBuild_AutoGenerateRunID(t *testing.T) {
	task := models.WorkflowTask{ID: "task-123"}
	template := models.DocumentTemplate{
		DocumentType: "test",
		Version:      "1.0",
	}

	input := BuildInput{
		Task:     task,
		Template: template,
		RunID:    "",
	}

	spec := Build(input)

	if spec.RunID == "" {
		t.Error("run_id should be auto-generated when empty")
	}
	if len(spec.RunID) < 10 {
		t.Errorf("run_id = %q, seems too short for auto-generated", spec.RunID)
	}
}

func TestValidate_Valid(t *testing.T) {
	spec := &DocumentSpec{
		SpecID:          "550e8400-e29b-41d4-a716-446655440000",
		RunID:           "550e8400-e29b-41d4-a716-446655440001",
		TaskID:          "550e8400-e29b-41d4-a716-446655440002",
		DocumentType:    "solicitacao_aquisicao_ti",
		TemplateKey:     "solicitacao_aquisicao_ti",
		TemplateVersion: "1.0",
		Output: DocumentOutput{
			Formats: []string{"docx"},
		},
	}

	err := Validate(spec)
	if err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

func TestValidate_InvalidSpecID(t *testing.T) {
	spec := &DocumentSpec{
		SpecID:          "not-a-uuid",
		RunID:           "550e8400-e29b-41d4-a716-446655440001",
		TaskID:          "550e8400-e29b-41d4-a716-446655440002",
		DocumentType:    "test",
		TemplateKey:     "test",
		TemplateVersion: "1.0",
		Output: DocumentOutput{
			Formats: []string{"docx"},
		},
	}

	err := Validate(spec)
	if err == nil {
		t.Error("Validate() should fail for invalid spec_id")
	}
}

func TestValidate_InvalidTemplateKey(t *testing.T) {
	spec := &DocumentSpec{
		SpecID:          "550e8400-e29b-41d4-a716-446655440000",
		RunID:           "550e8400-e29b-41d4-a716-446655440001",
		TaskID:          "550e8400-e29b-41d4-a716-446655440002",
		DocumentType:    "test",
		TemplateKey:     "../invalid",
		TemplateVersion: "1.0",
		Output: DocumentOutput{
			Formats: []string{"docx"},
		},
	}

	err := Validate(spec)
	if err == nil {
		t.Error("Validate() should fail for invalid template_key")
	}
}

func TestValidate_InvalidFormat(t *testing.T) {
	spec := &DocumentSpec{
		SpecID:          "550e8400-e29b-41d4-a716-446655440000",
		RunID:           "550e8400-e29b-41d4-a716-446655440001",
		TaskID:          "550e8400-e29b-41d4-a716-446655440002",
		DocumentType:    "test",
		TemplateKey:     "test",
		TemplateVersion: "1.0",
		Output: DocumentOutput{
			Formats: []string{"invalid_format"},
		},
	}

	err := Validate(spec)
	if err == nil {
		t.Error("Validate() should fail for invalid format")
	}
}

func TestValidate_NoFormats(t *testing.T) {
	spec := &DocumentSpec{
		SpecID:          "550e8400-e29b-41d4-a716-446655440000",
		RunID:           "550e8400-e29b-41d4-a716-446655440001",
		TaskID:          "550e8400-e29b-41d4-a716-446655440002",
		DocumentType:    "test",
		TemplateKey:     "test",
		TemplateVersion: "1.0",
		Output: DocumentOutput{
			Formats: []string{},
		},
	}

	err := Validate(spec)
	if err == nil {
		t.Error("Validate() should fail for empty formats")
	}
}
