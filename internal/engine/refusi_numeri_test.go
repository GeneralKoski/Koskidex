package engine

import (
	"sort"
	"testing"
)

// Le soglie di Documentale, che copiano la fuzziness AUTO di Elasticsearch: un
// refuso da 3 caratteri, due da 6.
func refusiDocumentale(suiNumeri bool) TypoSettings {
	return TypoSettings{Enabled: true, MinWordLengthOneTypo: 3, MinWordLengthTwoTypos: 6, DisableOnNumbers: !suiNumeri}
}

func TestTyposOnNumbersAreOnByDefault(t *testing.T) {
	if DefaultSettings().TypoTolerance.DisableOnNumbers {
		t.Fatal("i refusi sui numeri devono restare accesi per default: il baseline deve restare misurabile")
	}
}

func TestMaxTyposDeniesTyposToATermMadeOnlyOfDigits(t *testing.T) {
	casi := []struct {
		termine   string
		fuzziness string
		oggi      int
		senza     int
	}{
		{"1209", "", 1, 0},
		{"1209", "AUTO", 1, 0},
		{"0078710", "AUTO", 2, 0},
		// Un codice misto non è un numero: tiene i refusi, come in Meilisearch.
		{"a651", "AUTO", 1, 1},
		{"crispiano", "AUTO", 2, 2},
		// La fuzziness esplicita vince sull'impostazione, come vince su Enabled.
		{"1209", "1", 1, 1},
		{"1209", "2", 2, 2},
	}
	for _, c := range casi {
		if got := MaxTypos(c.termine, refusiDocumentale(true), c.fuzziness); got != c.oggi {
			t.Errorf("%q fuzziness %q, refusi sui numeri: %d refusi, attesi %d", c.termine, c.fuzziness, got, c.oggi)
		}
		if got := MaxTypos(c.termine, refusiDocumentale(false), c.fuzziness); got != c.senza {
			t.Errorf("%q fuzziness %q, senza refusi sui numeri: %d refusi, attesi %d", c.termine, c.fuzziness, got, c.senza)
		}
	}
}

func cercaNumeri(t *testing.T, suiNumeri bool, q string) []string {
	idx := NewInvertedIndex()
	s := DefaultSettings()
	s.SearchableFields = []string{"t"}
	s.TypoTolerance = refusiDocumentale(suiNumeri)
	s.DisablePrefixSearch = true
	for id, testo := range map[string]string{
		"giusto": "determina 1209 crispiano", "vicino": "determina 1109 crispiano",
		"codice": "protocollo a652 gorizia", "altro": "avviso 77 lecce",
	} {
		idx.AddDocument(id, map[string]interface{}{"t": testo}, s)
	}
	ids, _ := idx.Search(q, s, "AUTO", nil)
	sort.Strings(ids)
	return ids
}

func TestDisableOnNumbersDropsDocumentsFoundOnlyByATypoOnANumber(t *testing.T) {
	if got := cercaNumeri(t, true, "1209 crispiano"); len(got) != 2 {
		t.Fatalf("con i refusi sui numeri 1209 deve trovare anche 1109: %v", got)
	}
	got := cercaNumeri(t, false, "1209 crispiano")
	if len(got) != 1 || got[0] != "giusto" {
		t.Fatalf("senza refusi sui numeri resta solo l'atto col numero esatto: %v", got)
	}
}

func TestDisableOnNumbersKeepsTyposOnWordsAndMixedCodes(t *testing.T) {
	if got := cercaNumeri(t, false, "1209 crispaino"); len(got) != 1 || got[0] != "giusto" {
		t.Fatalf("il refuso sulla parola deve restare ammesso: %v", got)
	}
	if got := cercaNumeri(t, false, "a651 gorizia"); len(got) != 1 || got[0] != "codice" {
		t.Fatalf("il codice misto deve restare con i refusi: %v", got)
	}
}
