package eval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func scriviTemporaneo(t *testing.T, contenuto string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "qrels.tsv")
	if err := os.WriteFile(path, []byte(contenuto), 0o600); err != nil {
		t.Fatal(err)
	}

	return path
}

func TestPoolIsTheUnionOfTheRunsCutAtDepth(t *testing.T) {
	a := []string{"d1", "d2", "d3", "d4"}
	b := []string{"d3", "d5", "d1", "d9"}

	got := PoolForQuery([][]string{a, b}, 2)

	// Solo i primi 2 di ciascuna: d1, d2 da a; d3, d5 da b.
	atteso := []string{"d1", "d2", "d3", "d5"}
	if len(got) != len(atteso) {
		t.Fatalf("atteso %v, ottenuto %v", atteso, got)
	}
	for i := range atteso {
		if got[i] != atteso[i] {
			t.Fatalf("atteso %v, ottenuto %v", atteso, got)
		}
	}
}

// Se il pool arrivasse in ordine di ranking, chi annota leggerebbe la
// posizione come un suggerimento e finirebbe per confermare il sistema che
// l'ha prodotta. I giudizi concorderebbero con quel sistema per costruzione.
func TestPoolHidesWhichRunFoundWhat(t *testing.T) {
	primo := []string{"d9", "d1"}
	secondo := []string{"d5", "d3"}

	got := PoolForQuery([][]string{primo, secondo}, 10)

	for i := 1; i < len(got); i++ {
		if got[i-1] > got[i] {
			t.Fatalf("il pool deve essere ordinato per id, non per ranking: %v", got)
		}
	}
	if got[0] != "d1" {
		t.Fatalf("in testa deve esserci l'id piu' basso, non il primo risultato di un run: %v", got)
	}
}

func TestPoolDeduplicates(t *testing.T) {
	got := PoolForQuery([][]string{{"d1", "d2"}, {"d1", "d2"}, {"d2"}}, 10)
	if len(got) != 2 {
		t.Fatalf("attesi 2 documenti distinti, ottenuti %v", got)
	}
}

func TestPoolOfNothingIsEmpty(t *testing.T) {
	if got := PoolForQuery(nil, 10); len(got) != 0 {
		t.Fatalf("atteso vuoto, ottenuto %v", got)
	}
	if got := PoolForQuery([][]string{{}, {}}, 10); len(got) != 0 {
		t.Fatalf("atteso vuoto, ottenuto %v", got)
	}
}

func TestSheetRoundTrips(t *testing.T) {
	righe := []SheetRow{
		{QueryID: "q-0001", Query: "manutenzione impianto", DocID: "doc-0001", Title: "Fattura 2026/0173", Snippet: "riga uno\nriga due\tcon tab", Grade: ""},
		{QueryID: "q-0001", Query: "manutenzione impianto", DocID: "doc-0002", Title: "Contratto", Snippet: "altro", Grade: "2"},
	}

	var buf strings.Builder
	if err := WriteSheet(&buf, righe); err != nil {
		t.Fatal(err)
	}

	riletto, err := ReadSheet(strings.NewReader(buf.String()))
	if err != nil {
		t.Fatal(err)
	}
	if len(riletto) != 2 {
		t.Fatalf("attese 2 righe, ottenute %d", len(riletto))
	}
	if riletto[0].DocID != "doc-0001" || riletto[1].Grade != "2" {
		t.Fatalf("rilettura sbagliata: %+v", riletto)
	}
	// A capo e tabulazioni dentro un campo spaccherebbero il TSV.
	if strings.ContainsAny(riletto[0].Snippet, "\n\t") {
		t.Fatalf("l'estratto deve stare su una riga sola: %q", riletto[0].Snippet)
	}
}

func TestReadSheetRejectsSomethingThatIsNotASheet(t *testing.T) {
	if _, err := ReadSheet(strings.NewReader("")); err == nil {
		t.Fatal("un file vuoto deve dare errore")
	}
	if _, err := ReadSheet(strings.NewReader("a\tb\tc\td\te\tf\n")); err == nil {
		t.Fatal("un'intestazione sbagliata deve dare errore")
	}
}

func TestSheetToQrels(t *testing.T) {
	q, err := SheetToQrels([]SheetRow{
		{QueryID: "q-0001", DocID: "doc-0001", Grade: "2"},
		{QueryID: "q-0001", DocID: "doc-0002", Grade: "0"},
		{QueryID: "q-0002", DocID: "doc-0001", Grade: " 1 "},
	})
	if err != nil {
		t.Fatal(err)
	}
	if q["q-0001"]["doc-0001"] != 2 || q["q-0001"]["doc-0002"] != 0 || q["q-0002"]["doc-0001"] != 1 {
		t.Fatalf("giudizi sbagliati: %+v", q)
	}
}

// "Non l'ho ancora guardato" e "l'ho guardato e non c'entra" non sono la
// stessa cosa, e leggere il primo come il secondo gonfia ogni numero che esce
// dal file.
func TestSheetToQrelsRefusesAnEmptyGrade(t *testing.T) {
	_, err := SheetToQrels([]SheetRow{
		{QueryID: "q-0001", DocID: "doc-0001", Grade: "1"},
		{QueryID: "q-0001", DocID: "doc-0002", Grade: ""},
	})
	if err == nil {
		t.Fatal("un grado vuoto deve dare errore, non valere zero")
	}
	if !strings.Contains(err.Error(), "doc-0002") {
		t.Fatalf("l'errore deve dire quale riga manca: %v", err)
	}
}

func TestSheetToQrelsRefusesAnInvalidGrade(t *testing.T) {
	for _, g := range []string{"3", "-1", "si", "1.5"} {
		if _, err := SheetToQrels([]SheetRow{{QueryID: "q", DocID: "d", Grade: g}}); err == nil {
			t.Fatalf("il grado %q doveva essere rifiutato", g)
		}
	}
}

// Il file deve essere letto dallo stesso LoadQrels che legge SciFact e
// NFCorpus: se il formato divergesse, la collezione di Documentale avrebbe
// bisogno di un lettore suo e non sarebbe piu' confrontabile.
func TestWrittenQrelsAreReadBackByTheBeirLoader(t *testing.T) {
	originali := Qrels{
		"q-0002": {"doc-0005": 1},
		"q-0001": {"doc-0003": 2, "doc-0001": 0},
	}

	var buf strings.Builder
	if err := WriteQrels(&buf, originali); err != nil {
		t.Fatal(err)
	}

	path := scriviTemporaneo(t, buf.String())
	riletto, err := LoadQrels(path)
	if err != nil {
		t.Fatalf("il file prodotto non si rilegge come qrels BEIR: %v", err)
	}
	if riletto["q-0001"]["doc-0003"] != 2 || riletto["q-0002"]["doc-0005"] != 1 {
		t.Fatalf("giudizi persi nel giro: %+v", riletto)
	}
	if _, presente := riletto["q-0001"]["doc-0001"]; !presente {
		t.Fatal("un giudizio a zero e' un giudizio e non va perso")
	}
}

func TestWriteQrelsIsDeterministic(t *testing.T) {
	q := Qrels{"q-0002": {"doc-0005": 1, "doc-0001": 0}, "q-0001": {"doc-0009": 2, "doc-0003": 1}}

	var primo strings.Builder
	if err := WriteQrels(&primo, q); err != nil {
		t.Fatal(err)
	}
	for giro := 0; giro < 5; giro++ {
		var altro strings.Builder
		if err := WriteQrels(&altro, q); err != nil {
			t.Fatal(err)
		}
		if altro.String() != primo.String() {
			t.Fatalf("giro %d diverso:\n%s\ncontro\n%s", giro, altro.String(), primo.String())
		}
	}
}
