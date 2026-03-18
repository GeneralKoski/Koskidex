package engine

import (
	"fmt"
	"testing"
)

func benchSetup(n int) (*InvertedIndex, Settings) {
	idx := NewInvertedIndex()
	settings := DefaultSettings()
	for i := 0; i < n; i++ {
		doc := map[string]interface{}{
			"id":    fmt.Sprintf("doc_%d", i),
			"title": fmt.Sprintf("The great adventure of document number %d in the world", i),
			"body":  fmt.Sprintf("This is the body text for document %d with various interesting words and phrases", i),
		}
		idx.AddDocument(fmt.Sprintf("doc_%d", i), doc, settings)
	}
	return idx, settings
}

func BenchmarkAddDocument(b *testing.B) {
	idx := NewInvertedIndex()
	settings := DefaultSettings()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		doc := map[string]interface{}{
			"id":    fmt.Sprintf("doc_%d", i),
			"title": fmt.Sprintf("The great adventure number %d", i),
			"body":  fmt.Sprintf("Body text for document %d with words", i),
		}
		idx.AddDocument(fmt.Sprintf("doc_%d", i), doc, settings)
	}
}

func BenchmarkSearch(b *testing.B) {
	idx, settings := benchSetup(10000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx.Search("great adventure world", settings)
	}
}

func BenchmarkSearchExact(b *testing.B) {
	idx, settings := benchSetup(10000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx.SearchExact("great adventure", settings)
	}
}

func BenchmarkFuzzySearchTerms(b *testing.B) {
	idx, settings := benchSetup(5000)
	_ = settings
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		idx.FuzzySearchTerms("advnture", 1, true)
	}
}

func BenchmarkTokenize(b *testing.B) {
	stopWords := map[string]bool{"the": true, "a": true, "is": true}
	texts := []struct {
		name string
		text string
	}{
		{"short", "hello world"},
		{"medium", "The quick brown fox jumps over the lazy dog near the river"},
		{"long", "In a hole in the ground there lived a hobbit not a nasty dirty wet hole filled with the ends of worms and an oozy smell nor yet a dry bare sandy hole with nothing in it to sit down on or to eat it was a hobbit hole and that means comfort"},
	}
	for _, tc := range texts {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				Tokenize(tc.text, "field", stopWords)
			}
		})
	}
}

func BenchmarkDamerauLevenshtein(b *testing.B) {
	pairs := []struct {
		name string
		a, z string
	}{
		{"identical", "hello", "hello"},
		{"one_swap", "hello", "hlelo"},
		{"one_sub", "hello", "hallo"},
		{"long", "international", "internatioanl"},
	}
	for _, tc := range pairs {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				DamerauLevenshtein(tc.a, tc.z)
			}
		})
	}
}
