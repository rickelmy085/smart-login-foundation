package knowledge

import (
	"bytes"
	"fmt"
	"os"

	"github.com/ledongthuc/pdf" // pure-Go PDF parser
)

// ExtractResult é o que o extractor devolve: o texto completo e a
// quantidade de páginas identificadas pelo parser.
type ExtractResult struct {
	Text      string
	PageCount int
}

// ExtractPDF lê `path` (um PDF) e devolve o texto concatenado de todas
// as páginas. Usa o parser ledongthuc/pdf — sem CGO, sem dependências nativas.
//
// Limitações conhecidas (aceitáveis para MVP/RAG):
//   - PDFs escaneados (imagens) sem camada de texto retornam string vazia.
//     Para isso seria preciso OCR (ex: tesseract), fora de escopo agora.
//   - Encoding de acentuação pode vir bagunçado em PDFs muito antigos;
//     o Normalizer a jusante ajuda a amenizar.
func ExtractPDF(path string) (ExtractResult, error) {
	fmt.Printf("[KNOWLEDGE] ExtractPDF iniciado path=%s\n", path)
	f, r, err := pdf.Open(path)
	if err != nil {
		fmt.Printf("[KNOWLEDGE] ExtractPDF erro open: %v\n", err)
		return ExtractResult{}, fmt.Errorf("open pdf %s: %w", path, err)
	}
	defer f.Close()

	var buf bytes.Buffer
	pageCount := r.NumPage()
	fmt.Printf("[KNOWLEDGE] ExtractPDF PDF aberto pages=%d\n", pageCount)

	for pageIdx := 1; pageIdx <= pageCount; pageIdx++ {
		page := r.Page(pageIdx)
		if page.V.IsNull() {
			fmt.Printf("[KNOWLEDGE] ExtractPDF pagina %d vazia\n", pageIdx)
			continue
		}
		// page.GetTextByFont não existe nesta versão; usamos o Text().
		text, err := page.GetPlainText(nil)
		if err != nil {
			// Não abortamos o documento inteiro por uma página ruim.
			// Apenas seguimos e marcamos nos logs.
			fmt.Printf("[KNOWLEDGE] ExtractPDF erro pagina %d: %v\n", pageIdx, err)
			continue
		}
		buf.WriteString(text)
		buf.WriteString("\n\n") // separador entre páginas
	}
	fmt.Printf("[KNOWLEDGE] ExtractPDF texto extraido length=%d\n", buf.Len())

	return ExtractResult{Text: buf.String(), PageCount: pageCount}, nil
}

// IsPDF rápido: checa os primeiros bytes ("%PDF-"). Útil antes de tentar
// parsear para evitar mensagens de erro confusas em arquivos com extensão
// errada.
func IsPDF(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	hdr := make([]byte, 5)
	n, err := f.Read(hdr)
	return err == nil && n == 5 && string(hdr) == "%PDF-"
}
