package engine

import (
	"math"
	"testing"
)

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

func TestBoostsOutOfBoundsAreRefused(t *testing.T) {
	for _, b := range []Boost{
		{Filter: "", Factor: 2}, {Filter: "kind", Factor: 2}, {Filter: "kind=zone"}, {Filter: "kind=zone", Factor: -1},
		{Filter: "kind=zone", Factor: 1e308}, {Filter: "kind=zone", Factor: math.Inf(1)}, {Filter: "kind=zone", Factor: math.NaN()},
		{Filter: "kind=zone", Factor: 5e-324}, {Filter: "kind=zone", Factor: MaxBoostFactor * 1.01},
		{Filter: "distance(geo)<1", Factor: 2}, {Filter: "garbage, kind=zone", Factor: 2},
	} {
		if ValidaBoosts([]Boost{b}) == nil {
			t.Errorf("%+v must be refused", b)
		}
	}
	if ValidaBoosts(make([]Boost, MaxBoosts+1)) == nil {
		t.Error("more than MaxBoosts must be refused")
	}
	ok := []Boost{{Filter: "kind=zone", Factor: 1.5}, {Filter: "kind=zone,store_id=1", Factor: MaxBoostFactor}, {Filter: "distance(_geo,45.4,9.1)<500", Factor: 1 / MaxBoostFactor}}
	if err := ValidaBoosts(ok); err != nil {
		t.Error(err)
	}
}

// A negative score, from a vector, moves up with the boost, not down.
func TestABoostLiftsANegativeScore(t *testing.T) {
	idx := NewInvertedIndex()
	s := DefaultSettings()
	idx.AddDocument("p", map[string]interface{}{"id": "p", "kind": "product", "_vector": []interface{}{-1.0, 0.0}}, s)
	idx.AddDocument("z", map[string]interface{}{"id": "z", "kind": "zone", "_vector": []interface{}{-1.0, 0.2}}, s)
	s.Boosts = []Boost{{Filter: "kind=zone", Factor: 2}}
	got, _ := idx.SearchScored("", s, "", []float64{1, 0})
	if got[0].DocID != "z" || got[0].Score <= got[1].Score {
		t.Fatalf("the boosted zone must stay first: %v", got)
	}
}
