package engine

import "testing"

// Korsia, "pese": the zone ties with two products and loses on the id.
func TestABoostBreaksATieByFieldValue(t *testing.T) {
	idx := NewInvertedIndex()
	s := DefaultSettings()
	s.SearchableFields = []string{"name"}
	for id, d := range map[string][2]string{"a": {"Pepe rosa", "product"}, "b": {"Pere Abate", "product"}, "z": {"Pesa", "zone"}, "c": {"Pesce spada", "product"}} {
		idx.AddDocument(id, map[string]interface{}{"id": id, "name": d[0], "kind": d[1]}, s)
	}
	prima, _ := idx.SearchScored("pese", s, "", nil)
	if prima[0].DocID == "z" || prima[0].Score != prima[len(prima)-1].Score {
		t.Fatalf("without a boost all tie and the zone loses on the id: %v", prima)
	}

	s.Boosts = []Boost{{Filter: "kind=zone", Factor: 1.01}}
	dopo, _ := idx.SearchScored("pese", s, "", nil)
	if dopo[0].DocID != "z" {
		t.Fatalf("the zone must come first: %v", dopo)
	}
	punteggi := map[string]float64{}
	for _, m := range prima {
		punteggi[m.DocID] = m.Score
	}
	for _, m := range dopo {
		want := punteggi[m.DocID]
		if m.DocID == "z" {
			want *= 1.01
		}
		if m.Score != want {
			t.Errorf("%s: score %v, want %v", m.DocID, m.Score, want)
		}
	}
}

func TestBoostsWithoutAFilterOrAPositiveFactorAreRefused(t *testing.T) {
	for _, b := range []Boost{{Filter: "", Factor: 2}, {Filter: "kind", Factor: 2}, {Filter: "kind=zone"}, {Filter: "kind=zone", Factor: -1}} {
		if ValidaBoosts([]Boost{b}) == nil {
			t.Errorf("%+v must be refused", b)
		}
	}
	if err := ValidaBoosts([]Boost{{Filter: "kind=zone", Factor: 1.5}}); err != nil {
		t.Error(err)
	}
}
