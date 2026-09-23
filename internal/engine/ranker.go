package engine

import (
	"math"
	"sort"
	"strings"
)

type SearchMatch struct {
	DocID        string
	WordsMatched int
	Typos        int
	Score        float64
	ExactMatches int
}

type TokenDocMatch struct {
	DocID        string
	Typos        int
	ExactMatches int
	MaxWeight    float64

	// For BM25. MatchedTerm is the index term that actually matched, which is
	// not always the query token: with typo tolerance on, "manutenzine" can
	// match "manutenzione", and the document frequency that matters is the
	// one of the term found in the index. TF counts its occurrences in this
	// document.
	MatchedTerm string
	TF          int
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

func (idx *InvertedIndex) findDocsForToken(token Token, settings Settings, highlights map[string][]string, fuzziness string) map[string]*TokenDocMatch {
	maxTypos := MaxTypos(token.Term, settings.TypoTolerance, fuzziness)
	// Search already holds idx.mu (read); use the lock-free variant to avoid
	// recursive read-locking, which can deadlock against a concurrent writer.
	matchedTerms := idx.fuzzySearchTermsLocked(token.Term, maxTypos, false)

	tokenDocBest := make(map[string]*TokenDocMatch)

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

			weight := 1.0
			if w, ok := settings.FieldWeights[p.Field]; ok {
				weight = w
			}

			if _, ok := tokenDocBest[p.DocID]; !ok {
				tokenDocBest[p.DocID] = &TokenDocMatch{
					DocID: p.DocID, Typos: matchDist, MaxWeight: weight,
					MatchedTerm: mTerm,
				}
			} else {
				if matchDist < tokenDocBest[p.DocID].Typos {
					tokenDocBest[p.DocID].Typos = matchDist
					// The closer term wins: its statistics are the ones to use.
					tokenDocBest[p.DocID].MatchedTerm = mTerm
					tokenDocBest[p.DocID].TF = 0
				}
				if weight > tokenDocBest[p.DocID].MaxWeight {
					tokenDocBest[p.DocID].MaxWeight = weight
				}
			}

			// One posting is one occurrence, but only of the term currently
			// credited to this document.
			if tokenDocBest[p.DocID].MatchedTerm == mTerm {
				tokenDocBest[p.DocID].TF++
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

// SearchScored fuzzy searches and returns the ranked matches, scores included.
// This is the whole search: Search is only a thin wrapper over it.
func (idx *InvertedIndex) SearchScored(query string, settings Settings, fuzziness string, queryVector []float64) ([]SearchMatch, map[string][]string) {
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

	if len(allTokens) == 0 && len(pq.OrTerms) == 0 && len(queryVector) == 0 {
		return nil, nil
	}

	docMatches := make(map[string]*SearchMatch)
	highlights := make(map[string][]string)

	// Process must (AND) terms
	for _, token := range allTokens {
		tokenDocBest := idx.findDocsForToken(token, settings, highlights, fuzziness)

		for docID, match := range tokenDocBest {
			if _, ok := docMatches[docID]; !ok {
				docMatches[docID] = &SearchMatch{DocID: docID}
			}
			docMatches[docID].WordsMatched++
			docMatches[docID].Typos += match.Typos
			docMatches[docID].ExactMatches += match.ExactMatches

			docMatches[docID].Score += idx.tokenScoreLocked(match, settings)
		}
	}

	// Filter: how many of the must terms a document has to carry.
	// In RetrievalAny one is enough, so the filter below lets everything
	// through: a document is already in docMatches only if it matched at
	// least one term.
	requiredMatches := len(allTokens)
	if settings.RetrievalMode == RetrievalAny {
		requiredMatches = 1
	}
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
			tokenDocBest := idx.findDocsForToken(token, settings, highlights, fuzziness)
			for docID, match := range tokenDocBest {
				if _, ok := docMatches[docID]; !ok {
					docMatches[docID] = &SearchMatch{DocID: docID}
				}
				docMatches[docID].WordsMatched++
				docMatches[docID].Typos += match.Typos
				docMatches[docID].ExactMatches += match.ExactMatches

				docMatches[docID].Score += idx.tokenScoreLocked(match, settings)
			}
		}
	}

	// Process exclude (NOT) terms: remove matching docs
	if hasExclude {
		for _, token := range pq.ExcludeTerms {
			tokenDocBest := idx.findDocsForToken(token, settings, nil, fuzziness)
			for docID := range tokenDocBest {
				delete(docMatches, docID)
				delete(highlights, docID)
			}
		}
	}

	if len(queryVector) > 0 {
		if len(allTokens) == 0 && len(pq.OrTerms) == 0 {
			for docID, doc := range idx.docs {
				if vecVal, ok := doc["_vector"]; ok {
					if docVec, valid := toFloat64Array(vecVal); valid && len(docVec) == len(queryVector) {
						sim := cosineSimilarity(queryVector, docVec)
						docMatches[docID] = &SearchMatch{DocID: docID, Score: sim * 20.0}
					}
				}
			}
		} else {
			for docID, m := range docMatches {
				if doc, ok := idx.docs[docID]; ok {
					if vecVal, ok := doc["_vector"]; ok {
						if docVec, valid := toFloat64Array(vecVal); valid && len(docVec) == len(queryVector) {
							sim := cosineSimilarity(queryVector, docVec)
							m.Score += sim * 20.0
						}
					}
				}
			}
		}
	}

	var results []SearchMatch
	for _, m := range docMatches {
		results = append(results, *m)
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		if results[i].WordsMatched != results[j].WordsMatched {
			return results[i].WordsMatched > results[j].WordsMatched
		}
		if results[i].Typos != results[j].Typos {
			return results[i].Typos < results[j].Typos
		}
		if results[i].ExactMatches != results[j].ExactMatches {
			return results[i].ExactMatches > results[j].ExactMatches
		}
		// Last criterion, deliberately arbitrary: results come out of a map and
		// sort.Slice is not stable, so without this documents that tie on every
		// other criterion come back in a different order on every run, and no
		// evaluation is reproducible. This expresses no opinion on relevance:
		// the engine already considers them equivalent.
		return results[i].DocID < results[j].DocID
	})

	for docID, terms := range highlights {
		highlights[docID] = removeDuplicateTerms(terms)
	}

	return results, highlights
}

// Search returns only the document IDs, in the order SearchScored decided.
// Signature and behaviour are unchanged: this is the API the handlers call.
func (idx *InvertedIndex) Search(query string, settings Settings, fuzziness string, queryVector []float64) ([]string, map[string][]string) {
	results, highlights := idx.SearchScored(query, settings, fuzziness, queryVector)

	var docIDs []string
	for _, r := range results {
		docIDs = append(docIDs, r.DocID)
	}
	return docIDs, highlights
}

// tokenScoreLocked is one term's contribution to a document's score. The
// caller must hold idx.mu.
func (idx *InvertedIndex) tokenScoreLocked(match *TokenDocMatch, settings Settings) float64 {
	if settings.ScoringMode == ScoringBM25 {
		return idx.bm25Locked(match, settings) * match.MaxWeight
	}
	// Heuristic scorer, the behaviour up to now: no term frequency, no
	// document frequency, no length normalisation.
	return (10.0 - float64(match.Typos) + float64(match.ExactMatches*2)) * match.MaxWeight
}

// bm25Locked computes the Robertson/Lucene BM25 weight of one term in one
// document. The caller must hold idx.mu.
//
//	idf = ln(1 + (N - df + 0.5) / (df + 0.5))
//	w   = idf * tf*(k1+1) / (tf + k1*(1 - b + b*|D|/avgdl))
//
// The idf form is Lucene's, which adds 1 inside the logarithm so the weight
// can never go negative: the textbook form turns negative for a term present
// in more than half the collection, and a term would then subtract score for
// being common rather than merely add little.
func (idx *InvertedIndex) bm25Locked(match *TokenDocMatch, settings Settings) float64 {
	n := float64(len(idx.docs))
	if n == 0 {
		return 0
	}

	df := float64(idx.docFreq[match.MatchedTerm])
	idf := math.Log(1 + (n-df+0.5)/(df+0.5))

	tf := float64(match.TF)
	avgdl := idx.averageDocLengthLocked()
	norma := 1.0
	if avgdl > 0 {
		norma = 1 - settings.BM25B + settings.BM25B*float64(idx.docLengths[match.DocID])/avgdl
	}

	return idf * (tf * (settings.BM25K1 + 1)) / (tf + settings.BM25K1*norma)
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

// Highlight wraps every occurrence of the matched terms in <em>…</em>.
// matchedTerms come from the index, so they are already lowercased and
// accent-stripped; the text is normalized the same way (per rune, keeping a
// byte->original-rune map) so matching is accent-insensitive and the wrapping
// always lands on whole original runes (never slicing mid-rune).
func Highlight(text string, matchedTerms []string) string {
	if text == "" || len(matchedTerms) == 0 {
		return text
	}

	origRunes := []rune(text)

	var norm strings.Builder
	byteToRune := make([]int, 0, len(text))
	for i, r := range origRunes {
		n := removeAccents(strings.ToLower(string(r)))
		for b := 0; b < len(n); b++ {
			byteToRune = append(byteToRune, i)
		}
		norm.WriteString(n)
	}
	normStr := norm.String()

	highlighted := make([]bool, len(origRunes))
	for _, term := range matchedTerms {
		if term == "" {
			continue
		}
		from := 0
		for from <= len(normStr)-len(term) {
			at := strings.Index(normStr[from:], term)
			if at == -1 {
				break
			}
			start := from + at
			end := start + len(term) - 1
			if start < len(byteToRune) && end < len(byteToRune) {
				for ri := byteToRune[start]; ri <= byteToRune[end]; ri++ {
					highlighted[ri] = true
				}
			}
			from = start + len(term)
		}
	}

	var out strings.Builder
	open := false
	for i, r := range origRunes {
		if highlighted[i] && !open {
			out.WriteString("<em>")
			open = true
		} else if !highlighted[i] && open {
			out.WriteString("</em>")
			open = false
		}
		out.WriteRune(r)
	}
	if open {
		out.WriteString("</em>")
	}
	return out.String()
}

func cosineSimilarity(a, b []float64) float64 {
	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

func toFloat64Array(v interface{}) ([]float64, bool) {
	switch arr := v.(type) {
	case []float64:
		return arr, true
	case []interface{}:
		res := make([]float64, len(arr))
		for i, val := range arr {
			if f, ok := toFloat64(val); ok {
				res[i] = f
			} else {
				return nil, false
			}
		}
		return res, true
	}
	return nil, false
}
