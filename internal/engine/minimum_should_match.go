package engine

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var spaziAttornoAlMinore = regexp.MustCompile(`\s*<\s*`)

// RequiredTerms is how many of n query terms a document must hold under an
// Elasticsearch minimum_should_match spec, computed as Elasticsearch's
// Queries.calculateMinShouldMatch: an integer, a negative integer (all but),
// a percentage rounded down, a negative percentage, or conditions "n<spec"
// applied in order. Never less than 1 nor more than n. An empty spec is 1.
func RequiredTerms(spec string, n int) (int, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return min(1, n), nil
	}
	r, err := calcolaMinimo(spec, n)
	if err != nil {
		return 0, err
	}
	return max(1, min(r, n)), nil
}

func calcolaMinimo(spec string, n int) (int, error) {
	if !strings.Contains(spec, "<") {
		return calcolaSemplice(spec, n)
	}
	risultato := n
	for _, parte := range strings.Fields(spaziAttornoAlMinore.ReplaceAllString(spec, "<")) {
		pezzi := strings.Split(parte, "<")
		if len(pezzi) != 2 {
			return 0, fmt.Errorf("minimum_should_match %q: condizione %q non valida", spec, parte)
		}
		soglia, err := strconv.Atoi(pezzi[0])
		if err != nil {
			return 0, fmt.Errorf("minimum_should_match %q: soglia %q non intera", spec, pezzi[0])
		}
		calcolo, err := calcolaSemplice(pezzi[1], n)
		if err != nil {
			return 0, err
		}
		if n <= soglia {
			return risultato, nil
		}
		risultato = calcolo
	}
	return risultato, nil
}

func calcolaSemplice(spec string, n int) (int, error) {
	if percentuale, ok := strings.CutSuffix(spec, "%"); ok {
		p, err := strconv.Atoi(percentuale)
		if err != nil {
			return 0, fmt.Errorf("minimum_should_match: percentuale %q non intera", spec)
		}
		calcolo := n * p / 100
		if p < 0 {
			return n + calcolo, nil
		}
		return calcolo, nil
	}
	v, err := strconv.Atoi(spec)
	if err != nil {
		return 0, fmt.Errorf("minimum_should_match: %q non è un intero né una percentuale", spec)
	}
	if v < 0 {
		return n + v, nil
	}
	return v, nil
}
