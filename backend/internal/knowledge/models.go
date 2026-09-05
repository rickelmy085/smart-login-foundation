// Package knowledge implementa o pipeline de ingestão e busca de
// documentos (PDFs) usado pelo ABIS como base para o futuro RAG.
//
// Responsabilidades deste pacote (cada uma no seu arquivo):
//
//	scanner     → varre diretórios e calcula hash dos arquivos
//	extractor   → extrai texto de PDFs (ledongthuc/pdf, puro-Go)
//	normalizer  → limpa/normaliza o texto bruto
//	chunker     → divide o texto em janelas com sobreposição
//	ingest      → orquestra o pipeline e grava no SQLite
//	search      → consulta FTS5 (BM25) e devolve trechos ranqueados
//
// Este pacote não conhece HTTP nem JWT — fica fácil reusar tanto num
// servidor quanto num CLI (cmd/ingest).
package knowledge

// Document é a representação interna de um PDF indexado.
type Document struct {
	ID         string // hash SHA-256 do arquivo (chave estável e deduplicável)
	SourcePath string
	Title      string
	PageCount  int
	CharCount  int
	Hash       string
}

// Chunk é um trecho de texto pertencente a um Document.
// É o que devolvemos em uma busca.
type Chunk struct {
	ID         int64
	DocumentID string
	Ord        int    // posição do chunk dentro do documento
	Content    string // texto do trecho
	CharStart  int    // offset inicial no texto original
	CharEnd    int    // offset final
}

// SearchHit é um resultado de busca: um chunk + metadados do doc + score.
// Score menor = mais relevante (convenção do BM25 do SQLite).
type SearchHit struct {
	Chunk     Chunk
	Document  Document
	Snippet   string  // trecho curto ao redor do match para highlight
	Score     float64 // bm25() retornado pelo SQLite
}
