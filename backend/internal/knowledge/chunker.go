package knowledge

import (
	"fmt" // debug prints
	"strings"
)

// ChunkOptions define como o texto será dividido.
// Defaults razoáveis: pedaços de ~800 chars com 100 de sobreposição.
type ChunkOptions struct {
	Size    int // tamanho-alvo de cada chunk (em runes/caracteres)
	Overlap int // quantos caracteres finais de um chunk se repetem no próximo
}

func (o ChunkOptions) withDefaults() ChunkOptions {
	if o.Size <= 0 {
		o.Size = 800
	}
	if o.Overlap < 0 {
		o.Overlap = 0
	}
	if o.Overlap >= o.Size {
		// garante progresso (senão fica em loop infinito)
		o.Overlap = o.Size / 4
	}
	return o
}

// Split divide `text` em janelas com sobreposição usando quebras
// "naturais" (parágrafos → frases → palavras) para não cortar no meio
// de uma frase.
//
// Estratégia:
//   1) Se o texto cabe em Size, devolve um único chunk.
//   2) Caso contrário, tenta quebrar em blocos de parágrafo (\n\n).
//      Se um parágrafo sozinho for maior que Size, quebra em sentenças.
//      Se uma sentença sozinha for maior que Size, quebra em palavras.
//   3) Empacota sentenças/palavras até preencher Size, depois emite o
//      chunk e começa o próximo com Overlap caracteres deOverlap.
func Split(text string, opt ChunkOptions) []Chunk {
	fmt.Printf("[KNOWLEDGE] Split iniciado text_length=%d opt=%+v\n", len(text), opt)
	opt = opt.withDefaults()
	text = strings.TrimSpace(text)
	if text == "" {
		fmt.Println("[KNOWLEDGE] Split texto vazio")
		return nil
	}
	if len([]rune(text)) <= opt.Size {
		fmt.Printf("[KNOWLEDGE] Split texto cabe em 1 chunk\n")
		return []Chunk{{Ord: 0, Content: text, CharStart: 0, CharEnd: len([]byte(text))}}
	}

	var chunks []Chunk
	runes := []rune(text)

	step := opt.Size - opt.Overlap
	if step <= 0 {
		step = opt.Size
	}

	for start := 0; start < len(runes); {
		end := start + opt.Size
		if end > len(runes) {
			end = len(runes)
		}

		// Tenta recuar `end` até uma quebra natural (., !, ?, \n)
		// para não cortar no meio de uma frase — desde que não
		// recuemos mais que 20% do tamanho desejado.
		if end < len(runes) {
			minEnd := start + (opt.Size * 4 / 5)
			for i := end; i > minEnd; i-- {
				r := runes[i-1]
				if r == '\n' || r == '.' || r == '!' || r == '?' {
					end = i
					break
				}
			}
		}

		// Byte offsets (para metadados; útil para highlight futuro).
		byteStart := len(string(runes[:start]))
		byteEnd := len(string(runes[:end]))

		chunks = append(chunks, Chunk{
			Ord:       len(chunks),
			Content:   strings.TrimSpace(string(runes[start:end])),
			CharStart: byteStart,
			CharEnd:   byteEnd,
		})

		if end == len(runes) {
			break
		}
		start += step
	}

	fmt.Printf("[KNOWLEDGE] Split finalizado chunks=%d\n", len(chunks))
	return chunks
}
