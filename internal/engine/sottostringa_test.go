package engine

import "testing"

// Le cartelle di Documentale, cercate per percorso. Su Elasticsearch la ricerca
// è un match fuzzy in OR unito a un wildcard *termine* sui termini indicizzati.
func indiceCartelle(sottostringa bool) (*InvertedIndex, Settings) {
	idx := NewInvertedIndex()
	s := DefaultSettings()
	s.SearchableFields = []string{"path"}
	s.RetrievalMode = RetrievalAny
	s.SubstringMatch = sottostringa
	idx.AddDocument("1", map[string]interface{}{"path": "Amministrazione/Delibere 2024"}, s)
	idx.AddDocument("2", map[string]interface{}{"path": "Personale/Ferie"}, s)
	idx.AddDocument("3", map[string]interface{}{"path": "Tributi/Università"}, s)
	return idx, s
}

// La scansione costa quanto il vocabolario: la misura della tesi lo riporta.
func TestVocabularySizeCountsDistinctTerms(t *testing.T) {
	idx, _ := indiceCartelle(false)
	// amministrazione, delibere, 2024, personale, ferie, tributi, universita
	if n := idx.VocabularySize(); n != 7 {
		t.Fatalf("attesi 7 termini distinti, sono %d", n)
	}
}

func trovati(idx *InvertedIndex, q string, s Settings) map[string]float64 {
	r, _ := idx.SearchScored(q, s, "AUTO", nil)
	out := map[string]float64{}
	for _, m := range r {
		out[m.DocID] = m.Score
	}
	return out
}

// Il default resta il comportamento di oggi: il baseline del 23/09 deve
// restare misurabile.
func TestSubstringMatchIsOffByDefault(t *testing.T) {
	if DefaultSettings().SubstringMatch {
		t.Fatal("la ricerca per sottostringa deve essere spenta per default")
	}
	idx, s := indiceCartelle(false)
	if got := trovati(idx, "liber", s); len(got) != 0 {
		t.Fatalf("senza sottostringa \"liber\" non deve trovare \"delibere\": %v", got)
	}
}

func TestSubstringMatchFindsATermContainingTheQuery(t *testing.T) {
	idx, s := indiceCartelle(true)
	for _, q := range []string{"liber", "LIBER", " liber ", "versità", "versita", "02"} {
		got := trovati(idx, q, s)
		atteso := "1"
		if q == "versità" || q == "versita" {
			atteso = "3"
		}
		if len(got) != 1 || got[atteso] <= 0 {
			t.Fatalf("%q deve trovare solo la cartella %s, con un punteggio positivo: %v", q, atteso, got)
		}
	}
}

// Il wildcard di Elasticsearch confronta la query intera con un termine alla
// volta, e un termine non contiene mai spazi né punteggiatura: una query con
// uno spazio o una barra non trova niente per questa via.
func TestSubstringMatchNeedsTheWholeQueryInsideOneTerm(t *testing.T) {
	spento, s0 := indiceCartelle(false)
	acceso, s1 := indiceCartelle(true)
	for _, q := range []string{"ibere 2024", "zione/ibere", "ibere"} {
		prima, dopo := trovati(spento, q, s0), trovati(acceso, q, s1)
		cambiato := len(prima) != len(dopo)
		for id, p := range prima {
			if dopo[id] != p {
				cambiato = true
			}
		}
		if q == "ibere" {
			// Controllo del controllo: da sola, la stessa parola la sottostringa la trova.
			if !cambiato {
				t.Fatalf("%q da sola è sottostringa di \"delibere\" e deve cambiare i risultati: %v", q, dopo)
			}
			continue
		}
		if cambiato {
			t.Fatalf("%q non sta dentro un solo termine: la sottostringa non deve cambiare niente, era %v ed è %v", q, prima, dopo)
		}
	}
}

// È un'unione, come il bool should: il match normale continua a trovare quello
// che trovava, e un documento trovato per entrambe le vie somma i contributi.
func TestSubstringMatchAddsToTheNormalMatch(t *testing.T) {
	spento, s0 := indiceCartelle(false)
	acceso, s1 := indiceCartelle(true)

	prima, dopo := trovati(spento, "ferie", s0), trovati(acceso, "ferie", s1)
	if len(prima) != 1 || len(dopo) != 1 || dopo["2"] <= prima["2"] {
		t.Fatalf("\"ferie\" è anche sottostringa di \"ferie\": il punteggio deve crescere, era %v ed è %v", prima, dopo)
	}
}
