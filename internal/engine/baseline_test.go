package engine

import "testing"

// baselineIndex builds a small corpus shaped like the one the thesis targets:
// Italian administrative documents, with identifiers, proper names and free
// text, so the four query classes can all be exercised.
func baselineIndex() (*InvertedIndex, Settings) {
	idx := NewInvertedIndex()
	s := DefaultSettings()
	s.SearchableFields = []string{"titolo", "testo"}
	s.FieldWeights = map[string]float64{"titolo": 2.0, "testo": 1.0}

	idx.AddDocument("d1", map[string]interface{}{
		"titolo": "Fattura 2026/0173 Rossi SpA",
		"testo":  "manutenzione impianto, imponibile 1.200 euro",
	}, s)
	idx.AddDocument("d2", map[string]interface{}{
		"titolo": "Contratto di manutenzione Rossi",
		"testo":  "durata annuale, rinnovo tacito",
	}, s)
	idx.AddDocument("d3", map[string]interface{}{
		"titolo": "DDT 2026/0173",
		"testo":  "consegna materiale Rossi SpA",
	}, s)
	idx.AddDocument("d4", map[string]interface{}{
		"titolo": "Delibera comunale",
		"testo":  "nessuna attinenza",
	}, s)

	return idx, s
}

func assertOrder(t *testing.T, got, atteso []string) {
	t.Helper()
	if len(got) != len(atteso) {
		t.Fatalf("attesi %v, ottenuti %v", atteso, got)
	}
	for i := range got {
		if got[i] != atteso[i] {
			t.Fatalf("attesi %v, ottenuti %v", atteso, got)
		}
	}
}

// TestBaselineRankingIsFrozen pins the order the engine produces TODAY.
// It is the thesis baseline: every later phase must keep it passing with the
// flags at their defaults, and the test itself must never be edited to make a
// change fit. If it goes red, the baseline moved.
//
// The `identificatore` class is missing on purpose: today it is not
// deterministic. See TestBaselineIdentifierTieIsNotDeterministic.
func TestBaselineRankingIsFrozen(t *testing.T) {
	idx, s := baselineIndex()

	casi := []struct {
		nome   string
		query  string
		atteso []string
	}{
		// d1 has both terms in `titolo` (weight 2), d3 has them in `testo`
		// (weight 1): 48 against 24. d2 does not appear at all, because
		// terms are ANDed and it has "Rossi" but not "SpA".
		{"entita", "Rossi SpA", []string{"d1", "d3"}},

		// d2 has it in `titolo` (24), d1 in `testo` (12).
		{"concetto", "manutenzione", []string{"d2", "d1"}},

		// Same pair with one typo: 18 and 9, the ratio holds, the typo costs
		// a constant factor. Typo tolerance works.
		{"refuso", "manutenzine", []string{"d2", "d1"}},
	}

	for _, c := range casi {
		t.Run(c.nome, func(t *testing.T) {
			got, _ := idx.Search(c.query, s, "auto", nil)
			assertOrder(t, got, c.atteso)
		})
	}
}

// TestBaselineIdentifierTieIsNotDeterministic documents a defect, it does not
// bless it.
//
// On "2026/0173" d1 and d3 come out with exactly the same score (48) and
// exactly the same tiebreakers (2 words matched, 0 typos, 2 exact matches).
// Results are collected from a map and sorted with sort.Slice, which is not
// stable, so their relative order changes between runs: over 30 runs, 26 gave
// [d1 d3] and 4 gave [d3 d1].
//
// This is why the class is not in the frozen test: there is nothing to freeze.
// Until ties break deterministically no evaluation is reproducible, because
// nDCG would move between runs with no change to the code.
//
// When the fix lands, this test is expected to fail and be replaced by a
// normal entry in TestBaselineRankingIsFrozen.
func TestBaselineIdentifierTieIsNotDeterministic(t *testing.T) {
	idx, s := baselineIndex()

	scored, _ := idx.SearchScored("2026/0173", s, "auto", nil)
	if len(scored) != 2 {
		t.Fatalf("attesi 2 risultati, ottenuti %d", len(scored))
	}

	visti := map[string]bool{}
	for _, r := range scored {
		visti[r.DocID] = true
	}
	if !visti["d1"] || !visti["d3"] {
		t.Fatalf("attesi d1 e d3, ottenuti %v", visti)
	}

	if scored[0].Score != scored[1].Score {
		t.Fatalf("i punteggi non sono piu' in parita' (%v e %v): il pareggio e' stato "+
			"risolto, aggiorna il baseline e sposta questo caso fra quelli congelati",
			scored[0].Score, scored[1].Score)
	}
}

// TestBaselineHybridIsFrozen does the same for the hybrid branch, which the
// plain corpus never reaches because none of its documents carry a vector.
//
// v1 scores 44 and v2 scores 24: both match "assistenza" in `titolo` for 24,
// and v1 takes a further 20 from a cosine of exactly 1. That +20 is the third
// defect in the flesh: a fixed constant added to a lexical score that grows
// without bound with query length. Here, on a single term, the vector decides
// the outcome on its own.
func TestBaselineHybridIsFrozen(t *testing.T) {
	idx, s := baselineIndex()
	idx.AddDocument("v1", map[string]interface{}{
		"titolo":  "Assistenza tecnica",
		"_vector": []interface{}{1.0, 0.0, 0.0},
	}, s)
	idx.AddDocument("v2", map[string]interface{}{
		"titolo":  "Assistenza legale",
		"_vector": []interface{}{0.0, 1.0, 0.0},
	}, s)

	got, _ := idx.Search("assistenza", s, "auto", []float64{1.0, 0.0, 0.0})
	assertOrder(t, got, []string{"v1", "v2"})
}
