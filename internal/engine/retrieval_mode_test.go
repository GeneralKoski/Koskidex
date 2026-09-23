package engine

import "testing"

func indiceDueTermini() (*InvertedIndex, Settings) {
	idx := NewInvertedIndex()
	s := DefaultSettings()
	s.SearchableFields = []string{"testo"}
	s.TypoTolerance.Enabled = false

	idx.AddDocument("entrambi", map[string]interface{}{"testo": "contratto manutenzione"}, s)
	idx.AddDocument("solo_uno", map[string]interface{}{"testo": "contratto annuale"}, s)
	idx.AddDocument("solo_due", map[string]interface{}{"testo": "manutenzione impianto"}, s)
	idx.AddDocument("nessuno", map[string]interface{}{"testo": "delibera comunale"}, s)

	return idx, s
}

func TestRetrievalAllIsTheDefaultAndRequiresEveryTerm(t *testing.T) {
	idx, s := indiceDueTermini()

	if s.RetrievalMode != RetrievalAll {
		t.Fatalf("il default deve essere %q, e' %q", RetrievalAll, s.RetrievalMode)
	}

	got, _ := idx.Search("contratto manutenzione", s, "0", nil)
	if len(got) != 1 || got[0] != "entrambi" {
		t.Fatalf("atteso solo il documento con entrambi i termini, ottenuto %v", got)
	}
}

// Settings are persisted to disk. An index saved before RetrievalMode existed
// unmarshals it as "", and must keep behaving exactly as it did, otherwise an
// upgrade silently changes everybody's results.
func TestEmptyRetrievalModeBehavesAsAll(t *testing.T) {
	idx, s := indiceDueTermini()
	s.RetrievalMode = ""

	got, _ := idx.Search("contratto manutenzione", s, "0", nil)
	if len(got) != 1 || got[0] != "entrambi" {
		t.Fatalf("una modalita' vuota deve restare congiuntiva, ottenuto %v", got)
	}
}

func TestRetrievalAnyReturnsDocumentsMatchingOneTerm(t *testing.T) {
	idx, s := indiceDueTermini()
	s.RetrievalMode = RetrievalAny

	got, _ := idx.Search("contratto manutenzione", s, "0", nil)
	if len(got) != 3 {
		t.Fatalf("attesi i 3 documenti con almeno un termine, ottenuto %v", got)
	}
	for _, id := range got {
		if id == "nessuno" {
			t.Fatalf("un documento senza nessuno dei termini non deve comparire: %v", got)
		}
	}
}

// Disjunctive retrieval without ranking would be useless: the document
// carrying both terms has to come first.
func TestRetrievalAnyRanksMoreMatchedTermsHigher(t *testing.T) {
	idx, s := indiceDueTermini()
	s.RetrievalMode = RetrievalAny

	scored, _ := idx.SearchScored("contratto manutenzione", s, "0", nil)
	if len(scored) != 3 {
		t.Fatalf("attesi 3 risultati, ottenuti %d", len(scored))
	}
	if scored[0].DocID != "entrambi" {
		t.Fatalf("in testa deve esserci il documento con entrambi i termini, c'e' %q", scored[0].DocID)
	}
	if scored[0].WordsMatched != 2 {
		t.Fatalf("il primo deve aver trovato 2 termini, ne ha %d", scored[0].WordsMatched)
	}
	if !(scored[0].Score > scored[1].Score) {
		t.Fatalf("chi trova piu' termini deve valere di piu': %v contro %v", scored[0].Score, scored[1].Score)
	}
}

// A single-term query cannot tell the two modes apart, which is exactly why
// NFCorpus suffers less than SciFact.
func TestRetrievalModesAgreeOnSingleTermQueries(t *testing.T) {
	idx, s := indiceDueTermini()

	conAll, _ := idx.Search("contratto", s, "0", nil)
	s.RetrievalMode = RetrievalAny
	conAny, _ := idx.Search("contratto", s, "0", nil)

	if len(conAll) != len(conAny) {
		t.Fatalf("su un termine solo le due modalita' devono coincidere: %v contro %v", conAll, conAny)
	}
}
