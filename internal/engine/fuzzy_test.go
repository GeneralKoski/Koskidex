package engine

import (
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
