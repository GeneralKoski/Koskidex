package engine

import (
	"testing"
)

func TestSearchAndRanker(t *testing.T) {
	idx := NewInvertedIndex()
	settings := Settings{}

	idx.AddDocument("1", map[string]interface{}{"title": "The Matrix"}, settings)
	idx.AddDocument("2", map[string]interface{}{"title": "The Godfather"}, settings)
	idx.AddDocument("3", map[string]interface{}{"title": "Goodfellas"}, settings)
	idx.AddDocument("4", map[string]interface{}{"title": "The Matrices"}, settings)

	// 1. Exact match test
	docs, _ := idx.Search("Matrix", settings)
	if len(docs) == 0 || docs[0] != "1" {
		t.Errorf("expected doc [1], got %v", docs)
	}

	// 2. Typo: "godfahter" -> Godfather (dist 1)
	docs, _ = idx.Search("godfahter", settings)
	if len(docs) == 0 || docs[0] != "2" {
		t.Errorf("expected doc [2], got %v", docs)
	}

	// 3. Highlight test
	hl := Highlight("The Godfather is a movie", []string{"godfather"})
	if hl != "The <em>Godfather</em> is a movie" {
		t.Errorf("highlight failed, got: %s", hl)
	}
}
