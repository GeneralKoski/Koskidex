package engine

import "testing"

func scoredTestIndex() (*InvertedIndex, Settings) {
	idx := NewInvertedIndex()
	settings := DefaultSettings()
	settings.SearchableFields = []string{"title"}

	idx.AddDocument("a", map[string]interface{}{"title": "gatto nero"}, settings)
	idx.AddDocument("b", map[string]interface{}{"title": "gatto"}, settings)
	idx.AddDocument("c", map[string]interface{}{"title": "cane"}, settings)

	return idx, settings
}

// SearchScored must return the same documents, in the same order, that Search
// returns: Search is only a wrapper that drops everything but the IDs.
func TestSearchScoredMatchesSearchOrder(t *testing.T) {
	idx, settings := scoredTestIndex()

	ids, _ := idx.Search("gatto nero", settings, "auto", nil)
	scored, _ := idx.SearchScored("gatto nero", settings, "auto", nil)

	if len(ids) != len(scored) {
		t.Fatalf("lunghezze diverse: Search %d, SearchScored %d", len(ids), len(scored))
	}
	for i := range ids {
		if ids[i] != scored[i].DocID {
			t.Fatalf("posizione %d: Search dice %q, SearchScored dice %q", i, ids[i], scored[i].DocID)
		}
	}
}

// The score is the object of the whole thesis: it has to actually come out.
func TestSearchScoredExposesTheScore(t *testing.T) {
	idx, settings := scoredTestIndex()

	scored, _ := idx.SearchScored("gatto nero", settings, "auto", nil)
	if len(scored) == 0 {
		t.Fatal("nessun risultato")
	}
	if scored[0].Score <= 0 {
		t.Fatalf("il primo risultato ha punteggio %v, atteso maggiore di zero", scored[0].Score)
	}
	for i := 1; i < len(scored); i++ {
		if scored[i-1].Score < scored[i].Score {
			t.Fatalf("punteggi non decrescenti: posizione %d ha %v, posizione %d ha %v",
				i-1, scored[i-1].Score, i, scored[i].Score)
		}
	}
}

// Both must deduplicate the highlights, not just Search.
func TestSearchScoredDeduplicatesHighlights(t *testing.T) {
	idx, settings := scoredTestIndex()

	_, fromSearch := idx.Search("gatto", settings, "auto", nil)
	_, fromScored := idx.SearchScored("gatto", settings, "auto", nil)

	for docID, terms := range fromScored {
		seen := map[string]bool{}
		for _, term := range terms {
			if seen[term] {
				t.Fatalf("documento %q: termine %q duplicato negli highlight", docID, term)
			}
			seen[term] = true
		}
	}
	if len(fromSearch) != len(fromScored) {
		t.Fatalf("highlight per %d documenti da Search, %d da SearchScored",
			len(fromSearch), len(fromScored))
	}
}
