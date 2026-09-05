package knowledge

import (
	"strings"
	"unicode"
)

// Normalize recebe o texto bruto extraído do PDF e devolve uma versão
// limpa, pronta para indexação e chunking.
//
// Operações (em ordem):
//  1. Substitui quebras de linha Windows e tabs por espaços.
//  2. Remove espaços duplicados.
//  3. Reúne hifenização de fim de linha ("desenvol-\nvimento" → "desenvolvimento").
//     Comum em PDFs quando uma palavra é quebrada entre linhas.
//  4. Reúne múltiplas quebras de linha em uma só (preserva parágrafos).
//  5. Trim final.
//
// Mantemos acentos! O FTS5 está configurado com `unicode61 remove_diacritics 2`,
// o que já trata a busca tolerante a acentos. Normalizar aqui evitaria
// devolver texto sem acentos ao usuário (o que fica esquisito na UI).
func Normalize(in string) string {
	s := in

	// CRLF, CR, tab → espaço.
	s = strings.NewReplacer("\r\n", " ", "\r", " ", "\t", " ").Replace(s)

	// Hifenização de fim de linha. Padrão "-\n" onde \n é newline real.
	// O ledongthuc às vezes retorna \n puro, às vezes \r\n (já tratado).
	// Aqui tratamos tanto "-\n" quanto "-\r\n".
	for {
		new := strings.ReplaceAll(s, "-\n", "")
		if new == s {
			break
		}
		s = new
	}

	// Múltiplas quebras de linha → uma (preserva separação de parágrafos).
	for {
		new := strings.ReplaceAll(s, "\n\n\n", "\n\n")
		if new == s {
			break
		}
		s = new
	}

	// Colapsa múltiplos espaços em um só.
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := false
	for _, r := range s {
		if r == '\n' {
			b.WriteRune(r)
			prevSpace = false
			continue
		}
		if unicode.IsSpace(r) {
			if !prevSpace {
				b.WriteRune(' ')
			}
			prevSpace = true
			continue
		}
		b.WriteRune(r)
		prevSpace = false
	}

	return strings.TrimSpace(b.String())
}
