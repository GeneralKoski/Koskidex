package engine

import (
	"reflect"
	"testing"
)

func TestInvertedIndex(t *testing.T) {
	idx := NewInvertedIndex()
	settings := Settings{
		SearchableFields: []string{"title", "genre"},
		StopWords:        map[string]bool{"the": true, "a": true},
	}

	// Add Document 1
	doc1 := map[string]interface{}{
		"id":    "1",
		"title": "The Matrix",
		"genre": "sci-fi",
	}
	idx.AddDocument("1", doc1, settings)

	// Add Document 2
	doc2 := map[string]interface{}{
		"id":    "2",
		"title": "The Godfather",
		"genre": "drama",
	}
	idx.AddDocument("2", doc2, settings)

	// Add Document 3
	doc3 := map[string]interface{}{
		"id":    "3",
		"title": "The Matrix Reloaded",
		"genre": "sci-fi",
	}
	idx.AddDocument("3", doc3, settings)

	if count := idx.GetDocCount(); count != 3 {
		t.Errorf("Expected 3 docs, got %d", count)
	}

	// Search exact single term
	res := idx.SearchExact("Matrix", settings)
	if len(res) != 2 {
		t.Errorf("Expected 2 results for 'Matrix', got %d: %v", len(res), res)
	}

	// Check if elements 1 and 3 are present
	has1 := false
	has3 := false
	for _, r := range res {
		if r == "1" { has1 = true }
		if r == "3" { has3 = true }
	}
	if !has1 || !has3 {
		t.Errorf("Expected results to contain 1 and 3 for 'Matrix', got %v", res)
	}

	// Search exact two terms
	res2 := idx.SearchExact("Matrix Reloaded", settings)
	if len(res2) != 1 || res2[0] != "3" {
		t.Errorf("Expected [3] for 'Matrix Reloaded', got %v", res2)
	}

	// Stopword search
	res3 := idx.SearchExact("The", settings)
	if len(res3) != 0 {
		t.Errorf("Expected 0 results for stopword 'The', got %v", res3)
	}

	// Get document
	retrieved, ok := idx.GetDocument("2")
	if !ok {
		t.Errorf("Expected to find document 2")
	}
	if !reflect.DeepEqual(retrieved, doc2) {
		t.Errorf("Retrieved doc mismatch. Got %v, want %v", retrieved, doc2)
	}

	// Test prefix map population
	prefixes := idx.prefixMap["ma"]
	if len(prefixes) == 0 {
		t.Errorf("Expected prefix map to contain 'ma' keys")
	}
	hasMatrix := false
	for _, p := range prefixes {
		if p == "matrix" { hasMatrix = true }
	}
	if !hasMatrix {
		t.Errorf("Expected prefix 'ma' to map to 'matrix'")
	}
}
