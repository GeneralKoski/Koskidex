package engine

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// DamerauLevenshtein calculates the distance between two strings
// allowing transposition of adjacent characters (e.g. teh -> the).
func DamerauLevenshtein(a, b string) int {
	rA, rB := []rune(a), []rune(b)
	lenA, lenB := len(rA), len(rB)

	if lenA == 0 {
		return lenB
	}
	if lenB == 0 {
		return lenA
	}

	d := make([][]int, lenA+1)
	for i := range d {
		d[i] = make([]int, lenB+1)
		d[i][0] = i
	}
	for j := range d[0] {
		d[0][j] = j
	}

	for i := 1; i <= lenA; i++ {
		for j := 1; j <= lenB; j++ {
			cost := 1
			if rA[i-1] == rB[j-1] {
				cost = 0
			}

			// substitution, insertion, deletion
			d[i][j] = min3(
				d[i-1][j]+1,      // deletion
				d[i][j-1]+1,      // insertion
				d[i-1][j-1]+cost, // substitution
			)

			// transposition
			if i > 1 && j > 1 && rA[i-1] == rB[j-2] && rA[i-2] == rB[j-1] {
				d[i][j] = min2(d[i][j], d[i-2][j-2]+cost)
			}
		}
	}

	return d[lenA][lenB]
}

func min2(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func min3(a, b, c int) int {
	return min2(a, min2(b, c))
}

// FuzzySearchTerms looks up terms matching queryTerm within maxDistance edits.
// It takes the read lock; internal callers that already hold it must use
// fuzzySearchTermsLocked instead to avoid recursive read-locking (which can
// deadlock against a concurrent writer).
func (idx *InvertedIndex) FuzzySearchTerms(queryTerm string, maxDistance int, exactness bool) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.fuzzySearchTermsLocked(queryTerm, maxDistance, exactness, false, 0)
}

// fuzzySearchTermsLocked is the lock-free body of FuzzySearchTerms. Caller must
// hold idx.mu (read or write). noPrefix and prefixLength are
// Settings.DisablePrefixSearch and Settings.PrefixLength.
func (idx *InvertedIndex) fuzzySearchTermsLocked(queryTerm string, maxDistance int, exactness bool, noPrefix bool, prefixLength int) []string {
	var matchedTerms []string

	// Direct match optimization
	if _, ok := idx.index[queryTerm]; ok {
		matchedTerms = append(matchedTerms, queryTerm)
		if exactness {
			// If we only want exact matches, return here
			return matchedTerms
		}
	}

	// As in Elasticsearch, a prefix longer than the query word is the whole word.
	queryRunes := []rune(queryTerm)
	exact := queryRunes
	if prefixLength < len(exact) {
		exact = exact[:prefixLength]
	}

	consider := func(candidate string) {
		if candidate == queryTerm {
			return // Already handled
		}

		// Prefix matching: if candidate starts with queryTerm, it's a match regardless of distance
		if !noPrefix && len(queryTerm) >= 2 && len(candidate) > len(queryTerm) {
			if candidate[:len(queryTerm)] == queryTerm {
				matchedTerms = append(matchedTerms, candidate)
				return
			}
		}

		if len(exact) > 0 && !strings.HasPrefix(candidate, string(exact)) {
			return
		}

		// The distance is at least the difference in length: skip the matrix.
		if d := utf8.RuneCountInString(candidate) - len(queryRunes); d > maxDistance || -d > maxDistance {
			return
		}

		dist := DamerauLevenshtein(queryTerm, candidate)
		if dist <= maxDistance {
			matchedTerms = append(matchedTerms, candidate)
		}
	}

	// Two words within maxDistance edits share at least (n-1) - 3*maxDistance
	// of the n-1 bigrams of the query word: an edit breaks up to two, a
	// transposition three. When that bound drops below one, a term within
	// reach may share no bigram ("atre" and "arte", "187" and "17"), and the
	// bigram lookup would never look at it: the whole vocabulary is scanned
	// instead. It happens only for short words with a typo allowed.
	if maxDistance > 0 && len(queryRunes)-1-3*maxDistance < 1 {
		for candidate := range idx.index {
			consider(candidate)
		}
		return matchedTerms
	}

	// Gather candidates from every bigram of the query term, not just the first
	// one: documents index all their bigrams, so probing only the leading bigram
	// would miss matches whose typo lands in the first two characters.
	for _, candidate := range idx.fuzzyCandidates(queryTerm) {
		consider(candidate)
	}
	return matchedTerms
}

// fuzzyCandidates returns the distinct terms sharing at least one bigram with
// queryTerm, mirroring how addDocumentLocked indexes every bigram into
// prefixMap. Caller must hold idx.mu.
func (idx *InvertedIndex) fuzzyCandidates(queryTerm string) []string {
	seen := make(map[string]bool)
	var candidates []string
	add := func(term string) {
		if !seen[term] {
			seen[term] = true
			candidates = append(candidates, term)
		}
	}

	runes := []rune(queryTerm)
	if len(runes) < 2 {
		for _, t := range idx.prefixMap[queryTerm] {
			add(t)
		}
		return candidates
	}
	for i := 0; i <= len(runes)-2; i++ {
		bigram := string(runes[i : i+2])
		for _, t := range idx.prefixMap[bigram] {
			add(t)
		}
	}
	return candidates
}

// MaxTypos is standard logic for allowed typos based on word length
func MaxTypos(term string, settings TypoSettings, fuzziness string) int {
	if fuzziness == "0" {
		return 0
	}
	if fuzziness == "1" {
		return 1
	}
	if fuzziness == "2" {
		return 2
	}
	if !settings.Enabled {
		return 0
	}
	if settings.DisableOnNumbers && soloCifre(term) {
		return 0
	}
	l := len([]rune(term))
	if l < settings.MinWordLengthOneTypo {
		return 0
	}
	if l < settings.MinWordLengthTwoTypos {
		return 1
	}
	return 2
}

func soloCifre(term string) bool {
	for _, r := range term {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return term != ""
}
