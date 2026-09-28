package engine

import (
	"sort"
	"testing"
)

func TestTyposOnAmountsAreOnByDefault(t *testing.T) {
	if DefaultSettings().TypoTolerance.DisableOnAmounts {
		t.Fatal("i refusi sugli importi devono restare accesi per default: il baseline deve restare misurabile")
	}
}

func TestMaxTyposDeniesTyposToAnAmount(t *testing.T) {
	importi := refusiDocumentale(true)
	importi.DisableOnAmounts = true
	casi := []struct {
		termine string
		oggi    int
		senza   int
	}{
		{"5056,03", 2, 0},
		{"1.234,56", 2, 0},
		{"14.01.2026", 2, 0},
		{"30,5", 1, 0},
		// Le sole cifre sono di disable_on_numbers, che qui resta spento.
		{"5056", 1, 1},
		// Non è un importo: non comincia o non finisce con una cifra, o ha lettere.
		{",503", 1, 1},
		{"503,", 1, 1},
		{"a5,03", 1, 1},
		{"crispiano", 2, 2},
	}
	for _, c := range casi {
		if got := MaxTypos(c.termine, refusiDocumentale(true), "AUTO"); got != c.oggi {
			t.Errorf("%q, refusi sugli importi: %d refusi, attesi %d", c.termine, got, c.oggi)
		}
		if got := MaxTypos(c.termine, importi, "AUTO"); got != c.senza {
			t.Errorf("%q, senza refusi sugli importi: %d refusi, attesi %d", c.termine, got, c.senza)
		}
	}
	if got := MaxTypos("5056,03", importi, "1"); got != 1 {
		t.Errorf("la fuzziness esplicita vince sull'impostazione: %d refusi, atteso 1", got)
	}
}

func cercaImporti(senzaRefusi bool, q string) []string {
	idx := NewInvertedIndex()
	s := DefaultSettings()
	s.SearchableFields = []string{"t"}
	s.Tokenizer = TokenizerStandard
	s.TypoTolerance = refusiDocumentale(false)
	s.TypoTolerance.DisableOnAmounts = senzaRefusi
	s.DisablePrefixSearch = true
	for id, testo := range map[string]string{
		"giusto": "importo di euro 5.056,03 crispiano", "vicino": "importo di euro 5.056,08 crispiano",
		"parola": "delibera crispiano",
	} {
		idx.AddDocument(id, map[string]interface{}{"t": testo}, s)
	}
	ids, _ := idx.Search(q, s, "AUTO", nil)
	sort.Strings(ids)
	return ids
}

func TestDisableOnAmountsDropsDocumentsFoundOnlyByATypoOnAnAmount(t *testing.T) {
	if got := cercaImporti(false, "5.056,03 crispiano"); len(got) != 2 {
		t.Fatalf("con i refusi sugli importi 5.056,03 deve trovare anche 5.056,08: %v", got)
	}
	if got := cercaImporti(true, "5.056,03 crispiano"); len(got) != 1 || got[0] != "giusto" {
		t.Fatalf("senza refusi sugli importi resta solo l'atto con l'importo esatto: %v", got)
	}
	if got := cercaImporti(true, "delbera crispiano"); len(got) != 1 || got[0] != "parola" {
		t.Fatalf("il refuso sulla parola deve restare ammesso: %v", got)
	}
}
