package engine

import "testing"

// Un numero cercato, "190", in sei atti, e un codice raro che comincia con le
// stesse cifre, in uno solo. "esatto" e "prefisso" sono lunghi uguali: l'unica
// differenza fra loro è il termine.
func indiceEspansioni(espansioni string) (*InvertedIndex, Settings) {
	idx := NewInvertedIndex()
	s := DefaultSettings()
	s.SearchableFields = []string{"t"}
	s.ScoringMode = ScoringBM25
	s.RetrievalMode = RetrievalAny
	s.TypoTolerance.Enabled = false
	s.BM25Expansion = espansioni
	docs := map[string]string{
		"esatto": "190 bari", "prefisso": "1900129 lecce",
		"a": "190 roma", "b": "190 pisa", "c": "190 asti", "d": "190 lodi", "e": "190 como",
		"f": "cane", "g": "gatto", "h": "topo",
	}
	for id, t := range docs {
		idx.AddDocument(id, map[string]interface{}{"t": t}, s)
	}
	return idx, s
}

func punteggi(idx *InvertedIndex, q string, s Settings) (map[string]float64, []string) {
	r, _ := idx.SearchScored(q, s, "0", nil)
	out, ordine := map[string]float64{}, []string{}
	for _, m := range r {
		out[m.DocID] = m.Score
		ordine = append(ordine, m.DocID)
	}
	return out, ordine
}

func TestBM25ExpansionIsOffByDefault(t *testing.T) {
	if DefaultSettings().BM25Expansion != "" {
		t.Fatal("la frequenza mescolata deve essere spenta per default: il baseline deve restare misurabile")
	}
}

// Il difetto: il codice raro trovato per prefisso pesa con il suo IDF, e passa
// davanti all'atto che contiene il numero esatto.
func TestBM25WeightsAPrefixExpansionWithTheRareTermFound(t *testing.T) {
	idx, s := indiceEspansioni("")
	p, _ := punteggi(idx, "190", s)
	if p["prefisso"] <= p["esatto"] {
		t.Fatalf("oggi il prefisso raro deve battere il numero esatto (e' il difetto da correggere): %v", p)
	}
}

// La correzione: le espansioni condividono la frequenza più alta fra i termini
// trovati, quindi a parità del resto il codice raro pesa quanto il numero, e
// il match esatto vince lo spareggio.
func TestBM25BlendedGivesAnExpansionNoMoreWeightThanTheExactTerm(t *testing.T) {
	idx, s := indiceEspansioni("blended")
	p, ordine := punteggi(idx, "190", s)
	quasi(t, p["prefisso"], p["esatto"])
	pos := map[string]int{}
	for i, id := range ordine {
		pos[id] = i
	}
	if pos["esatto"] > pos["prefisso"] {
		t.Fatalf("a punteggio pari il match esatto viene prima: %v", ordine)
	}
}

// La frequenza mescolata è quella del più comune dei termini trovati: qui
// df("190") = 6, quindi l'IDF di entrambi è quello di "190".
func TestBM25BlendedUsesTheHighestDocumentFrequencyAmongTheTermsFound(t *testing.T) {
	oggi, s0 := indiceEspansioni("")
	mescolato, s1 := indiceEspansioni("blended")
	p0, _ := punteggi(oggi, "190", s0)
	p1, _ := punteggi(mescolato, "190", s1)
	// Il termine esatto è già il più comune: il suo punteggio non cambia.
	quasi(t, p1["esatto"], p0["esatto"])
	if p1["prefisso"] >= p0["prefisso"] {
		t.Fatalf("il codice raro deve perdere peso: era %v, e' %v", p0["prefisso"], p1["prefisso"])
	}
}

// Cambiano i punteggi, non cosa si trova.
func TestBM25BlendedDoesNotChangeTheResultSet(t *testing.T) {
	oggi, s0 := indiceEspansioni("")
	mescolato, s1 := indiceEspansioni("blended")
	for _, q := range []string{"190", "190 lecce", "19", "cane 190"} {
		p0, _ := punteggi(oggi, q, s0)
		p1, _ := punteggi(mescolato, q, s1)
		if len(p0) != len(p1) {
			t.Fatalf("%q: insiemi diversi, %d contro %d", q, len(p0), len(p1))
		}
		for id := range p0 {
			if _, ok := p1[id]; !ok {
				t.Fatalf("%q: %s sparito con la frequenza mescolata", q, id)
			}
		}
	}
}

// L'impostazione riguarda solo BM25: il punteggio euristico non ha IDF.
func TestBM25ExpansionDoesNotTouchTheLegacyScorer(t *testing.T) {
	oggi, s0 := indiceEspansioni("")
	mescolato, s1 := indiceEspansioni("blended")
	s0.ScoringMode, s1.ScoringMode = ScoringLegacy, ScoringLegacy
	p0, _ := punteggi(oggi, "190", s0)
	p1, _ := punteggi(mescolato, "190", s1)
	for id, v := range p0 {
		if p1[id] != v {
			t.Fatalf("con l'euristico nessun punteggio deve cambiare: %s era %v, e' %v", id, v, p1[id])
		}
	}
}

// With BM25ExpansionSynonym a document's TF is the occurrences of every
// expansion it holds, so the credited term, and the order the terms are
// visited in, no longer matter.
func TestEspansioniSinonimo(t *testing.T) {
	testi := map[string]string{"a": "cani cani cani cant", "b": "cant cant cani", "c": "casa cane"}
	for _, ordine := range [][]string{{"a", "b", "c"}, {"c", "b", "a"}} {
		settings := DefaultSettings()
		settings.ScoringMode = ScoringBM25
		settings.BM25Expansion = BM25ExpansionSynonym
		settings.RetrievalMode = RetrievalAny
		idx := NewInvertedIndex()
		for _, id := range ordine {
			idx.AddDocument(id, map[string]interface{}{"title": testi[id]}, settings)
		}
		idx.mu.RLock()
		trovati := idx.docsForTermsLocked(Token{Term: "canx"}, idx.matchedTermsLocked(Token{Term: "canx"}, settings, "1"), settings, nil, nil, 0)
		idx.mu.RUnlock()
		for id, tf := range map[string]int{"a": 4, "b": 3, "c": 1} {
			if trovati[id] == nil || trovati[id].TF != tf {
				t.Errorf("ordine %v, %s: TF %v, atteso %d", ordine, id, trovati[id], tf)
			}
		}
	}
}
