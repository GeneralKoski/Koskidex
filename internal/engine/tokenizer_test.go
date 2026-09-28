package engine

import (
	"reflect"
	"testing"
)

func TestTokenize(t *testing.T) {
	stopwords := map[string]bool{
		"the": true,
		"a":   true,
		"is":  true,
		"in":  true,
	}

	tests := []struct {
		name      string
		text      string
		field     string
		stopwords map[string]bool
		expected  []Token
	}{
		{
			name:      "basic lowercase and split",
			text:      "Hello World",
			field:     "title",
			stopwords: nil,
			expected: []Token{
				{Term: "hello", Position: 0, Field: "title"},
				{Term: "world", Position: 1, Field: "title"},
			},
		},
		{
			name:      "remove accents",
			text:      "Pizzéria Romàñ",
			field:     "name",
			stopwords: nil,
			expected: []Token{
				{Term: "pizzeria", Position: 0, Field: "name"},
				{Term: "roman", Position: 1, Field: "name"},
			},
		},
		{
			name:      "stopwords",
			text:      "The Matrix is in a theater",
			field:     "desc",
			stopwords: stopwords,
			expected: []Token{
				{Term: "matrix", Position: 1, Field: "desc"}, // Position logic: "the" was pos 0, matrix is pos 1
				{Term: "theater", Position: 5, Field: "desc"},
			},
		},
		{
			name:      "punctuation",
			text:      "O'Connor, John - Jr.!",
			field:     "name",
			stopwords: nil,
			expected: []Token{
				{Term: "o", Position: 0, Field: "name"},
				{Term: "connor", Position: 1, Field: "name"},
				{Term: "john", Position: 2, Field: "name"},
				{Term: "jr", Position: 3, Field: "name"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Tokenize(tt.text, tt.field, Settings{StopWords: tt.stopwords})
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("Tokenize() \nGot:  %+v\nWant: %+v", got, tt.expected)
			}
		})
	}
}

func TestNormalizzaNumeri(t *testing.T) {
	entrambi := Settings{NormalizeDates: true, NormalizeAmounts: true}
	casi := []struct{ testo, atteso string }{
		{"del 14/01/2026", "del 20260114"},
		{"del 4/1/26", "del 20260104"},
		{"il 14.01.2026.", "il 20260114."},
		{"il 14-01-2026", "il 20260114"},
		{"il 14 Gennaio 2026", "il 20260114"},
		{"dal 24/09/2026 al 09/10/2026", "dal 20260924 al 20261009"},
		{"mese 13: 14/13/2026", "mese 13: 14/13/2026"},
		{"giorno 32: 32.01.2026", "giorno 32: 32.01.2026"},
		{"anno: 14/01/1850", "anno: 14/01/1850"},
		{"n. 3/2001", "n. 3/2001"},
		{"114/01/2026", "114/01/2026"},
		{"14/01/20261", "14/01/20261"},
		{"versione 1.2.2026.3", "versione 1.2.2026.3"},
		{"euro 1.234,56", "euro 1234,56"},
		{"euro 1.234.567", "euro 1234567"},
		{"euro 5.056,03 e 12.500", "euro 5056,03 e 12500"},
		{"ip 192.168.1.1", "ip 192.168.1.1"},
		{"codice 1.2345", "codice 1.2345"},
		{"senza niente", "senza niente"},
	}
	for _, c := range casi {
		if got := normalizzaNumeri(c.testo, entrambi); got != c.atteso {
			t.Errorf("normalizzaNumeri(%q) = %q; atteso %q", c.testo, got, c.atteso)
		}
	}
	if got := normalizzaNumeri("14/01/2026 1.234,56", Settings{}); got != "14/01/2026 1.234,56" {
		t.Errorf("spento cambia il testo: %q", got)
	}
	if got := normalizzaNumeri("14/01/2026 1.234,56", Settings{NormalizeAmounts: true}); got != "14/01/2026 1234,56" {
		t.Errorf("solo importi: %q", got)
	}
}

// A date is found whatever format the query writes it in, with both
// tokenizers, and its day stops matching a small number; an amount is found
// with or without thousands dots.
func TestDateEImportiNellaRicerca(t *testing.T) {
	for _, tokenizer := range []string{"", TokenizerStandard} {
		for _, acceso := range []bool{false, true} {
			settings := DefaultSettings()
			settings.Tokenizer = tokenizer
			settings.NormalizeDates, settings.NormalizeAmounts = acceso, acceso
			idx := NewInvertedIndex()
			idx.AddDocument("barra", map[string]interface{}{"title": "determina del 14/01/2026"}, settings)
			idx.AddDocument("mese", map[string]interface{}{"title": "delibera del 18 marzo 2025"}, settings)
			idx.AddDocument("importo", map[string]interface{}{"title": "liquidazione di euro 1.234,56"}, settings)
			idx.AddDocument("numero", map[string]interface{}{"title": "ordinanza 18"}, settings)
			trova := func(q string) []string {
				ids, _ := idx.Search(q, settings, "0", nil)
				return ids
			}
			if acceso {
				for _, q := range []string{"14/01/2026", "14.01.2026", "14 gennaio 2026", "14-01-2026", "14/01/26"} {
					if got := trova(q); !reflect.DeepEqual(got, []string{"barra"}) {
						t.Errorf("tokenizer %q, %q: %v", tokenizer, q, got)
					}
				}
				if got := trova("18/03/2025"); !reflect.DeepEqual(got, []string{"mese"}) {
					t.Errorf("tokenizer %q, 18/03/2025: %v", tokenizer, got)
				}
				if got := trova("18"); !reflect.DeepEqual(got, []string{"numero"}) {
					t.Errorf("tokenizer %q, 18: %v, il giorno di una data non deve combaciare", tokenizer, got)
				}
				for _, q := range []string{"1234,56", "1.234,56"} {
					if got := trova(q); !reflect.DeepEqual(got, []string{"importo"}) {
						t.Errorf("tokenizer %q, %q: %v", tokenizer, q, got)
					}
				}
			} else if got := trova("14 gennaio 2026"); len(got) != 0 {
				t.Errorf("spento, tokenizer %q: la data in lettere trova %v", tokenizer, got)
			}
		}
	}
}
