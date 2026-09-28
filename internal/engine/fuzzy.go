package engine

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// DamerauLevenshtein calculates the distance between two strings
// allowing transposition of adjacent characters (e.g. teh -> the).
func DamerauLevenshtein(a, b string) int {
	// The recurrence reads only the two rows above the current one, so three
	// rows are enough. For the short words of a query they, and the runes,
	// live in arrays on the stack: the whole matrix was allocated on every
	// call, and this runs once per candidate term of every search.
	var bufA, bufB [32]rune
	rA, rB := appendRunes(bufA[:0], a), appendRunes(bufB[:0], b)
	lenA, lenB := len(rA), len(rB)

	if lenA == 0 {
		return lenB
	}
	if lenB == 0 {
		return lenA
	}

	var bufRighe [3 * 33]int
	righe := bufRighe[:0]
	if 3*(lenB+1) <= len(bufRighe) {
		righe = bufRighe[:3*(lenB+1)]
	} else {
		righe = make([]int, 3*(lenB+1))
	}
	dueSopra, sopra, riga := righe[:lenB+1], righe[lenB+1:2*(lenB+1)], righe[2*(lenB+1):]
	for j := range sopra {
		sopra[j] = j
	}

	for i := 1; i <= lenA; i++ {
		riga[0] = i
		for j := 1; j <= lenB; j++ {
			cost := 1
			if rA[i-1] == rB[j-1] {
				cost = 0
			}

			// substitution, insertion, deletion
			v := min3(
				sopra[j]+1,      // deletion
				riga[j-1]+1,     // insertion
				sopra[j-1]+cost, // substitution
			)

			// transposition
			if i > 1 && j > 1 && rA[i-1] == rB[j-2] && rA[i-2] == rB[j-1] {
				v = min2(v, dueSopra[j-2]+cost)
			}
			riga[j] = v
		}
		dueSopra, sopra, riga = sopra, riga, dueSopra
	}

	return sopra[lenB]
}

// appendRunes is []rune(s) into dst, so that dst can be a stack array.
func appendRunes(dst []rune, s string) []rune {
	for _, r := range s {
		dst = append(dst, r)
	}
	return dst
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

	exactStr := string(exact)
	// Prefix matching: if candidate starts with queryTerm, it's a match regardless of distance
	esteso := func(candidate string) bool {
		return !noPrefix && len(queryTerm) >= 2 && len(candidate) > len(queryTerm) &&
			candidate[:len(queryTerm)] == queryTerm
	}
	// passa is everything consider checks before the distance. It depends on
	// the candidate alone, so fuzzyCandidates can apply it before dropping
	// the duplicates and give back the same terms in the same order.
	passa := func(candidate string) bool {
		if candidate == queryTerm {
			return false // Already handled
		}
		if esteso(candidate) {
			return true
		}
		if len(exact) > 0 && !strings.HasPrefix(candidate, exactStr) {
			return false
		}
		// The distance is at least the difference in length: skip the matrix.
		d := utf8.RuneCountInString(candidate) - len(queryRunes)
		return d <= maxDistance && -d <= maxDistance
	}
	consider := func(candidate string) {
		if passa(candidate) && (esteso(candidate) || DamerauLevenshtein(queryTerm, candidate) <= maxDistance) {
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
	for _, candidate := range idx.fuzzyCandidates(queryTerm, passa) {
		consider(candidate)
	}
	return matchedTerms
}

// fuzzyCandidates returns the distinct terms sharing at least one bigram with
// queryTerm, mirroring how addDocumentLocked indexes every bigram into
// prefixMap, in order of first appearance, keeping only those passa accepts
// (all of them when passa is nil). Caller must hold idx.mu.
func (idx *InvertedIndex) fuzzyCandidates(queryTerm string, passa func(string) bool) []string {
	if passa == nil {
		passa = func(string) bool { return true }
	}
	runes := []rune(queryTerm)
	if len(runes) < 2 {
		return dedup(idx.prefixMap[queryTerm], passa)
	}
	// A bigram is two runes; in valid UTF-8, as the tokenizer produces, it is
	// the substring between their offsets, with no conversion to allocate.
	var bufInizi [64]int
	inizi := bufInizi[:0]
	for i := range queryTerm {
		inizi = append(inizi, i)
	}
	inizi = append(inizi, len(queryTerm))
	valido := utf8.ValidString(queryTerm)
	bigramma := func(k int) string {
		if valido {
			return queryTerm[inizi[k]:inizi[k+2]]
		}
		return string(runes[k : k+2])
	}

	// The lists of common bigrams hold thousands of terms and passa rejects
	// nearly all of them: checking it first keeps the map of the terms seen
	// as small as the answer, instead of as large as the lists.
	seen := make(map[string]struct{})
	var candidates []string
	for k := 0; k+2 < len(inizi); k++ {
		for _, t := range idx.prefixMap[bigramma(k)] {
			if !passa(t) {
				continue
			}
			if _, ok := seen[t]; !ok {
				seen[t] = struct{}{}
				candidates = append(candidates, t)
			}
		}
	}
	return candidates
}

// dedup keeps the first occurrence of each term passa accepts, in order.
func dedup(terms []string, passa func(string) bool) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, t := range terms {
		if !passa(t) {
			continue
		}
		if _, ok := seen[t]; !ok {
			seen[t] = struct{}{}
			out = append(out, t)
		}
	}
	return out
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
