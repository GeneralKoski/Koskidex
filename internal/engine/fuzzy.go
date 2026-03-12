package engine

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

// FuzzySearch lookup matching terms from prefix logic
func (idx *InvertedIndex) FuzzySearchTerms(queryTerm string, maxDistance int, exactness bool) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	var matchedTerms []string

	// Direct match optimization
	if _, ok := idx.index[queryTerm]; ok {
		matchedTerms = append(matchedTerms, queryTerm)
		if exactness {
			// If we only want exact matches, return here
			return matchedTerms
		}
	}

	prefix := getPrefix(queryTerm)
	candidates := idx.prefixMap[prefix]

	for _, candidate := range candidates {
		if candidate == queryTerm {
			continue // Already handled
		}

		dist := DamerauLevenshtein(queryTerm, candidate)
		if dist <= maxDistance {
			matchedTerms = append(matchedTerms, candidate)
		}
	}

	return matchedTerms
}

// MaxTypos is standard logic for allowed typos based on word length
func MaxTypos(term string, settings TypoSettings) int {
	if !settings.Enabled {
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
