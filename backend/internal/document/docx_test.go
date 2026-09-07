package document

import (
	"bytes"
	"archive/zip"
	"strings"
	"testing"
)

func TestGenerateDOCX_ValidPackage(t *testing.T) {
	docx, err := GenerateDOCX("Test Title", "Test body content")
	if err != nil {
		t.Fatalf("GenerateDOCX failed: %v", err)
	}

	if len(docx) == 0 {
		t.Fatal("DOCX is empty")
	}

	// Verify it's a valid ZIP
	zr, err := zip.NewReader(bytes.NewReader(docx), int64(len(docx)))
	if err != nil {
		t.Fatalf("not a valid zip: %v", err)
	}

	// Verify required entries exist
	required := map[string]bool{
		"[Content_Types].xml":          false,
		"_rels/.rels":                  false,
		"word/document.xml":            false,
		"word/_rels/document.xml.rels": false,
		"word/styles.xml":              false,
	}

	for _, f := range zr.File {
		if _, ok := required[f.Name]; ok {
			required[f.Name] = true
		}
	}

	for name, found := range required {
		if !found {
			t.Errorf("missing required entry: %s", name)
		}
	}
}

func TestGenerateDOCX_ContainsContent(t *testing.T) {
	docx, err := GenerateDOCX("Meu Documento", "Conteúdo do documento")
	if err != nil {
		t.Fatalf("GenerateDOCX failed: %v", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(docx), int64(len(docx)))
	if err != nil {
		t.Fatalf("not a valid zip: %v", err)
	}

	// Find document.xml and verify it contains the title and body
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("failed to open document.xml: %v", err)
			}
			defer rc.Close()

			buf := new(bytes.Buffer)
			if _, err := buf.ReadFrom(rc); err != nil {
				t.Fatalf("failed to read document.xml: %v", err)
			}

			content := buf.String()
			if !strings.Contains(content, "Meu Documento") {
				t.Error("document.xml does not contain title")
			}
			if !strings.Contains(content, "Conteúdo do documento") {
				t.Error("document.xml does not contain body")
			}
			return
		}
	}

	t.Fatal("word/document.xml not found in zip")
}

func TestGeneratePDF_Valid(t *testing.T) {
	pdf, err := GeneratePDF("Test PDF", "Some content here")
	if err != nil {
		t.Fatalf("GeneratePDF failed: %v", err)
	}

	if len(pdf) == 0 {
		t.Fatal("PDF is empty")
	}

	// Verify PDF magic bytes
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		t.Error("PDF does not start with %PDF- magic bytes")
	}
}

func TestApplyPlaceholders(t *testing.T) {
	template := "Solicitante: {{solicitante}} | Valor: {{valor}} | Faltando: {{falta}}"
	data := map[string]string{
		"solicitante": "João Silva",
		"valor":       "R$ 100,00",
		// falta is missing - should remain as {{falta}}
	}

	result := ApplyPlaceholders(template, data)

	if !strings.Contains(result, "Solicitante: João Silva") {
		t.Error("solicitante not replaced")
	}
	if !strings.Contains(result, "Valor: R$ 100,00") {
		t.Error("valor not replaced")
	}
	// Missing keys remain as {{key}} — no artificial defaults
	if !strings.Contains(result, "Faltando: {{falta}}") {
		t.Error("missing key should remain as {{key}}, not be replaced with a default")
	}
}

func TestApplyPlaceholders_EmptyData(t *testing.T) {
	template := "Campo: {{campo}}"
	data := map[string]string{}

	result := ApplyPlaceholders(template, data)

	// Missing keys remain as {{key}} — no artificial defaults
	if !strings.Contains(result, "Campo: {{campo}}") {
		t.Error("empty data should leave {{key}} placeholder untouched")
	}
}

func TestApplyPlaceholders_EmptyValue(t *testing.T) {
	template := "Campo: {{campo}}"
	data := map[string]string{
		"campo": "", // key exists but value is empty
	}

	result := ApplyPlaceholders(template, data)

	// Empty values (key exists) are replaced with [field_name]
	if !strings.Contains(result, "[campo]") {
		t.Error("empty value should be replaced with [field_name]")
	}
}

func TestGenerateDOCX_EmptyBody(t *testing.T) {
	docx, err := GenerateDOCX("", "")
	if err != nil {
		t.Fatalf("GenerateDOCX with empty content failed: %v", err)
	}

	if len(docx) == 0 {
		t.Fatal("DOCX with empty content is empty")
	}

	// Should still be a valid zip
	zr, err := zip.NewReader(bytes.NewReader(docx), int64(len(docx)))
	if err != nil {
		t.Fatalf("empty content DOCX is not a valid zip: %v", err)
	}

	if len(zr.File) == 0 {
		t.Fatal("empty content DOCX has no entries")
	}
}