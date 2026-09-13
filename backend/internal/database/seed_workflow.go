package database

import (
	"context"
	"database/sql"
	"fmt" // debug prints
	"log/slog"
)

// SeedWorkflowTemplates inserts initial document templates based on
// normative documents found in the system. This is idempotent.
func SeedWorkflowTemplates(ctx context.Context, db *sql.DB) error {
	fmt.Println("[DB] SeedWorkflowTemplates iniciada")
	templates := []struct {
		id, name, description, docType, templateKey, version, templatePath string
	}{
		{
			id:           "tmpl-aquisicao-ti",
			name:         "Solicitação de Aquisição de TI",
			description:  "Documento para solicitação de aquisição de bens ou serviços de tecnologia da informação",
			docType:      "solicitacao_aquisicao",
			templateKey:  "solicitacao_aquisicao_ti",
			version:      "1.0",
			templatePath: "storage/templates/solicitacao_aquisicao_ti.txt",
		},
		{
			id:           "tmpl-memorando-interno",
			name:         "Memorando Interno",
			description:  "Documento para comunicação interna entre departamentos",
			docType:      "memorando",
			templateKey:  "memorando_interno",
			version:      "1.0",
			templatePath: "storage/templates/memorando_interno.txt",
		},
		{
			id:           "tmpl-relatorio-operacional",
			name:         "Relatório Operacional",
			description:  "Documento para registro de atividades e resultados operacionais",
			docType:      "relatorio",
			templateKey:  "relatorio_operacional",
			version:      "1.0",
			templatePath: "storage/templates/relatorio_operacional.txt",
		},
	}

	for _, tpl := range templates {
		fmt.Printf("[DB] SeedWorkflowTemplates processando template id=%s key=%s name=%s\n", tpl.id, tpl.templateKey, tpl.name)
		var existingID, existingKey, existingPath string
		err := db.QueryRowContext(ctx, `SELECT id, template_key, template_path FROM document_templates WHERE id = ?`, tpl.id).Scan(&existingID, &existingKey, &existingPath)
		if err == nil {
			// Template exists, check if template_key or template_path needs update
			updated := false
			if existingKey != tpl.templateKey {
				fmt.Printf("[DB] SeedWorkflowTemplates atualizando template_key de %s para %s\n", existingKey, tpl.templateKey)
				_, err = db.ExecContext(ctx, `UPDATE document_templates SET template_key = ?, updated_at = datetime('now') WHERE id = ?`, tpl.templateKey, tpl.id)
				if err != nil {
					fmt.Printf("[DB] SeedWorkflowTemplates erro update template_key: %v\n", err)
				}
				updated = true
			}
			if existingPath != tpl.templatePath {
				fmt.Printf("[DB] SeedWorkflowTemplates atualizando template_path de %s para %s\n", existingPath, tpl.templatePath)
				_, err = db.ExecContext(ctx, `UPDATE document_templates SET template_path = ?, updated_at = datetime('now') WHERE id = ?`, tpl.templatePath, tpl.id)
				if err != nil {
					fmt.Printf("[DB] SeedWorkflowTemplates erro update template_path: %v\n", err)
				}
				updated = true
			}
			if !updated {
				fmt.Printf("[DB] SeedWorkflowTemplates template %s ja existe com key e path corretos, pulando\n", tpl.id)
			}
			// Continue to ensure fields are seeded
		} else {
			_, err = db.ExecContext(ctx, `
				INSERT INTO document_templates (id, name, description, document_type, template_key, version, active, template_path)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`, tpl.id, tpl.name, tpl.description, tpl.docType, tpl.templateKey, tpl.version, 1, tpl.templatePath)
			if err != nil {
				fmt.Printf("[DB] SeedWorkflowTemplates erro insert template: %v\n", err)
				slog.Warn("failed to seed template", "id", tpl.id, "error", err.Error())
				continue
			}
			fmt.Printf("[DB] SeedWorkflowTemplates template inserido id=%s key=%s\n", tpl.id, tpl.templateKey)
		}

		// Insert template fields with normative traceability
		var fields []struct {
			id, fieldName, label, fieldType string
			required                        int
			normativeDocument               string
		}

		switch tpl.templateKey {
		case "solicitacao_aquisicao_ti":
			fields = []struct {
				id, fieldName, label, fieldType string
				required                        int
				normativeDocument               string
			}{
				{tpl.id + "-tf-1", "numero_solicitacao", "Número da Solicitação", "text", 0, "Política de Compras da Organização Bradesco"},
				{tpl.id + "-tf-2", "data_solicitacao", "Data da Solicitação", "date", 1, "Política de Compras da Organização Bradesco"},
				{tpl.id + "-tf-3", "solicitante", "Solicitante", "employee", 1, "Política de Compras da Organização Bradesco"},
				{tpl.id + "-tf-4", "area_solicitante", "Área Solicitante", "department", 1, "Política de Compras da Organização Bradesco"},
				{tpl.id + "-tf-5", "fornecedor", "Fornecedor", "supplier", 1, "Política de Compras da Organização Bradesco"},
				{tpl.id + "-tf-6", "valor", "Valor da Aquisição", "currency", 1, "Política de Compras da Organização Bradesco"},
				{tpl.id + "-tf-7", "justificativa", "Justificativa", "textarea", 1, "Política de Compras da Organização Bradesco"},
				{tpl.id + "-tf-8", "categoria_produto", "Categoria do Produto", "text", 1, "Política de Compras da Organização Bradesco"},
				{tpl.id + "-tf-9", "aprovacao_cade", "Aprovação CADE", "select", 0, "Norma do Programa de Compliance Concorrencial"},
				{tpl.id + "-tf-10", "observacoes", "Observações", "textarea", 0, ""},
			}
		case "memorando_interno":
			fields = []struct {
				id, fieldName, label, fieldType string
				required                        int
				normativeDocument               string
			}{
				{tpl.id + "-tf-1", "numero_documento", "Número do Documento", "text", 0, ""},
				{tpl.id + "-tf-2", "data", "Data", "date", 1, ""},
				{tpl.id + "-tf-3", "destinatario", "Destinatário", "text", 1, ""},
				{tpl.id + "-tf-4", "remetente", "Remetente", "employee", 1, ""},
				{tpl.id + "-tf-5", "departamento", "Departamento", "department", 1, ""},
				{tpl.id + "-tf-6", "assunto", "Assunto", "text", 1, ""},
				{tpl.id + "-tf-7", "objetivo", "Objetivo", "textarea", 1, ""},
				{tpl.id + "-tf-8", "conteudo", "Conteúdo", "textarea", 1, ""},
				{tpl.id + "-tf-9", "observacoes", "Observações", "textarea", 0, ""},
			}
		case "relatorio_operacional":
			fields = []struct {
				id, fieldName, label, fieldType string
				required                        int
				normativeDocument               string
			}{
				{tpl.id + "-tf-1", "numero_relatorio", "Número do Relatório", "text", 0, ""},
				{tpl.id + "-tf-2", "data", "Data", "date", 1, ""},
				{tpl.id + "-tf-3", "responsavel", "Responsável", "employee", 1, ""},
				{tpl.id + "-tf-4", "area", "Área", "department", 1, ""},
				{tpl.id + "-tf-5", "periodo", "Período", "text", 1, ""},
				{tpl.id + "-tf-6", "objetivo", "Objetivo", "textarea", 1, ""},
				{tpl.id + "-tf-7", "resumo_executivo", "Resumo Executivo", "textarea", 1, ""},
				{tpl.id + "-tf-8", "atividades", "Atividades Realizadas", "textarea", 0, ""},
				{tpl.id + "-tf-9", "resultados", "Resultados", "textarea", 0, ""},
				{tpl.id + "-tf-10", "ocorrencias", "Ocorrências", "textarea", 0, ""},
				{tpl.id + "-tf-11", "recomendacoes", "Recomendações", "textarea", 0, ""},
				{tpl.id + "-tf-12", "observacoes", "Observações", "textarea", 0, ""},
			}
		default:
			fmt.Printf("[DB] SeedWorkflowTemplates sem campos definidos para template_key=%s\n", tpl.templateKey)
			continue
		}

		for _, f := range fields {
			fmt.Printf("[DB] SeedWorkflowTemplates inserindo field id=%s name=%s\n", f.id, f.fieldName)
			_, err = db.ExecContext(ctx, `
				INSERT INTO template_fields (id, template_id, field_name, label, type, required, source_requirement, normative_document)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`, f.id, tpl.id, f.fieldName, f.label, f.fieldType, f.required, f.fieldName, f.normativeDocument)
			if err != nil {
				fmt.Printf("[DB] SeedWorkflowTemplates erro insert field: %v\n", err)
				slog.Warn("failed to seed field", "field", f.fieldName, "error", err.Error())
			}
		}

		slog.Info("template seeded", "id", tpl.id, "name", tpl.name)
		fmt.Printf("[DB] SeedWorkflowTemplates template completo id=%s\n", tpl.id)
	}

	fmt.Println("[DB] SeedWorkflowTemplates finalizada")
	return nil
}
