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
			got := Tokenize(tt.text, tt.field, tt.stopwords)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("Tokenize() \nGot:  %+v\nWant: %+v", got, tt.expected)
			}
		})
	}
}
