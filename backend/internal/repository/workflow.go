package repository

import (
	"context"
	"database/sql"
	"strconv"

	"github.com/bsmart/abis/internal/models"
)

// WorkflowRepo handles database access for workflow tasks and related entities.
type WorkflowRepo struct {
	db *sql.DB
}

func NewWorkflowRepo(db *sql.DB) *WorkflowRepo {
	return &WorkflowRepo{db: db}
}

// CreateTask inserts a new workflow task.
func (r *WorkflowRepo) CreateTask(ctx context.Context, t models.WorkflowTask) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO workflow_tasks (id, employee_id, intent, procedure, status, original_request, template_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, t.ID, t.EmployeeID, t.Intent, t.Procedure, t.Status, t.OriginalRequest, t.TemplateID)
	return err
}

// GetTask retrieves a task by ID.
func (r *WorkflowRepo) GetTask(ctx context.Context, id string) (models.WorkflowTask, error) {
	var t models.WorkflowTask
	err := r.db.QueryRowContext(ctx, `
		SELECT id, employee_id, intent, procedure, status, original_request, template_id, created_at, updated_at
		FROM workflow_tasks WHERE id = ?
	`, id).Scan(
		&t.ID, &t.EmployeeID, &t.Intent, &t.Procedure, &t.Status,
		&t.OriginalRequest, &t.TemplateID, &t.CreatedAt, &t.UpdatedAt,
	)
	return t, err
}

// UpdateTaskStatus updates the status and template_id of a task.
func (r *WorkflowRepo) UpdateTaskStatus(ctx context.Context, id string, status models.TaskStatus, templateID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE workflow_tasks SET status = ?, template_id = ?, updated_at = datetime('now')
		WHERE id = ?
	`, status, templateID, id)
	return err
}

// UpdateTaskTemplate updates only the template_id.
func (r *WorkflowRepo) UpdateTaskTemplate(ctx context.Context, id, templateID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE workflow_tasks SET template_id = ?, updated_at = datetime('now')
		WHERE id = ?
	`, templateID, id)
	return err
}

// CreateRequirement inserts a requirement for a task.
func (r *WorkflowRepo) CreateRequirement(ctx context.Context, req models.WorkflowRequirement) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO workflow_requirements (id, task_id, name, label, required, source_document, source_chunk_id, source_snippet, rank)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, req.ID, req.TaskID, req.Name, req.Label, req.Required, req.SourceDocument,
		req.SourceChunkID, req.SourceSnippet, req.Rank)
	return err
}

// GetRequirements returns all requirements for a task, ordered by rank.
func (r *WorkflowRepo) GetRequirements(ctx context.Context, taskID string) ([]models.WorkflowRequirement, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, task_id, name, label, required, source_document, source_chunk_id, source_snippet, rank, created_at
		FROM workflow_requirements WHERE task_id = ? ORDER BY rank
	`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.WorkflowRequirement
	for rows.Next() {
		var req models.WorkflowRequirement
		var chunkID sql.NullInt64
		err := rows.Scan(
			&req.ID, &req.TaskID, &req.Name, &req.Label, &req.Required,
			&req.SourceDocument, &chunkID, &req.SourceSnippet, &req.Rank, &req.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if chunkID.Valid {
			req.SourceChunkID = &chunkID.Int64
		}
		result = append(result, req)
	}
	return result, rows.Err()
}

// SetData records a data field value for a task.
func (r *WorkflowRepo) SetData(ctx context.Context, taskID, field, value string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO workflow_data (id, task_id, field_name, value)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(task_id, field_name) DO UPDATE SET value = excluded.value, updated_at = datetime('now')
	`, "wd-"+taskID+"-"+field, taskID, field, value)
	return err
}

// GetData returns all data for a task.
func (r *WorkflowRepo) GetData(ctx context.Context, taskID string) (map[string]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT field_name, value FROM workflow_data WHERE task_id = ?
	`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]string)
	for rows.Next() {
		var field, value string
		if err := rows.Scan(&field, &value); err != nil {
			return nil, err
		}
		result[field] = value
	}
	return result, rows.Err()
}

// --- Templates ---

// GetTemplate retrieves a template by ID.
func (r *WorkflowRepo) GetTemplate(ctx context.Context, id string) (models.DocumentTemplate, error) {
	var t models.DocumentTemplate
	var active int
	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, description, document_type, version, active, template_path, created_at, updated_at
		FROM document_templates WHERE id = ?
	`, id).Scan(
		&t.ID, &t.Name, &t.Description, &t.DocumentType, &t.Version,
		&active, &t.TemplatePath, &t.CreatedAt, &t.UpdatedAt,
	)
	t.Active = active == 1
	return t, err
}

// GetTemplateByKey retrieves a template by a key (matched against name).
func (r *WorkflowRepo) GetTemplateByKey(ctx context.Context, key string) (models.DocumentTemplate, error) {
	var t models.DocumentTemplate
	var active int
	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, description, document_type, version, active, template_path, created_at, updated_at
		FROM document_templates WHERE name LIKE ? AND active = 1 LIMIT 1
	`, "%"+key+"%").Scan(
		&t.ID, &t.Name, &t.Description, &t.DocumentType, &t.Version,
		&active, &t.TemplatePath, &t.CreatedAt, &t.UpdatedAt,
	)
	t.Active = active == 1
	return t, err
}

// GetTemplateFields returns all fields for a template.
func (r *WorkflowRepo) GetTemplateFields(ctx context.Context, templateID string) ([]models.TemplateField, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, template_id, field_name, label, type, required, validation_rule, source_requirement, normative_document, normative_chunk_id, created_at
		FROM template_fields WHERE template_id = ? ORDER BY created_at
	`, templateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.TemplateField
	for rows.Next() {
		var f models.TemplateField
		var chunkID sql.NullInt64
		var required, activeInt int
		if err := rows.Scan(
			&f.ID, &f.TemplateID, &f.FieldName, &f.Label, &f.Type,
			&required, &f.ValidationRule, &f.SourceRequirement, &f.NormativeDocument,
			&chunkID, &f.CreatedAt,
		); err != nil {
			return nil, err
		}
		f.Required = required == 1
		if chunkID.Valid {
			f.NormativeChunkID = &chunkID.Int64
		}
		_ = activeInt
		result = append(result, f)
	}
	return result, rows.Err()
}

// --- Document Runs ---

// CreateDocumentRun creates a new document run record.
func (r *WorkflowRepo) CreateDocumentRun(ctx context.Context, run models.DocumentRun) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO document_runs (id, task_id, employee_id, template_id, template_version, status, docx_path, pdf_path)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, run.ID, run.TaskID, run.EmployeeID, run.TemplateID, run.TemplateVersion,
		run.Status, run.DocxPath, run.PdfPath)
	return err
}

// UpdateDocumentRunStatus updates the status and file paths of a document run.
func (r *WorkflowRepo) UpdateDocumentRunStatus(ctx context.Context, id string, status models.DocumentRunStatus, docxPath, pdfPath string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE document_runs SET status = ?, docx_path = ?, pdf_path = ?, updated_at = datetime('now')
		WHERE id = ?
	`, status, docxPath, pdfPath, id)
	return err
}

// GetDocumentRun retrieves a document run by ID.
func (r *WorkflowRepo) GetDocumentRun(ctx context.Context, id string) (models.DocumentRun, error) {
	var r_ models.DocumentRun
	var docxPath, pdfPath sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT id, task_id, employee_id, template_id, template_version, status, docx_path, pdf_path, created_at, updated_at
		FROM document_runs WHERE id = ?
	`, id).Scan(
		&r_.ID, &r_.TaskID, &r_.EmployeeID, &r_.TemplateID, &r_.TemplateVersion,
		&r_.Status, &docxPath, &pdfPath, &r_.CreatedAt, &r_.UpdatedAt,
	)
	if docxPath.Valid {
		r_.DocxPath = docxPath.String
	}
	if pdfPath.Valid {
		r_.PdfPath = pdfPath.String
	}
	return r_, err
}

// CreateDocumentSource records a source used in a document run.
func (r *WorkflowRepo) CreateDocumentSource(ctx context.Context, ds models.DocumentSource) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO document_sources (id, document_run_id, normative_document, normative_chunk_id, normative_snippet, requirement)
		VALUES (?, ?, ?, ?, ?, ?)
	`, ds.ID, ds.DocumentRunID, ds.NormativeDocument, ds.NormativeChunkID,
		ds.NormativeSnippet, ds.Requirement)
	return err
}

// GetDocumentSources returns all sources for a document run.
func (r *WorkflowRepo) GetDocumentSources(ctx context.Context, runID string) ([]models.DocumentSource, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, document_run_id, normative_document, normative_chunk_id, normative_snippet, requirement, created_at
		FROM document_sources WHERE document_run_id = ?
	`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.DocumentSource
	for rows.Next() {
		var ds models.DocumentSource
		var chunkID sql.NullInt64
		if err := rows.Scan(
			&ds.ID, &ds.DocumentRunID, &ds.NormativeDocument, &chunkID,
			&ds.NormativeSnippet, &ds.Requirement, &ds.CreatedAt,
		); err != nil {
			return nil, err
		}
		if chunkID.Valid {
			ds.NormativeChunkID = &chunkID.Int64
		}
		result = append(result, ds)
	}
	return result, rows.Err()
}

// --- Tasks query helpers ---

// GetTasksByEmployee returns recent tasks for an employee.
func (r *WorkflowRepo) GetTasksByEmployee(ctx context.Context, employeeID string) ([]models.WorkflowTask, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, employee_id, intent, procedure, status, original_request, template_id, created_at, updated_at
		FROM workflow_tasks WHERE employee_id = ? ORDER BY created_at DESC LIMIT 50
	`, employeeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.WorkflowTask
	for rows.Next() {
		var t models.WorkflowTask
		err := rows.Scan(
			&t.ID, &t.EmployeeID, &t.Intent, &t.Procedure, &t.Status,
			&t.OriginalRequest, &t.TemplateID, &t.CreatedAt, &t.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, t)
	}
	return result, rows.Err()
}

// GetDocumentRunByEmployee returns all document runs for an employee (via tasks).
func (r *WorkflowRepo) GetDocumentRunByEmployee(ctx context.Context, employeeID string) ([]models.DocumentRun, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT dr.id, dr.task_id, dr.employee_id, dr.template_id, dr.template_version,
		       dr.status, dr.docx_path, dr.pdf_path, dr.created_at, dr.updated_at
		FROM document_runs dr
		JOIN workflow_tasks wt ON wt.id = dr.task_id
		WHERE wt.employee_id = ?
		ORDER BY dr.created_at DESC
	`, employeeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []models.DocumentRun
	for rows.Next() {
		var dr models.DocumentRun
		var docxPath, pdfPath sql.NullString
		if err := rows.Scan(
			&dr.ID, &dr.TaskID, &dr.EmployeeID, &dr.TemplateID, &dr.TemplateVersion,
			&dr.Status, &docxPath, &pdfPath, &dr.CreatedAt, &dr.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if docxPath.Valid {
			dr.DocxPath = docxPath.String
		}
		if pdfPath.Valid {
			dr.PdfPath = pdfPath.String
		}
		result = append(result, dr)
	}
	return result, rows.Err()
}

// IntToString is a helper for converting.
func IntToString(i int) string {
	return strconv.Itoa(i)
}
