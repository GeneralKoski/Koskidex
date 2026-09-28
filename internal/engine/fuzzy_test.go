package engine

import (
	"math/rand"
	"reflect"
	"sort"
	"testing"
)

func TestDamerauLevenshtein(t *testing.T) {
	tests := []struct {
		a    string
		b    string
		want int
	}{
		{"", "", 0},
		{"a", "", 1},
		{"", "a", 1},
		{"hello", "hello", 0},
		{"teh", "the", 1},       // transposition
		{"hello", "helo", 1},    // deletion
		{"hello", "helllo", 1},  // insertion
		{"kitten", "sitting", 3},// substitutions
		{"godfather", "godfahter", 1}, // transposition
	}

	for _, tt := range tests {
		got := DamerauLevenshtein(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("DL(%q, %q) = %d; want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

// damerauLevenshteinMatrice is the full-matrix version the three-row one
// replaced, kept as the reference it must agree with.
func damerauLevenshteinMatrice(a, b string) int {
	rA, rB := []rune(a), []rune(b)
	d := make([][]int, len(rA)+1)
	for i := range d {
		d[i] = make([]int, len(rB)+1)
		d[i][0] = i
	}
	for j := range d[0] {
		d[0][j] = j
	}
	for i := 1; i <= len(rA); i++ {
		for j := 1; j <= len(rB); j++ {
			cost := 1
			if rA[i-1] == rB[j-1] {
				cost = 0
			}
			d[i][j] = min3(d[i-1][j]+1, d[i][j-1]+1, d[i-1][j-1]+cost)
			if i > 1 && j > 1 && rA[i-1] == rB[j-2] && rA[i-2] == rB[j-1] {
				d[i][j] = min2(d[i][j], d[i-2][j-2]+cost)
			}
		}
	}
	return d[len(rA)][len(rB)]
}

func TestDamerauLevenshteinComeLaMatrice(t *testing.T) {
	alfabeto := []rune("abcaeèéiìoòuùß日本\uFFFD")
	r := rand.New(rand.NewSource(1))
	parola := func() string {
		// up to 40 runes, past the 32 that fit on the stack
		n := r.Intn(41)
		out := make([]rune, n)
		for i := range out {
			out[i] = alfabeto[r.Intn(len(alfabeto))]
		}
		return string(out)
	}
	for i := 0; i < 200000; i++ {
		a, b := parola(), parola()
		if rb := []rune(a); i%3 == 0 && len(rb) > 1 {
			k := r.Intn(len(rb) - 1)
			rb[k], rb[k+1] = rb[k+1], rb[k]
			b = string(rb)
		}
		if got, want := DamerauLevenshtein(a, b), damerauLevenshteinMatrice(a, b); got != want {
			t.Fatalf("DL(%q, %q) = %d; la matrice dà %d", a, b, got, want)
		}
	}
	for _, c := range [][2]string{{"\xff\xfe", "\xfe\xff"}, {"a\xffb", "ab"}} {
		if got, want := DamerauLevenshtein(c[0], c[1]), damerauLevenshteinMatrice(c[0], c[1]); got != want {
			t.Fatalf("DL(%q, %q) = %d; la matrice dà %d", c[0], c[1], got, want)
		}
	}
}

func TestFuzzySearchTerms(t *testing.T) {
	idx := NewInvertedIndex()
	settings := Settings{
		SearchableFields: nil, // index all fields
		StopWords:        nil,
	}

	idx.AddDocument("1", map[string]interface{}{"title": "the godfather"}, settings)
	idx.AddDocument("2", map[string]interface{}{"title": "goodfellas"}, settings)
	idx.AddDocument("3", map[string]interface{}{"title": "godzilla"}, settings)

	// godfahter -> godfather (dist 1)
	terms := idx.FuzzySearchTerms("godfahter", 1, false)
	sort.Strings(terms)
	
	expected := []string{"godfather"}
	if !reflect.DeepEqual(terms, expected) {
		t.Errorf("got %v, want %v", terms, expected)
	}

	// goodfellas with a typo
	terms2 := idx.FuzzySearchTerms("godfellas", 1, false)
	if len(terms2) != 1 || terms2[0] != "goodfellas" {
		t.Errorf("expected [goodfellas], got %v", terms2)
	}
}

func TestFuzzyCandidatesComePrima(t *testing.T) {
	prima := func(idx *InvertedIndex, queryTerm string) []string {
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
			for _, t := range idx.prefixMap[string(runes[i:i+2])] {
				add(t)
			}
		}
		return candidates
	}

	idx := NewInvertedIndex()
	testi := []string{"il padrino e i quei bravi ragazzi", "città perché più già", "godfather goodfellas godzilla", "日本語 テキスト"}
	for i, testo := range testi {
		idx.AddDocument(string(rune('a'+i)), map[string]interface{}{"title": testo}, Settings{})
	}
	for _, q := range []string{"", "g", "go", "godfahter", "città", "pià", "日本", "a\xffb", "\xff\xfe", "zzz"} {
		got, want := idx.fuzzyCandidates(q), prima(idx, q)
		if len(got) != len(want) || (len(want) > 0 && !reflect.DeepEqual(got, want)) {
			t.Errorf("fuzzyCandidates(%q) = %v; prima %v", q, got, want)
		}
	}
}
