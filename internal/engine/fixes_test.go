package engine

import (
	"strings"
	"sync"
	"testing"
)

// Re-adding an existing document id must replace it, not accumulate stale
// postings from the previous version.
func TestUpdateReplacesStalePostings(t *testing.T) {
	idx := NewInvertedIndex()
	settings := Settings{SearchableFields: []string{"title"}}

	idx.AddDocument("1", map[string]interface{}{"id": "1", "title": "matrix"}, settings)
	if res := idx.SearchExact("matrix", settings); len(res) != 1 {
		t.Fatalf("expected 1 hit for 'matrix' before update, got %v", res)
	}

	// Update the same id with different content.
	idx.AddDocument("1", map[string]interface{}{"id": "1", "title": "gladiator"}, settings)

	if count := idx.GetDocCount(); count != 1 {
		t.Fatalf("expected 1 doc after update, got %d", count)
	}
	if res := idx.SearchExact("matrix", settings); len(res) != 0 {
		t.Fatalf("stale term 'matrix' should be gone after update, got %v", res)
	}
	if res := idx.SearchExact("gladiator", settings); len(res) != 1 || res[0] != "1" {
		t.Fatalf("expected [1] for 'gladiator' after update, got %v", res)
	}
}

// A typo in the first two characters must still find the term (candidates are
// gathered from every bigram, not just the leading one).
func TestFuzzyFirstCharTypo(t *testing.T) {
	idx := NewInvertedIndex()
	settings := Settings{SearchableFields: []string{"brand"}}
	idx.AddDocument("1", map[string]interface{}{"id": "1", "brand": "samsung"}, settings)

	terms := idx.FuzzySearchTerms("xamsung", 1, false)
	found := false
	for _, tm := range terms {
		if tm == "samsung" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected 'samsung' to match 'xamsung' within 1 typo, got %v", terms)
	}
}

func TestHighlightAccentInsensitiveAndMultiple(t *testing.T) {
	// Accent-insensitive: matched term "cafe" (accent-stripped by the tokenizer)
	// must still wrap the accented original.
	out := Highlight("Café Society", []string{"cafe"})
	if !strings.Contains(out, "<em>Café</em>") {
		t.Fatalf("expected accented word highlighted, got %q", out)
	}

	// Every occurrence is wrapped, not just the first.
	out = Highlight("cat CAT", []string{"cat"})
	if strings.Count(out, "<em>") != 2 {
		t.Fatalf("expected both occurrences highlighted, got %q", out)
	}
}

// Search run concurrently with writers must not deadlock or race. This would
// hang on the previous code, which recursively read-locked idx.mu during search.
func TestSearchConcurrentWithWrites(t *testing.T) {
	idx := NewInvertedIndex()
	settings := DefaultSettings()
	settings.SearchableFields = []string{"title"}

	for i := 0; i < 50; i++ {
		idx.AddDocument(string(rune('a'+i%26))+"1", map[string]interface{}{
			"id": "seed", "title": "interstellar gladiator matrix",
		}, settings)
	}

	var wg sync.WaitGroup
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				idx.Search("matrx gladiatr", settings, "AUTO", nil)
			}
		}()
	}
	for w := 0; w < 2; w++ {
		wg.Add(1)
		go func(base int) {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				id := "doc" + string(rune('a'+base)) + string(rune('0'+i%10))
				idx.AddDocument(id, map[string]interface{}{"id": id, "title": "matrix reloaded"}, settings)
				idx.DeleteDocument(id)
			}
		}(w)
	}
	wg.Wait()
}
