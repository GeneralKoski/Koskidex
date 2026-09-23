package eval

import (
	"testing"

	"github.com/GeneralKoski/Koskidex/internal/engine"
)

func documentiDiProva() []Document {
	return []Document{
		{ID: "d1", Title: "Manutenzione impianti", Text: "contratto annuale di assistenza"},
		{ID: "d2", Title: "Delibera comunale", Text: "nessuna attinenza"},
		{ID: "d3", Title: "Assistenza tecnica", Text: "interventi di manutenzione"},
	}
}

func TestKoskidexSearcherFindsAndTruncates(t *testing.T) {
	s := NewKoskidexSearcher(documentiDiProva(), nil)

	got := s.Search("manutenzione", 10)
	if len(got) != 2 {
		t.Fatalf("attesi 2 risultati, ottenuti %v", got)
	}

	if uno := s.Search("manutenzione", 1); len(uno) != 1 {
		t.Fatalf("il limite non e' rispettato: %v", uno)
	}
}

// The title has to be searchable, otherwise half of each BEIR document is
// invisible and every number is wrong for a reason nobody would look for.
func TestKoskidexSearcherIndexesTitleAndText(t *testing.T) {
	s := NewKoskidexSearcher(documentiDiProva(), nil)

	if got := s.Search("delibera", 10); len(got) != 1 || got[0] != "d2" {
		t.Fatalf("termine presente solo nel titolo non trovato: %v", got)
	}
	if got := s.Search("interventi", 10); len(got) != 1 || got[0] != "d3" {
		t.Fatalf("termine presente solo nel testo non trovato: %v", got)
	}
}

// Trap 3 of SOURCE.md: the reference runs are flat, one field, one weight.
func TestKoskidexSearcherUsesOneFieldWithOneWeight(t *testing.T) {
	s := NewKoskidexSearcher(documentiDiProva(), nil)
	st := s.Settings()

	if len(st.SearchableFields) != 1 {
		t.Fatalf("atteso un solo campo cercabile, ottenuti %v", st.SearchableFields)
	}
	for campo, peso := range st.FieldWeights {
		if peso != 1.0 {
			t.Fatalf("il campo %q ha peso %v, atteso 1.0", campo, peso)
		}
	}
}

// Trap 2 of SOURCE.md, and the one that fails most quietly: Lucene has no typo
// tolerance, Koskidex has it on by default. If it stays on, what is measured
// is not what was published, and the number looks plausible anyway.
func TestKoskidexSearcherDoesNotMatchTypos(t *testing.T) {
	s := NewKoskidexSearcher(documentiDiProva(), nil)

	if s.Settings().TypoTolerance.Enabled {
		t.Fatal("la tolleranza ai refusi deve essere spenta")
	}

	// "manutenzine" is one deletion away from "manutenzione": with tolerance on
	// it would match, and here it must not.
	if got := s.Search("manutenzine", 10); len(got) != 0 {
		t.Fatalf("un refuso ha prodotto risultati: %v", got)
	}

	// The correct spelling still has to work, otherwise the test above would
	// pass on a searcher that simply finds nothing.
	if got := s.Search("manutenzione", 10); len(got) == 0 {
		t.Fatal("la parola scritta bene non trova piu' niente")
	}
}

// The hook exists so a later phase can switch a ranking flag on without this
// file changing, and it must not undo the two settings above by accident.
func TestKoskidexSearcherAppliesTheSettingsHook(t *testing.T) {
	var visto bool
	s := NewKoskidexSearcher(documentiDiProva(), func(st *engine.Settings) {
		visto = true
		st.StopWords = map[string]bool{"di": true}
	})

	if !visto {
		t.Fatal("la funzione di modifica non e' stata chiamata")
	}
	if !s.Settings().StopWords["di"] {
		t.Fatal("la modifica non e' arrivata alle settings usate")
	}
	if s.Settings().TypoTolerance.Enabled {
		t.Fatal("la modifica non deve poter riaccendere i refusi per sbaglio")
	}
}
