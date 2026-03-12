package engine

import (
	"sort"
	"strings"
)

// SearchMatch represents a single document's match details for a query
type SearchMatch struct {
	DocID      string
	WordsMatched int
	Typos        int
	Score        float64
	ExactMatches int
	// We could track proximity and field weights, but keep it simple
}

// Search fuzzy searches and returns ranked document IDs
func (idx *InvertedIndex) Search(query string, settings Settings) ([]string, map[string][]string) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	queryTokens := Tokenize(query, "", settings.StopWords)
	if len(queryTokens) == 0 {
		return nil, nil
	}

	docMatches := make(map[string]*SearchMatch)
	highlights := make(map[string][]string)

	for _, token := range queryTokens {
		maxTypos := MaxTypos(token.Term, settings.TypoTolerance)
		// For simplicity, we assume we match the whole word exactly if maxTypos=0, otherwise fuzzy
		// Find all matching terms in dictionary
		matchedTerms := idx.FuzzySearchTerms(token.Term, maxTypos, false)

		// Calculate best match per document for this token
		tokenDocBest := make(map[string]*SearchMatch)

		for _, mTerm := range matchedTerms {
			dist := DamerauLevenshtein(token.Term, mTerm)
			postings := idx.index[mTerm]

			for _, p := range postings {
				if _, ok := tokenDocBest[p.DocID]; !ok {
					tokenDocBest[p.DocID] = &SearchMatch{DocID: p.DocID, Typos: dist}
				} else {
					if dist < tokenDocBest[p.DocID].Typos {
						tokenDocBest[p.DocID].Typos = dist
					}
				}
				
				// Exact match logic
				if dist == 0 {
					tokenDocBest[p.DocID].ExactMatches = 1
				}

				// Basic highlight tracking: track the matched term
				highlights[p.DocID] = append(highlights[p.DocID], mTerm)
			}
		}

		// Merge token matches into overall query matches
		for docID, match := range tokenDocBest {
			if _, ok := docMatches[docID]; !ok {
				docMatches[docID] = &SearchMatch{DocID: docID}
			}
			docMatches[docID].WordsMatched++
			docMatches[docID].Typos += match.Typos
			docMatches[docID].ExactMatches += match.ExactMatches
		}
	}

	var results []SearchMatch
	for _, m := range docMatches {
		// Calculate a basic score: more words = better, fewer typos = better
		m.Score = float64(m.WordsMatched*10) - float64(m.Typos) + float64(m.ExactMatches*2)
		results = append(results, *m)
	}

	// Tie-breaking ranker:
	// 1. Words matched (descending)
	// 2. Typos (ascending)
	// 3. Exact Match (descending)
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
		// Dedup highlights if needed (simplified)
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
