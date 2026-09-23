package engine

import (
	"bufio"
	"os"
	"testing"
)

// The oracle is Porter's own test vocabulary, downloaded from tartarus.org:
// 23531 words and the stem each one must produce. Cases picked by hand would
// only test the cases I already thought of; this tests the ones I did not.
func TestPorterMatchesTheOfficialVocabulary(t *testing.T) {
	parole := leggiRighe(t, "testdata/voc.txt")
	attesi := leggiRighe(t, "testdata/output.txt")

	if len(parole) != len(attesi) {
		t.Fatalf("i due file non hanno lo stesso numero di righe: %d e %d", len(parole), len(attesi))
	}
	if len(parole) < 20000 {
		t.Fatalf("il vocabolario di prova ha solo %d parole, ne attendevo oltre 20000: file troncato?", len(parole))
	}

	sbagliate := 0
	for i, parola := range parole {
		got := stemPorter(parola)
		if got != attesi[i] {
			sbagliate++
			if sbagliate <= 10 {
				t.Errorf("%q: atteso %q, ottenuto %q", parola, attesi[i], got)
			}
		}
	}
	if sbagliate > 0 {
		t.Fatalf("%d parole su %d non corrispondono", sbagliate, len(parole))
	}
}

// Porter NON e' idempotente, e questo test e' qui per impedire che qualcuno lo
// dia per scontato.
//
// "abase" diventa "abas", e "abas" ripassato diventa "aba", perche' la seconda
// volta la s finale sembra un plurale. Conseguenza pratica, ed e' il motivo per
// cui vale la pena scriverlo: l'analizzatore va applicato **esattamente una
// volta** su ogni percorso. Se il testo di una query passasse due volte dal
// tokenizer, produrrebbe termini che nell'indice non esistono, e il sintomo
// sarebbe "non trova niente" senza nessun errore.
func TestPorterIsNotIdempotent(t *testing.T) {
	if una, due := stemPorter("abase"), stemPorter(stemPorter("abase")); una == due {
		t.Fatalf("attese due radici diverse, ottenuto %q entrambe le volte: se Porter fosse diventato idempotente, questo commento va riscritto", una)
	}

	quante := 0
	for _, parola := range leggiRighe(t, "testdata/voc.txt") {
		if una := stemPorter(parola); stemPorter(una) != una {
			quante++
		}
	}
	if quante == 0 {
		t.Fatal("nessuna parola cambia alla seconda passata: inatteso")
	}
	t.Logf("%d parole su 23531 cambiano se stemmate due volte", quante)
}

// Parole di una o due lettere non hanno niente da togliere.
func TestPorterLeavesVeryShortWordsAlone(t *testing.T) {
	for _, parola := range []string{"", "a", "is", "be", "of"} {
		if got := stemPorter(parola); got != parola {
			t.Fatalf("%q doveva restare uguale, e' diventato %q", parola, got)
		}
	}
}

func leggiRighe(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		out = append(out, sc.Text())
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}

	return out
}
