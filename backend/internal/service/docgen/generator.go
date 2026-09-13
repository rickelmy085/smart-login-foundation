package docgen

import (
	"context"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/bsmart/abis/internal/docspec"
	"github.com/bsmart/abis/internal/document"
	"github.com/bsmart/abis/internal/documentengine"
	"github.com/bsmart/abis/internal/models"
	"github.com/bsmart/abis/internal/repository"
	"github.com/google/uuid"
)

var (
	ErrTaskNotFound     = errors.New("task not found")
	ErrTemplateNotFound = errors.New("template not found")
	ErrMissingData      = errors.New("missing required data")
	ErrValidationFailed = errors.New("validation failed")
)

// Generator handles document generation using either the Python Document Engine or Go generator.
type Generator struct {
	repo             *repository.WorkflowRepo
	docEngine        *documentengine.Client
	docEngineEnabled bool
	docEngineFallback bool
	employeeRepo     *repository.EmployeeRepo
}

func NewGenerator(repo *repository.WorkflowRepo) *Generator {
	return &Generator{repo: repo}
}

func (g *Generator) WithDocumentEngine(client *documentengine.Client, enabled, fallback bool) *Generator {
	g.docEngine = client
	g.docEngineEnabled = enabled
	g.docEngineFallback = enabled && fallback
	return g
}

func (g *Generator) WithEmployeeRepo(repo *repository.EmployeeRepo) *Generator {
	g.employeeRepo = repo
	return g
}

func (g *Generator) Generate(ctx context.Context, taskID string) (*models.DocumentRun, error) {
	task, err := g.repo.GetTask(ctx, taskID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}

	if task.Status != models.StatusReadyToGenerate {
		return nil, fmt.Errorf("task not ready for generation: current status %s", task.Status)
	}

	if task.TemplateID == "" {
		return nil, errors.New("templateID vazio")
	}

	taskData, err := g.repo.GetData(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("get data: %w", err)
	}

	fields, err := g.repo.GetTemplateFields(ctx, task.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("get template fields: %w", err)
	}

	for _, f := range fields {
		if f.Required {
			if val, exists := taskData[f.FieldName]; !exists || strings.TrimSpace(val) == "" {
				return nil, fmt.Errorf("missing required field: %s", f.FieldName)
			}
		}
	}

	requirements, _ := g.repo.GetRequirements(ctx, taskID)

	if g.docEngineEnabled && g.docEngine != nil {
		run, err := g.generateViaPython(ctx, task, taskData, requirements)
		if err != nil {
			slog.Error("Document Engine Python failed", "task_id", taskID, "error", err.Error())
			if g.docEngineFallback {
				slog.Warn("Falling back to Go generator", "task_id", taskID, "reason", err.Error())
			} else {
				return nil, fmt.Errorf("document engine failed: %w", err)
			}
		} else {
			if err := g.repo.UpdateTaskStatus(ctx, taskID, models.StatusGenerated, task.TemplateID); err != nil {
				return nil, fmt.Errorf("update task status: %w", err)
			}
			return run, nil
		}
	}

	run, err := g.generateViaGo(ctx, taskID, task.TemplateID, task.EmployeeID, taskData, requirements)
	if err != nil {
		return nil, err
	}

	if err := g.repo.UpdateTaskStatus(ctx, taskID, models.StatusGenerated, task.TemplateID); err != nil {
		return nil, fmt.Errorf("update task status: %w", err)
	}

	return run, nil
}

func (g *Generator) generateViaPython(ctx context.Context, task models.WorkflowTask, data map[string]string, requirements []models.WorkflowRequirement) (*models.DocumentRun, error) {
	template, err := g.repo.GetTemplate(ctx, task.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("get template: %w", err)
	}

	var employee *models.Employee
	if g.employeeRepo != nil {
		emp, err := g.employeeRepo.FindByID(ctx, task.EmployeeID)
		if err == nil {
			employee = &emp
		}
	}

	runID := uuid.NewString()
	spec := docspec.Build(docspec.BuildInput{
		Task:         task,
		Template:     template,
		Data:         data,
		Requirements: requirements,
		Employee:     employee,
		RunID:        runID,
	})

	response, err := g.docEngine.Generate(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("python engine: %w", err)
	}

	outputDir := filepath.Join("data", "documents", runID)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}

	var docxPath, pdfPath string
	for _, file := range response.Files {
		if file.DataBase64 == "" {
			continue
		}
		decoded, err := base64.StdEncoding.DecodeString(file.DataBase64)
		if err != nil {
			return nil, fmt.Errorf("decode %s: %w", file.Format, err)
		}

		outputPath := filepath.Join(outputDir, file.Filename)
		if err := os.WriteFile(outputPath, decoded, 0o644); err != nil {
			return nil, fmt.Errorf("write %s: %w", file.Format, err)
		}

		if file.Format == "docx" {
			docxPath = outputPath
		} else if file.Format == "pdf" {
			pdfPath = outputPath
		}
	}

	if docxPath == "" {
		return nil, fmt.Errorf("no DOCX file in response")
	}

	run := &models.DocumentRun{
		ID:              runID,
		TaskID:          task.ID,
		EmployeeID:      task.EmployeeID,
		TemplateID:      template.ID,
		TemplateVersion: template.Version,
		Status:          models.RunStatusPending,
		DocxPath:        docxPath,
		PdfPath:         pdfPath,
	}

	if err := g.repo.CreateDocumentRun(ctx, *run); err != nil {
		return nil, fmt.Errorf("create document run: %w", err)
	}

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
		g.repo.CreateDocumentSource(ctx, ds)
	}

	slog.Info("Document generated via Python engine",
		"run_id", runID,
		"task_id", task.ID,
		"template", template.Name,
		"files", len(response.Files),
	)

	return run, nil
}

func (g *Generator) generateViaGo(ctx context.Context, taskID, templateID, employeeID string, data map[string]string, requirements []models.WorkflowRequirement) (*models.DocumentRun, error) {
	template, err := g.repo.GetTemplate(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("template not found: %w", err)
	}

	raw, err := document.ReadTextTemplate(template.TemplatePath)
	if err != nil {
		return nil, fmt.Errorf("read template: %w", err)
	}

	filled := document.ApplyPlaceholders(raw, data)

	docxBytes, err := document.GenerateDOCX(template.Name, filled)
	if err != nil {
		return nil, fmt.Errorf("generate docx: %w", err)
	}

	pdfBytes, err := document.GeneratePDF(template.Name, filled)
	if err != nil {
		return nil, fmt.Errorf("generate pdf: %w", err)
	}

	runID := uuid.NewString()
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

	run := &models.DocumentRun{
		ID:              uuid.NewString(),
		TaskID:          taskID,
		EmployeeID:      employeeID,
		TemplateID:      templateID,
		TemplateVersion: template.Version,
		Status:          models.RunStatusPending,
		DocxPath:        docxPath,
		PdfPath:         pdfPath,
	}

	if err := g.repo.CreateDocumentRun(ctx, *run); err != nil {
		return nil, fmt.Errorf("create document run: %w", err)
	}

	for _, req := range requirements {
		ds := models.DocumentSource{
			ID:                "ds-" + uuid.NewString(),
			DocumentRunID:     run.ID,
			NormativeDocument: req.SourceDocument,
			NormativeSnippet:  req.SourceSnippet,
			Requirement:       req.Name,
		}
		if req.SourceChunkID != nil {
			ds.NormativeChunkID = req.SourceChunkID
		}
		g.repo.CreateDocumentSource(ctx, ds)
	}

	return run, nil
}

type TemplateResolver struct {
	repo *repository.WorkflowRepo
}

func NewTemplateResolver(repo *repository.WorkflowRepo) *TemplateResolver {
	return &TemplateResolver{repo: repo}
}

func (r *TemplateResolver) Resolve(ctx context.Context, task models.WorkflowTask) (models.DocumentTemplate, error) {
	if task.TemplateKey != "" {
		validKeys, err := r.repo.ListTemplateKeys(ctx)
		if err != nil {
			return models.DocumentTemplate{}, fmt.Errorf("list template keys: %w", err)
		}

		isValid := false
		for _, k := range validKeys {
			if k == task.TemplateKey {
				isValid = true
				break
			}
		}

		if isValid {
			template, err := r.repo.GetTemplateByKey(ctx, task.TemplateKey)
			if err == nil {
				return template, nil
			}
		}
	}

	if task.Intent != models.IntentDocumentGeneration {
		return models.DocumentTemplate{}, sql.ErrNoRows
	}

	proc := task.Procedure
	if proc == "" {
		proc = task.OriginalRequest
	}
	normalizedKey := normalizeTemplateKey(proc)

	if normalizedKey != "" {
		template, err := r.repo.GetTemplateByKey(ctx, normalizedKey)
		if err == nil {
			return template, nil
		}
	}

	return models.DocumentTemplate{}, sql.ErrNoRows
}

func normalizeTemplateKey(text string) string {
	lower := strings.ToLower(text)

	mappings := map[string]string{
		"aquisição":      "solicitacao_aquisicao_ti",
		"aquisicao":      "solicitacao_aquisicao_ti",
		"compra":         "solicitacao_aquisicao_ti",
		"compras":        "solicitacao_aquisicao_ti",
		"memorando":      "memorando_interno",
		"memo":           "memorando_interno",
		"relatório":      "relatorio_operacional",
		"relatorio":      "relatorio_operacional",
		"report":         "relatorio_operacional",
	}

	for keyword, key := range mappings {
		if strings.Contains(lower, keyword) {
			return key
		}
	}

	return ""
}