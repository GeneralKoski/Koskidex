package engine

import "testing"

// Le impostazioni che avvicinano il matching a quello di Elasticsearch in
// Documentale (multi_match best_fields, operator and, fuzziness AUTO,
// prefix_length 1). Tutte spente per default: il baseline del 23/09 deve
// restare misurabile, e un indice salvato prima non deve cambiare.
func TestElasticsearchCompatibilityIsOffByDefault(t *testing.T) {
	s := DefaultSettings()
	if s.DisablePrefixSearch || s.PrefixLength != 0 || s.AllTermsInOneField {
		t.Fatalf("le impostazioni di compatibilità devono essere spente per default: %+v", s)
	}
}

func cercaIn(t *testing.T, s Settings, docs map[string]map[string]interface{}, q string) map[string]bool {
	t.Helper()
	idx := NewInvertedIndex()
	for id, d := range docs {
		idx.AddDocument(id, d, s)
	}
	got, _ := idx.Search(q, s, "AUTO", nil)
	out := map[string]bool{}
	for _, id := range got {
		out[id] = true
	}
	return out
}

// Koskidex accetta qualunque termine che comincia con la parola cercata, a
// qualsiasi distanza; Elasticsearch no.
func TestDisablePrefixSearchKeepsOnlyTypoMatches(t *testing.T) {
	docs := map[string]map[string]interface{}{"1": {"t": "manutenzione"}}
	s := DefaultSettings()
	if !cercaIn(t, s, docs, "manut")["1"] {
		t.Fatal("per default \"manut\" trova \"manutenzione\" per prefisso")
	}
	s.DisablePrefixSearch = true
	if cercaIn(t, s, docs, "manut")["1"] {
		t.Fatal("senza ricerca per prefisso \"manut\" non deve trovare \"manutenzione\"")
	}
	if !cercaIn(t, s, docs, "manutenzion")["1"] {
		t.Fatal("senza ricerca per prefisso i refusi restano: \"manutenzion\" dista uno")
	}
	if !cercaIn(t, s, docs, "manutenzione")["1"] {
		t.Fatal("senza ricerca per prefisso la parola esatta resta")
	}
}

// prefix_length di Elasticsearch: i primi N caratteri di un termine trovato con
// un refuso devono essere esatti.
func TestPrefixLengthKeepsTheFirstCharactersExact(t *testing.T) {
	docs := map[string]map[string]interface{}{"1": {"t": "casa"}}
	s := DefaultSettings()
	for _, q := range []string{"vasa", "cosa"} {
		if !cercaIn(t, s, docs, q)["1"] {
			t.Fatalf("per default %q trova \"casa\" con un refuso", q)
		}
	}
	s.PrefixLength = 1
	if cercaIn(t, s, docs, "vasa")["1"] {
		t.Fatal("con prefix_length 1 \"vasa\" non deve trovare \"casa\": la prima lettera è diversa")
	}
	if !cercaIn(t, s, docs, "cosa")["1"] {
		t.Fatal("con prefix_length 1 \"cosa\" trova \"casa\": la prima lettera è uguale")
	}
	s.PrefixLength = 2
	if cercaIn(t, s, docs, "cosa")["1"] {
		t.Fatal("con prefix_length 2 \"cosa\" non deve trovare \"casa\": la seconda lettera è diversa")
	}
	if !cercaIn(t, s, docs, "casa")["1"] {
		t.Fatal("prefix_length non tocca la parola esatta")
	}
}

// prefix_length conta caratteri, non byte. Le lettere accentate italiane non
// bastano a dimostrarlo, perché la normalizzazione toglie gli accenti; il greco
// resta di due byte a lettera.
func TestPrefixLengthCountsCharacters(t *testing.T) {
	docs := map[string]map[string]interface{}{"1": {"t": "αβγδ"}}
	s := DefaultSettings()
	s.PrefixLength = 3
	if !cercaIn(t, s, docs, "αβγε")["1"] {
		t.Fatal("con prefix_length 3 \"αβγε\" deve trovare \"αβγδ\": i primi tre caratteri sono uguali")
	}
	s.PrefixLength = 4
	if cercaIn(t, s, docs, "αβγε")["1"] {
		t.Fatal("con prefix_length 4 il refuso sta dentro il prefisso esatto: contati in byte, sarebbero solo due lettere")
	}
}

// best_fields con operator and: tutte le parole nello stesso campo. Koskidex
// le accetta sparse fra i campi del documento.
func TestAllTermsInOneFieldRequiresOneFieldWithEveryTerm(t *testing.T) {
	docs := map[string]map[string]interface{}{
		"sparse":   {"name": "delibera", "summary": "giunta comunale"},
		"insieme":  {"name": "delibera della giunta"},
		"refuso":   {"summary": "delibra giunta"},
		"una_sola": {"name": "delibera"},
	}
	s := DefaultSettings()
	s.SearchableFields = []string{"name", "summary"}

	oggi := cercaIn(t, s, docs, "delibera giunta")
	if !oggi["sparse"] || !oggi["insieme"] || !oggi["refuso"] || oggi["una_sola"] {
		t.Fatalf("per default le parole possono stare in campi diversi: %v", oggi)
	}

	s.AllTermsInOneField = true
	got := cercaIn(t, s, docs, "delibera giunta")
	if got["sparse"] || !got["insieme"] || !got["refuso"] || got["una_sola"] || len(got) != 2 {
		t.Fatalf("con all_terms_in_one_field restano solo i documenti con un campo che ha tutte le parole, refusi compresi: %v", got)
	}

	// Con il recupero disgiuntivo basta una parola: l'impostazione non ha niente da vincolare.
	s.RetrievalMode = RetrievalAny
	if got := cercaIn(t, s, docs, "delibera giunta"); len(got) != 4 {
		t.Fatalf("con retrieval any l'impostazione non deve togliere niente: %v", got)
	}
}
