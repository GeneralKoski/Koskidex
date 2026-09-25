package engine

import (
	"reflect"
	"testing"
)

// Tre documenti: il primo condivide la parola con la query, il secondo parla
// della stessa cosa con altre parole ed è vicino nel vettore, il terzo è
// lontano in tutto.
func indiceIbrido(s Settings) *InvertedIndex {
	idx := NewInvertedIndex()
	idx.AddDocument("lessicale", map[string]interface{}{"text": "contratto di manutenzione", "_vector": []interface{}{1.0, 0.0}}, s)
	idx.AddDocument("semantico", map[string]interface{}{"text": "accordo di assistenza tecnica", "_vector": []interface{}{0.98, 0.2}}, s)
	idx.AddDocument("lontano", map[string]interface{}{"text": "delibera sul bilancio", "_vector": []interface{}{0.0, 1.0}}, s)
	return idx
}

func impostazioniIbride(modo string, k int) Settings {
	s := DefaultSettings()
	s.SearchableFields = []string{"text"}
	s.HybridMode = modo
	s.VectorTopK = k
	return s
}

func TestUnionBringsInTheSemanticDocument(t *testing.T) {
	s := impostazioniIbride(HybridUnion, 2)
	got, _ := indiceIbrido(s).Search("contratto", s, "0", []float64{1.0, 0.0})
	if want := []string{"lessicale", "semantico"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("con union attesi %v, ottenuti %v", want, got)
	}
}

// Il baseline: senza HybridMode il vettore riordina soltanto.
func TestRerankStaysTheDefault(t *testing.T) {
	s := impostazioniIbride("", 2)
	got, _ := indiceIbrido(s).Search("contratto", s, "0", []float64{1.0, 0.0})
	if want := []string{"lessicale"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("senza HybridMode attesi %v, ottenuti %v", want, got)
	}
}

func TestVectorModeIgnoresTheLexicalMatch(t *testing.T) {
	s := impostazioniIbride(HybridVector, 2)
	got, _ := indiceIbrido(s).Search("delibera", s, "0", []float64{1.0, 0.0})
	if want := []string{"lessicale", "semantico"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("con vector attesi i due più vicini, %v, ottenuti %v", want, got)
	}
}

// In union i candidati lessicali hanno lo stesso punteggio del re-ranking:
// cambia solo chi entra, non come si fonde.
func TestUnionKeepsTheRerankScores(t *testing.T) {
	r := impostazioniIbride("", 1)
	// K abbastanza grande perché anche il documento lessicale sia fra i
	// vicini: il suo punteggio non deve essere sostituito da sim * 20.
	u := impostazioniIbride(HybridUnion, 3)
	q := []float64{0.0, 1.0}
	rr, _ := indiceIbrido(r).SearchScored("contratto", r, "0", q)
	uu, _ := indiceIbrido(u).SearchScored("contratto", u, "0", q)
	if len(rr) != 1 || len(uu) != 3 {
		t.Fatalf("attesi 1 risultato in rerank e 3 in union: %v, %v", rr, uu)
	}
	punti := map[string]float64{}
	for _, m := range uu {
		punti[m.DocID] = m.Score
	}
	if punti["lessicale"] != rr[0].Score {
		t.Fatalf("punteggio del lessicale diverso fra rerank (%v) e union (%v)", rr[0].Score, punti["lessicale"])
	}
	if punti["lontano"] != 20 {
		t.Fatalf("un documento del solo vettore vale sim * 20, cioè 20: %v", punti["lontano"])
	}
}

// Zero vuol dire 100, come la profondità di Recall@100; un vettore di
// dimensione diversa si salta, come oggi.
func TestVectorTopKDefaultAndWrongDimensions(t *testing.T) {
	s := impostazioniIbride(HybridVector, 0)
	idx := NewInvertedIndex()
	for i := 0; i < 150; i++ {
		idx.AddDocument(string(rune('a'+i%26))+string(rune('a'+i/26)), map[string]interface{}{"text": "x", "_vector": []interface{}{1.0, float64(i)}}, s)
	}
	idx.AddDocument("corto", map[string]interface{}{"text": "x", "_vector": []interface{}{1.0}}, s)
	got, _ := idx.Search("niente", s, "0", []float64{1.0, 0.0})
	if len(got) != 100 {
		t.Fatalf("attesi 100 candidati con VectorTopK zero, ottenuti %d", len(got))
	}
	for _, id := range got {
		if id == "corto" {
			t.Fatal("un vettore di dimensione diversa va saltato")
		}
	}
}
