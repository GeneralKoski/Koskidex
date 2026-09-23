package engine

import "testing"

func indiceAnalizzato(stemmer string, stop map[string]bool) (*InvertedIndex, Settings) {
	s := DefaultSettings()
	s.SearchableFields = []string{"testo"}
	s.TypoTolerance.Enabled = false
	s.RetrievalMode = RetrievalAny
	s.Stemmer = stemmer
	s.StopWords = stop

	idx := NewInvertedIndex()
	idx.AddDocument("a", map[string]interface{}{"testo": "the maintenance of industrial plants"}, s)
	idx.AddDocument("b", map[string]interface{}{"testo": "maintaining a plant is expensive"}, s)
	idx.AddDocument("c", map[string]interface{}{"testo": "connection established"}, s)

	return idx, s
}

// Il punto di tutto: una forma flessa nella query deve incontrare una forma
// diversa nel documento.
func TestStemmerMakesInflectedFormsMeet(t *testing.T) {
	idx, s := indiceAnalizzato(StemmerNone, nil)
	if got, _ := idx.Search("plants", s, "0", nil); len(got) != 1 {
		t.Fatalf("senza stemmer \"plants\" deve trovare solo il documento che lo scrive cosi', ottenuto %v", got)
	}

	idx, s = indiceAnalizzato(StemmerPorter, nil)
	got, _ := idx.Search("plants", s, "0", nil)
	if len(got) != 2 {
		t.Fatalf("con lo stemmer \"plants\" deve trovare anche \"plant\", ottenuto %v", got)
	}
}

// Il pezzo che si rompe piu' facilmente: se lo stemmer arrivasse solo
// all'indicizzazione e non alla query, l'indice conterrebbe "plant" e la query
// cercherebbe "plants". Zero risultati, nessun errore.
func TestStemmerIsAppliedToTheQueryToo(t *testing.T) {
	idx, s := indiceAnalizzato(StemmerPorter, nil)

	// "maintaining" -> "maintain"; nel documento c'e' "maintenance", che
	// Porter porta a "mainten": sono diversi, e va bene cosi'.
	// Qui basta che la query stemmata trovi il documento che la contiene.
	if got, _ := idx.Search("maintaining", s, "0", nil); len(got) != 1 || got[0] != "b" {
		t.Fatalf("la query deve essere stemmata come il documento, ottenuto %v", got)
	}
	if got, _ := idx.Search("maintain", s, "0", nil); len(got) != 1 || got[0] != "b" {
		t.Fatalf("la forma gia' ridotta deve trovare lo stesso documento, ottenuto %v", got)
	}
}

// Le settings sono persistite: un indice salvato prima che il campo esistesse
// lo rilegge vuoto e non deve cambiare comportamento.
func TestEmptyStemmerBehavesAsBefore(t *testing.T) {
	senza, s1 := indiceAnalizzato(StemmerNone, nil)
	vuoto, s2 := indiceAnalizzato("", nil)

	for _, q := range []string{"plants", "maintenance", "connection"} {
		a, _ := senza.Search(q, s1, "0", nil)
		b, _ := vuoto.Search(q, s2, "0", nil)
		if len(a) != len(b) {
			t.Fatalf("query %q: %v contro %v", q, a, b)
		}
	}
}

// Un nome sconosciuto non deve impedire l'apertura di un indice: rankera'
// diverso, e si vede, mentre un rifiuto di partire no.
func TestUnknownStemmerFallsBackToNone(t *testing.T) {
	idx, s := indiceAnalizzato("qualcosa-che-non-esiste", nil)
	got, _ := idx.Search("plants", s, "0", nil)
	if len(got) != 1 {
		t.Fatalf("atteso il comportamento senza stemmer, ottenuto %v", got)
	}
}

func TestStopWordsAreRemovedFromIndexAndQuery(t *testing.T) {
	idx, s := indiceAnalizzato(StemmerNone, EnglishStopWords())

	if got, _ := idx.Search("the", s, "0", nil); len(got) != 0 {
		t.Fatalf("una stopword non deve trovare niente, ottenuto %v", got)
	}
	if idx.DocFrequency("the") != 0 {
		t.Fatalf("\"the\" non deve stare nell'indice, df = %d", idx.DocFrequency("the"))
	}
	if got, _ := idx.Search("maintenance", s, "0", nil); len(got) != 1 {
		t.Fatalf("una parola normale deve continuare a funzionare, ottenuto %v", got)
	}
}

// L'ordine conta: Lucene toglie le stopword PRIMA di stemmare, perche' la
// lista e' fatta di parole intere. Se si stemmasse prima, "this" diventerebbe
// "thi" e non verrebbe piu' riconosciuto come stopword.
func TestStopWordsAreRemovedBeforeStemming(t *testing.T) {
	s := DefaultSettings()
	s.SearchableFields = []string{"testo"}
	s.Stemmer = StemmerPorter
	s.StopWords = EnglishStopWords()

	tokens := Tokenize("this is the plants", "testo", s)

	for _, tk := range tokens {
		if tk.Term == "thi" {
			t.Fatal("\"this\" e' stato stemmato prima di essere confrontato con le stopword")
		}
	}
	if len(tokens) != 1 || tokens[0].Term != "plant" {
		t.Fatalf("atteso il solo termine \"plant\", ottenuto %+v", tokens)
	}
}

// Le statistiche BM25 si costruiscono in indicizzazione: devono contare i
// termini analizzati, non quelli grezzi.
func TestIndexStatisticsCountAnalyzedTerms(t *testing.T) {
	idx, _ := indiceAnalizzato(StemmerPorter, nil)

	if df := idx.DocFrequency("plant"); df != 2 {
		t.Fatalf("\"plant\" sta in due documenti dopo lo stemming, df = %d", df)
	}
	if df := idx.DocFrequency("plants"); df != 0 {
		t.Fatalf("la forma non stemmata non deve esistere nell'indice, df = %d", df)
	}
}
