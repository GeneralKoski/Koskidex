package engine

import (
	"reflect"
	"testing"
)

func termini(testo string, s Settings) []string {
	var out []string
	for _, t := range Tokenize(testo, "", s) {
		out = append(out, t.Term)
	}
	return out
}

// Il tokenizer di oggi spezza su tutto ciò che non è lettera o cifra.
func TestDefaultTokenizerSplitsOnEveryPunctuation(t *testing.T) {
	if DefaultSettings().Tokenizer != "" {
		t.Fatal("il tokenizer predefinito deve restare quello di oggi")
	}
	if got := termini("dell'illuminazione 14.01.2026", DefaultSettings()); !reflect.DeepEqual(got, []string{"dell", "illuminazione", "14", "01", "2026"}) {
		t.Fatalf("per default si spezza sulla punteggiatura: %v", got)
	}
}

// Le attese sono le risposte di _analyze con l'analizzatore standard di
// Elasticsearch 9.1 (regole UAX#29), lette il 25/09/2026.
func TestStandardTokenizerKeepsWordInternalPunctuationAsElasticsearch(t *testing.T) {
	s := DefaultSettings()
	s.Tokenizer = TokenizerStandard
	casi := map[string][]string{
		"dell'illuminazione": {"dell'illuminazione"},
		"all’Associazione":   {"all’associazione"},
		"l‘atto":             {"l‘atto"},
		"l'atto n.187/2026":  {"l'atto", "n", "187", "2026"},
		"14.01.2026":         {"14.01.2026"},
		"N.187":              {"n", "187"},
		"3,5":                {"3,5"},
		"1;2":                {"1;2"},
		"1'000":              {"1'000"},
		"e-mail":             {"e", "mail"},
		"d.lgs. 50/2016":     {"d.lgs", "50", "2016"},
		"l' atto":            {"l", "atto"},
		"a:b":                {"a:b"},
		"1:2":                {"1", "2"},
		"a;b":                {"a", "b"},
		"a_b":                {"a_b"},
		"1_2":                {"1_2"},
		"_ab":                {"_ab"},
		"ab_":                {"ab_"},
		"x.y.z":              {"x.y.z"},
		"3.a":                {"3", "a"},
		"a.3":                {"a", "3"},
		"a..b":               {"a", "b"},
		"1,,2":               {"1", "2"},
		"a·b":                {"a·b"},
		"rock’n’roll":        {"rock’n’roll"},
		"d.lgs.":             {"d.lgs"},
		"'atto":              {"atto"},
		"187,":               {"187"},
		":ab":                {"ab"},
	}
	for testo, atteso := range casi {
		if got := termini(testo, s); !reflect.DeepEqual(got, atteso) {
			t.Errorf("%q: attesi %v, avuti %v", testo, atteso, got)
		}
	}
}

// In Elasticsearch "illuminazione" non trova "dell'illuminazione": con il
// tokenizer standard Koskidex fa lo stesso, e la query elisa si trova.
func TestStandardTokenizerAppliesToQueriesToo(t *testing.T) {
	s := DefaultSettings()
	s.Tokenizer = TokenizerStandard
	s.TypoTolerance.Enabled = false
	idx := NewInvertedIndex()
	idx.AddDocument("1", map[string]interface{}{"t": "sostituzione dei pali dell'illuminazione pubblica"}, s)
	if got, _ := idx.Search("illuminazione", s, "0", nil); len(got) != 0 {
		t.Fatalf("come in Elasticsearch, \"illuminazione\" non deve trovare \"dell'illuminazione\": %v", got)
	}
	if got, _ := idx.Search("dell'illuminazione", s, "0", nil); len(got) != 1 {
		t.Fatalf("la parola elisa intera deve trovarsi: %v", got)
	}
}
