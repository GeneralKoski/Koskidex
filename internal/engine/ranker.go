package engine

import (
	"sort"
	"strings"
)

// SearchMatch represents a single document's match details for a query
type SearchMatch struct {
	DocID        string
	WordsMatched int
	Typos        int
	Score        float64
	ExactMatches int
}

// ParsedQuery represents a parsed query with AND/OR/NOT semantics
type ParsedQuery struct {
	MustTerms    []Token // AND terms (all must match)
	OrTerms      []Token // OR terms (at least one must match)
	ExcludeTerms []Token // NOT terms (must not match)
}

// ParseQuery splits a raw query into must/or/exclude terms.
// Syntax: "term1 term2" = AND, "term1 OR term2" = OR, "-term" = NOT
func ParseQuery(raw string, stopWords map[string]bool) ParsedQuery {
	var pq ParsedQuery
	words := strings.Fields(raw)

	for i := 0; i < len(words); i++ {
		word := words[i]

		// OR operator
		if word == "OR" && i+1 < len(words) {
			next := words[i+1]
			tokens := Tokenize(next, "", stopWords)
			pq.OrTerms = append(pq.OrTerms, tokens...)
			// Also move previous must term to OR if it was the last added
			if len(pq.MustTerms) > 0 {
				last := pq.MustTerms[len(pq.MustTerms)-1]
				pq.MustTerms = pq.MustTerms[:len(pq.MustTerms)-1]
				pq.OrTerms = append(pq.OrTerms, last)
			}
			i++
			continue
		}

		// NOT operator (prefix -)
		if strings.HasPrefix(word, "-") && len(word) > 1 {
			tokens := Tokenize(word[1:], "", stopWords)
			pq.ExcludeTerms = append(pq.ExcludeTerms, tokens...)
			continue
		}

		// Regular AND term
		tokens := Tokenize(word, "", stopWords)
		pq.MustTerms = append(pq.MustTerms, tokens...)
	}

	return pq
}

func (idx *InvertedIndex) findDocsForToken(token Token, settings Settings, highlights map[string][]string) map[string]*SearchMatch {
	maxTypos := MaxTypos(token.Term, settings.TypoTolerance)
	matchedTerms := idx.FuzzySearchTerms(token.Term, maxTypos, false)

	tokenDocBest := make(map[string]*SearchMatch)

	for _, mTerm := range matchedTerms {
		dist := DamerauLevenshtein(token.Term, mTerm)

		isPrefix := false
		if len(token.Term) >= 2 && len(mTerm) > len(token.Term) {
			if mTerm[:len(token.Term)] == token.Term {
				isPrefix = true
			}
		}

		postings := idx.index[mTerm]

		for _, p := range postings {
			matchDist := dist
			if isPrefix && dist > 0 {
				_ = matchDist // prefix match tracking
			}

			if _, ok := tokenDocBest[p.DocID]; !ok {
				tokenDocBest[p.DocID] = &SearchMatch{DocID: p.DocID, Typos: matchDist}
			} else {
				if matchDist < tokenDocBest[p.DocID].Typos {
					tokenDocBest[p.DocID].Typos = matchDist
				}
			}

			if dist == 0 {
				tokenDocBest[p.DocID].ExactMatches = 1
			}

			if highlights != nil {
				highlights[p.DocID] = append(highlights[p.DocID], mTerm)
			}
		}
	}

	return tokenDocBest
}

// Search fuzzy searches and returns ranked document IDs
func (idx *InvertedIndex) Search(query string, settings Settings) ([]string, map[string][]string) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	pq := ParseQuery(query, settings.StopWords)
	hasOR := len(pq.OrTerms) > 0
	hasExclude := len(pq.ExcludeTerms) > 0

	// If no special operators, use all tokens as must terms (original behavior)
	allTokens := pq.MustTerms
	if !hasOR && !hasExclude {
		allTokens = Tokenize(query, "", settings.StopWords)
	}

	if len(allTokens) == 0 && len(pq.OrTerms) == 0 {
		return nil, nil
	}

	docMatches := make(map[string]*SearchMatch)
	highlights := make(map[string][]string)

	// Process must (AND) terms
	for _, token := range allTokens {
		tokenDocBest := idx.findDocsForToken(token, settings, highlights)

		for docID, match := range tokenDocBest {
			if _, ok := docMatches[docID]; !ok {
				docMatches[docID] = &SearchMatch{DocID: docID}
			}
			docMatches[docID].WordsMatched++
			docMatches[docID].Typos += match.Typos
			docMatches[docID].ExactMatches += match.ExactMatches
		}
	}

	// Filter: only docs matching ALL must terms
	requiredMatches := len(allTokens)
	if requiredMatches > 0 {
		for docID, m := range docMatches {
			if m.WordsMatched < requiredMatches {
				delete(docMatches, docID)
			}
		}
	}

	// Process OR terms: add docs that match at least one OR term
	if hasOR {
		for _, token := range pq.OrTerms {
			tokenDocBest := idx.findDocsForToken(token, settings, highlights)
			for docID, match := range tokenDocBest {
				if _, ok := docMatches[docID]; !ok {
					docMatches[docID] = &SearchMatch{DocID: docID}
				}
				docMatches[docID].WordsMatched++
				docMatches[docID].Typos += match.Typos
				docMatches[docID].ExactMatches += match.ExactMatches
			}
		}
	}

	// Process exclude (NOT) terms: remove matching docs
	if hasExclude {
		for _, token := range pq.ExcludeTerms {
			tokenDocBest := idx.findDocsForToken(token, settings, nil)
			for docID := range tokenDocBest {
				delete(docMatches, docID)
				delete(highlights, docID)
			}
		}
	}

	var results []SearchMatch
	for _, m := range docMatches {
		m.Score = float64(m.WordsMatched*10) - float64(m.Typos) + float64(m.ExactMatches*2)
		results = append(results, *m)
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].WordsMatched != results[j].WordsMatched {
			return results[i].WordsMatched > results[j].WordsMatched
		}
		if results[i].Typos != results[j].Typos {
			return results[i].Typos < results[j].Typos
		}
		return results[i].ExactMatches > results[j].ExactMatches
	})

	var docIDs []string
	for _, r := range results {
		docIDs = append(docIDs, r.DocID)
	}

	for docID, terms := range highlights {
		highlights[docID] = removeDuplicateTerms(terms)
	}

	return docIDs, highlights
}

func removeDuplicateTerms(terms []string) []string {
	seen := make(map[string]bool)
	var final []string
	for _, t := range terms {
		if !seen[t] {
			final = append(final, t)
			seen[t] = true
		}
	}
	return final
}

// Highlight replaces matched terms in a text string with <em>tags</em>
func Highlight(text string, matchedTerms []string) string {
	res := text
	for _, term := range matchedTerms {
		// Simple case insensitive replace
		// A full implementation would use regex or a token-aware replacer to avoid partial word matches
		lowerRes := strings.ToLower(res)
		idx := strings.Index(lowerRes, term)
		if idx != -1 {
			orig := res[idx : idx+len(term)]
			res = res[:idx] + "<em>" + orig + "</em>" + res[idx+len(term):]
		}
	}
	return res
}
