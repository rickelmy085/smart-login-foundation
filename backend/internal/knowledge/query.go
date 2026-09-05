package knowledge

import (
	"strings"
)

// ptStopwords contém as palavras mais comuns do português que
// devem ser ignoradas em buscas lexicais. Usa a forma normalizada
// (sem acentos, minúscula) para casar com o tokenizer unicode61
// remove_diacritics 2 do FTS5.
var ptStopwords = map[string]struct{}{
	"a": {}, "ao": {}, "aos": {}, "as": {},
	"com": {}, "como": {},
	"da": {}, "das": {}, "de": {}, "do": {}, "dos": {},
	"e": {}, "em": {}, "entre": {}, "esta": {}, "este": {}, "estes": {}, "estas": {},
	"foi": {},
	"ja":  {},
	"la":  {}, "lhe": {}, "lo": {}, "lhos": {},
	"mais": {}, "mas": {}, "me": {}, "mesmo": {}, "meu": {}, "minha": {}, "muito": {},
	"na": {}, "nas": {}, "nao": {}, "no": {}, "nos": {}, "nossa": {}, "nosso": {}, "num": {}, "numa": {},
	"o": {}, "os": {}, "ou": {}, "para": {}, "pela": {}, "pelas": {}, "pelo": {}, "pelos": {}, "por": {}, "qual": {}, "quando": {},
	"se": {}, "sem": {}, "seu": {}, "sua": {}, "somos": {}, "suas": {}, "sou": {},
	"tambem": {}, "te": {}, "tem": {}, "tinha": {}, "tua": {}, "tuas": {}, "tudo": {},
	"um": {}, "uma": {}, "umas": {}, "uns": {},
	"voce": {}, "voces": {},
}

// normalizeRemoveAccents normaliza uma palavra removendo acentos
// para casar com o comportamento do tokenizer unicode61.
func normalizeRemoveAccents(w string) string {
	return strings.NewReplacer(
		"á", "a", "à", "a", "ã", "a", "â", "a", "ä", "a",
		"é", "e", "ê", "e",
		"í", "i",
		"ó", "o", "ô", "o", "õ", "o", "ö", "o",
		"ú", "u", "ü", "u",
		"ç", "c",
	).Replace(w)
}

// significantTerms extrai termos significativos de uma consulta:
// remove stopwords, palavras curtas (<4 chars) e normaliza acentos.
func significantTerms(q string) []string {
	words := strings.Fields(strings.ToLower(q))
	var out []string
	for _, w := range words {
		normalized := normalizeRemoveAccents(w)
		normalized = strings.Trim(normalized, ".,;:!?\"'()[]{}")
		if len(normalized) < 4 {
			continue
		}
		if _, skip := ptStopwords[normalized]; skip {
			continue
		}
		out = append(out, normalized)
	}
	return out
}

// significantTermsString devolve os termos significativos como uma
// string pronta para usar em MATCH do FTS5 (separados por OR).
func significantTermsString(q string) string {
	terms := significantTerms(q)
	if len(terms) == 0 {
		return ""
	}
	return strings.Join(terms, " OR ")
}
