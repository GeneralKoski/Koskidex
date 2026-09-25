package engine

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Token represents a single parsed term from a document
type Token struct {
	Term     string
	Position int
	Field    string
}

// removeAccents strips diacritical marks from letters (e.g., è -> e)
func removeAccents(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, _ := transform.String(t, s)
	return result
}

// Tokenize processes a string, removes stops, lowercases, and splits by non-letter/number characters
//
// It takes the whole Settings rather than the stop words alone so that a new
// stage - today the stemmer - cannot reach some call sites and miss others.
// There are seven of them, across indexing, query parsing and search, and if
// the index were built with a stage the query does not apply, the index would
// hold terms the query can no longer produce. The symptom would be "it finds
// nothing", with no error anywhere, which is the hardest kind to trace back.
// With Settings in the signature, a call site left behind does not compile.
func Tokenize(text string, field string, settings Settings) []Token {
	stopwords := settings.StopWords
	// L'ordine e' quello di EnglishAnalyzer di Lucene: prima si tolgono le
	// stopword, poi si stemma. Invertirlo cambierebbe i risultati, perche' la
	// lista di stopword e' fatta di parole intere e non di radici.
	norm := analizzatore(settings.Stemmer)

	// 1. Single pass normalization (lowercase + remove accents)
	normalized := removeAccents(strings.ToLower(text))

	var tokens []Token
	var currentTerm strings.Builder
	position := 0

	addToken := func() {
		if currentTerm.Len() > 0 {
			term := currentTerm.String()
			if stopwords == nil || !stopwords[term] {
				if norm != nil {
					term = norm.Normalize(term)
				}
				tokens = append(tokens, Token{
					Term:     term,
					Position: position,
					Field:    field,
				})
			}
			currentTerm.Reset()
			position++
		}
	}

	standard := settings.Tokenizer == TokenizerStandard
	rs := []rune(normalized)
	for i, r := range rs {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || (standard && dentroLaParola(rs, i)) {
			currentTerm.WriteRune(r)
		} else {
			addToken()
		}
	}
	addToken() // flush remaining

	return tokens
}

// dentroLaParola says whether the punctuation at rs[i] stays inside the word
// under TokenizerStandard, following the UAX#29 rules Elasticsearch applies:
// WB6-7 between letters, WB11-12 between digits, WB13a-b for the underscore.
func dentroLaParola(rs []rune, i int) bool {
	parola := func(j int) bool {
		return j >= 0 && j < len(rs) && (unicode.IsLetter(rs[j]) || unicode.IsNumber(rs[j]) || rs[j] == '_')
	}
	r := rs[i]
	if r == '_' {
		return parola(i-1) || parola(i+1)
	}
	if i == 0 || i == len(rs)-1 {
		return false
	}
	prima, dopo := rs[i-1], rs[i+1]
	switch r {
	case '\'', '\u2018', '\u2019', '.':
		return (unicode.IsLetter(prima) && unicode.IsLetter(dopo)) || (unicode.IsNumber(prima) && unicode.IsNumber(dopo))
	case ':', '\u00b7':
		return unicode.IsLetter(prima) && unicode.IsLetter(dopo)
	case ',', ';':
		return unicode.IsNumber(prima) && unicode.IsNumber(dopo)
	}
	return false
}
