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

	for _, r := range normalized {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			currentTerm.WriteRune(r)
		} else {
			addToken()
		}
	}
	addToken() // flush remaining

	return tokens
}
