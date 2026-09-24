package engine

import "testing"

// Documentale manda tags, subjects e additional_data come liste. Prima
// finivano nel documento salvato ma non nell'indice, in silenzio: una parola
// che stava solo in una lista non si trovava.
func TestAListFieldIsIndexedElementByElement(t *testing.T) {
	idx := NewInvertedIndex()
	s := DefaultSettings()
	s.SearchableFields = []string{"summary", "subjects"}
	s.TypoTolerance.Enabled = false

	idx.AddDocument("8", map[string]interface{}{
		"summary":  "Fornitura energia elettrica",
		"subjects": []interface{}{"Enel Energia", "Comune di Crispiano"},
	}, s)

	for _, q := range []string{"enel", "crispiano"} {
		if got, _ := idx.Search(q, s, "0", nil); len(got) != 1 {
			t.Fatalf("%q sta solo nella lista subjects e va trovata: %v", q, got)
		}
	}
}

// Senza campi dichiarati, Koskidex indicizza da solo i campi stringa. Una
// lista di stringhe e' testo quanto una stringa.
func TestAListFieldIsSearchableWithoutDeclaringIt(t *testing.T) {
	idx := NewInvertedIndex()
	s := DefaultSettings()
	s.TypoTolerance.Enabled = false

	idx.AddDocument("1", map[string]interface{}{"tags": []interface{}{"fattura", "manutenzione"}}, s)

	if got, _ := idx.Search("manutenzione", s, "0", nil); len(got) != 1 {
		t.Fatalf("un campo lista deve essere cercabile anche senza dichiararlo: %v", got)
	}
}

// additional_data porta i valori dei campi estratti, che possono essere numeri.
func TestNumbersInsideAListAreIndexed(t *testing.T) {
	idx := NewInvertedIndex()
	s := DefaultSettings()
	s.SearchableFields = []string{"additional_data"}
	s.TypoTolerance.Enabled = false

	idx.AddDocument("1", map[string]interface{}{"additional_data": []interface{}{"Area Tecnica", float64(1223), true}}, s)

	if got, _ := idx.Search("1223", s, "0", nil); len(got) != 1 {
		t.Fatalf("un numero dentro una lista va indicizzato: %v", got)
	}
}

// Un numero da solo e' testo solo se il campo e' dichiarato cercabile: senza
// campi dichiarati, prezzi e anni di un catalogo non devono diventare
// cercabili di colpo.
func TestAScalarNumberIsIndexedOnlyInADeclaredField(t *testing.T) {
	dichiarato, libero := NewInvertedIndex(), NewInvertedIndex()
	s := DefaultSettings()
	s.TypoTolerance.Enabled = false
	sd := s
	sd.SearchableFields = []string{"numero"}

	doc := map[string]interface{}{"numero": float64(187), "titolo": "ordinanza"}
	dichiarato.AddDocument("1", doc, sd)
	libero.AddDocument("1", doc, s)

	if got, _ := dichiarato.Search("187", sd, "0", nil); len(got) != 1 {
		t.Fatalf("in un campo dichiarato il numero va indicizzato: %v", got)
	}
	if got, _ := libero.Search("187", s, "0", nil); len(got) != 0 {
		t.Fatalf("senza campi dichiarati un numero non deve diventare cercabile: %v", got)
	}
}

// Gli elementi di una lista non sono una frase: l'ultima parola di uno non
// deve risultare vicina alla prima del successivo. Stesso salto che usa
// Elasticsearch (position_increment_gap = 100).
func TestListElementsAreNotAdjacent(t *testing.T) {
	s := DefaultSettings()
	tok, _ := tokenizzaValore([]interface{}{"acme srl", "enel energia"}, "subjects", s, true)

	pos := map[string]int{}
	for _, t := range tok {
		pos[t.Term] = t.Position
	}
	if pos["enel"]-pos["srl"] < distanzaFraElementi {
		t.Fatalf("fra due elementi servono almeno %d posizioni, ci sono %d (%v)", distanzaFraElementi, pos["enel"]-pos["srl"], pos)
	}
}

// BM25 normalizza per lunghezza: le parole delle liste devono contare.
func TestListTokensCountTowardsTheDocumentLength(t *testing.T) {
	idx := NewInvertedIndex()
	s := DefaultSettings()
	s.SearchableFields = []string{"tags"}

	idx.AddDocument("1", map[string]interface{}{"tags": []interface{}{"uno due", "tre"}}, s)

	if l := idx.DocLength("1"); l != 3 {
		t.Fatalf("attesi 3 token, contati %d", l)
	}
}
