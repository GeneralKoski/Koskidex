package engine

import (
	"math"
	"math/rand"
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

// Un vettore arrivato come []interface{}, come lo decodificano JSON e gob,
// si converte una volta sola all'aggiunta: la ricerca non lo ricopia per ogni
// documento.
func TestAVectorIsStoredAsFloats(t *testing.T) {
	s := impostazioniIbride("", 2)
	idx := indiceIbrido(s)
	doc, _ := idx.GetDocument("semantico")
	if v, ok := doc["_vector"].([]float64); !ok || !reflect.DeepEqual(v, []float64{0.98, 0.2}) {
		t.Fatalf("_vector salvato come %#v, atteso []float64{0.98, 0.2}", doc["_vector"])
	}
}

// La conversione non cambia niente: stessi punteggi, al bit, con i vettori dati
// come []interface{} o come []float64, in ogni modo di usare il vettore.
func TestVectorScoresDoNotDependOnTheVectorType(t *testing.T) {
	comeFloat := func(s Settings) *InvertedIndex {
		idx := NewInvertedIndex()
		idx.AddDocument("lessicale", map[string]interface{}{"text": "contratto di manutenzione", "_vector": []float64{1.0, 0.0}}, s)
		idx.AddDocument("semantico", map[string]interface{}{"text": "accordo di assistenza tecnica", "_vector": []float64{0.98, 0.2}}, s)
		idx.AddDocument("lontano", map[string]interface{}{"text": "delibera sul bilancio", "_vector": []float64{0.0, 1.0}}, s)
		return idx
	}
	q := []float64{0.6, 0.8}
	for _, s := range []Settings{impostazioniIbride("", 3), impostazioniIbride(HybridUnion, 3), impostazioniIbride(HybridVector, 3),
		func() Settings { s := impostazioniIbride("", 3); s.FusionMode = FusionRRF; return s }(),
		func() Settings { s := impostazioniIbride("", 3); s.FusionMode = FusionConvex; return s }()} {
		for _, query := range []string{"contratto", ""} {
			a, _ := indiceIbrido(s).SearchScored(query, s, "0", q)
			b, _ := comeFloat(s).SearchScored(query, s, "0", q)
			if !reflect.DeepEqual(a, b) {
				t.Fatalf("modo %q fusione %q query %q: %v contro %v", s.HybridMode, s.FusionMode, query, a, b)
			}
		}
	}
}

// La similarità con le norme calcolate prima dà gli stessi bit della formula
// che le ricalcolava a ogni documento, qui copiata com'era.
func TestPrecomputedNormsGiveTheSameBits(t *testing.T) {
	vecchia := func(a, b []float64) float64 {
		var dotProduct, normA, normB float64
		for i := range a {
			dotProduct += a[i] * b[i]
			normA += a[i] * a[i]
			normB += b[i] * b[i]
		}
		if normA == 0 || normB == 0 {
			return 0
		}
		return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
	}
	r := rand.New(rand.NewSource(7))
	for n := 0; n < 1000; n++ {
		a, b := make([]float64, 1024), make([]float64, 1024)
		for i := range a {
			a[i], b[i] = r.NormFloat64(), r.NormFloat64()*3
		}
		if n == 0 {
			b = make([]float64, 1024)
		}
		if got, want := similarita(a, norma(a), b, norma(b)), vecchia(a, b); got != want {
			t.Fatalf("vettori %d: %v invece di %v", n, got, want)
		}
	}
}

// Un documento aggiunto di nuovo con un altro vettore prende la norma nuova.
func TestAnUpdatedVectorGetsItsNorm(t *testing.T) {
	s := impostazioniIbride("", 1)
	idx := NewInvertedIndex()
	idx.AddDocument("x", map[string]interface{}{"text": "atto", "_vector": []interface{}{1.0, 0.0}}, s)
	idx.AddDocument("x", map[string]interface{}{"text": "atto", "_vector": []interface{}{3.0, 4.0}}, s)
	ms, _ := idx.SearchScored("", s, "0", []float64{0.0, 1.0})
	if len(ms) != 1 {
		t.Fatalf("atteso un documento, ottenuti %v", ms)
	}
	circa(t, ms[0].Score, 0.8*20, "similarità col vettore nuovo, per il peso 20")
}
