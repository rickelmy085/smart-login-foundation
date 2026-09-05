package database

import (
	"context"
	"database/sql"
	"log/slog"
)

// SeedWorkflowTemplates inserts initial document templates based on
// normative documents found in the system. This is idempotent.
func SeedWorkflowTemplates(ctx context.Context, db *sql.DB) error {
	templates := []struct {
		id, name, description, docType, version, templatePath string
	}{
		{
			id:          "tmpl-aquisicao-ti",
			name:        "Solicitação de Aquisição de TI",
			description: "Documento para solicitação de aquisição de bens ou serviços de tecnologia da informação",
			docType:     "solicitacao_aquisicao",
			version:     "1.0",
			templatePath: "backend/storage/templates/solicitacao_aquisicao_ti.txt",
		},
	}

	for _, tpl := range templates {
		var existing string
		err := db.QueryRowContext(ctx, `SELECT id FROM document_templates WHERE id = ?`, tpl.id).Scan(&existing)
		if err == nil {
			continue // já existe
		}

		_, err = db.ExecContext(ctx, `
			INSERT INTO document_templates (id, name, description, document_type, version, active, template_path)
			VALUES (?, ?, ?, ?, ?, 1, ?)
		`, tpl.id, tpl.name, tpl.description, tpl.docType, tpl.version, tpl.templatePath)
		if err != nil {
			slog.Warn("failed to seed template", "id", tpl.id, "error", err.Error())
			continue
		}

		// Insert template fields with normative traceability
		fields := []struct {
			id, fieldName, label, fieldType string
			required                        int
			normativeDocument               string
		}{
			{"tf-1", "numero_solicitacao", "Número da Solicitação", "text", 0, "Política de Compras da Organização Bradesco"},
			{"tf-2", "data_solicitacao", "Data da Solicitação", "date", 1, "Política de Compras da Organização Bradesco"},
			{"tf-3", "solicitante", "Solicitante", "employee", 1, "Política de Compras da Organização Bradesco"},
			{"tf-4", "area_solicitante", "Área Solicitante", "department", 1, "Política de Compras da Organização Bradesco"},
			{"tf-5", "fornecedor", "Fornecedor", "supplier", 1, "Política de Compras da Organização Bradesco"},
			{"tf-6", "valor", "Valor da Aquisição", "currency", 1, "Política de Compras da Organização Bradesco"},
			{"tf-7", "justificativa", "Justificativa", "textarea", 1, "Política de Compras da Organização Bradesco"},
			{"tf-8", "categoria_produto", "Categoria do Produto", "text", 1, "Política de Compras da Organização Bradesco"},
			{"tf-9", "aprovacao_cade", "Aprovação CADE", "select", 0, "Norma do Programa de Compliance Concorrencial"},
			{"tf-10", "observacoes", "Observações", "textarea", 0, ""},
		}

		for _, f := range fields {
			_, err = db.ExecContext(ctx, `
				INSERT INTO template_fields (id, template_id, field_name, label, type, required, source_requirement, normative_document)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`, f.id, tpl.id, f.fieldName, f.label, f.fieldType, f.required, f.fieldName, f.normativeDocument)
			if err != nil {
				slog.Warn("failed to seed field", "field", f.fieldName, "error", err.Error())
			}
		}

		slog.Info("template seeded", "id", tpl.id, "name", tpl.name)
	}

	return nil
}
