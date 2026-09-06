package document

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/phpdave11/gofpdf"
)

func GeneratePDF(title, body string) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 15)
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)
	pdf.MultiCell(0, 10, title, "", "L", false)
	pdf.Ln(4)
	pdf.SetFont("Arial", "", 12)
	pdf.MultiCell(0, 6, body, "", "L", false)

	var b bytes.Buffer
	if err := pdf.Output(&b); err != nil {
		return nil, fmt.Errorf("generate pdf: %w", err)
	}

	return b.Bytes(), nil
}

func ApplyPlaceholders(template string, data map[string]string) string {
	out := template
	for k, v := range data {
		placeholder := "{{" + k + "}}"
		if strings.TrimSpace(v) == "" {
			v = "[" + k + "]"
		}
		out = strings.ReplaceAll(out, placeholder, v)
	}
	return out
}
