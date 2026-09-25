package engine

import (
	"reflect"
	"strings"
	"testing"
)

// The oracle is the vocabulary Lucene tests its own ItalianLightStemmer with:
// 35494 words with accents, and the stem each one must produce.
func TestItalianLightMatchesLucenesVocabulary(t *testing.T) {
	righe := leggiRighe(t, "testdata/itlight.txt")
	if len(righe) != 35494 {
		t.Fatalf("attese 35494 righe, trovate %d: file troncato?", len(righe))
	}

	sbagliate, cambiate := 0, 0
	for _, riga := range righe {
		parola, atteso, ok := strings.Cut(riga, "\t")
		if !ok {
			t.Fatalf("riga senza tabulazione: %q", riga)
		}
		if parola != atteso {
			cambiate++
		}
		if got := stemItalianLight(parola); got != atteso {
			sbagliate++
			if sbagliate <= 10 {
				t.Errorf("%q: atteso %q, ottenuto %q", parola, atteso, got)
			}
		}
	}
	if sbagliate > 0 {
		t.Fatalf("%d parole su %d non corrispondono", sbagliate, len(righe))
	}
	if cambiate < 20000 {
		t.Fatalf("solo %d parole cambiano: il vocabolario non mette alla prova lo stemmer", cambiate)
	}
}

// Koskidex folds accents before the stemmer sees a term, so the stemmer is
// always handed the folded word. The stem must be the folded Lucene stem all
// the same, or an accented document and its unaccented query would part.
func TestItalianLightAfterFoldingGivesTheFoldedStem(t *testing.T) {
	for _, riga := range leggiRighe(t, "testdata/itlight.txt") {
		parola, atteso, _ := strings.Cut(riga, "\t")
		if got, want := stemItalianLight(removeAccents(parola)), removeAccents(atteso); got != want {
			t.Fatalf("%q piegata: atteso %q, ottenuto %q", parola, want, got)
		}
	}
}

func TestItalianStopWordsAreSnowballsFolded(t *testing.T) {
	parole := ItalianStopWords()
	// 279 nella lista di Snowball; "è" ed "e" diventano una.
	if len(parole) != 278 {
		t.Fatalf("attese 278 stopword, trovate %d", len(parole))
	}
	for _, p := range []string{"e", "perche", "piu", "sara", "della", "dell", "gli", "stessero"} {
		if !parole[p] {
			t.Errorf("%q dovrebbe essere una stopword", p)
		}
	}
	for p := range parole {
		if p != removeAccents(p) {
			t.Errorf("%q ha un accento: il tokenizer non lo vedrebbe mai", p)
		}
	}
	parole["comune"] = true
	if ItalianStopWords()["comune"] {
		t.Fatal("la mappa va restituita nuova ogni volta")
	}
}

func TestElisionStripsOnlyTheArticles(t *testing.T) {
	articoli := ItalianElisionArticles()
	casi := map[string]string{
		"dell'illuminazione": "illuminazione",
		"l’affidamento":      "affidamento",
		"un'area":            "area",
		"all'albo":           "albo",
		"quest'anno":         "quest'anno",
		"sant'angelo":        "sant'angelo",
		"l‘affidamento":      "l‘affidamento",
		"dell'l'altro":       "l'altro",
		"comune":             "comune",
	}
	for in, want := range casi {
		if got := togliElisione(in, articoli); got != want {
			t.Errorf("%q: atteso %q, ottenuto %q", in, want, got)
		}
	}
}

func TestItalianAnalysisEndToEnd(t *testing.T) {
	s := DefaultSettings()
	s.Tokenizer = TokenizerStandard
	s.ElisionArticles = ItalianElisionArticles()
	s.StopWords = ItalianStopWords()
	s.Stemmer = StemmerItalianLight

	var got []string
	for _, tok := range Tokenize("Affidamento dell'illuminazione pubblica e L’AFFIDAMENTO delle strade comunali", "t", s) {
		got = append(got, tok.Term)
	}
	want := []string{"affidament", "illuminazion", "pubblic", "affidament", "strad", "comunal"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("atteso %v, ottenuto %v", want, got)
	}
}

// Without the settings nothing changes: an index saved before these fields
// existed reads them as empty.
func TestItalianAnalysisIsOffByDefault(t *testing.T) {
	s := DefaultSettings()
	s.Tokenizer = TokenizerStandard
	var got []string
	for _, tok := range Tokenize("dell'illuminazione pubblica", "t", s) {
		got = append(got, tok.Term)
	}
	if want := []string{"dell'illuminazione", "pubblica"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("atteso %v, ottenuto %v", want, got)
	}
}

// The rarer vowels in Lucene's map never reach the end of a word in the
// vocabulary, so they are pinned here.
func TestItalianLightFoldsTheRareVowels(t *testing.T) {
	casi := map[string]string{"citroën": "citroen", "naïveté": "naivet", "façade": "façad", "perché": "perc", "città": "città"}
	for in, want := range casi {
		if got := stemItalianLight(in); got != want {
			t.Errorf("%q: atteso %q, ottenuto %q", in, want, got)
		}
	}
}
