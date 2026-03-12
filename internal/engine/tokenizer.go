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
func Tokenize(text string, field string, stopwords map[string]bool) []Token {
	// 1. Single pass normalization (lowercase + remove accents)
	normalized := removeAccents(strings.ToLower(text))

	var tokens []Token
	var currentTerm strings.Builder
	position := 0

	addToken := func() {
		if currentTerm.Len() > 0 {
			term := currentTerm.String()
			if stopwords == nil || !stopwords[term] {
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
