package engine

import (
	"fmt"
	"math"
	"math/rand"
	"runtime"
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
		idx.Search("great adventure world", settings, "AUTO", nil)
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
				Tokenize(tc.text, "field", Settings{StopWords: stopWords})
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

// vettoreCasuale returns a normalized vector of d dimensions, as bge-m3 gives.
func vettoreCasuale(r *rand.Rand, d int) []float64 {
	v := make([]float64, d)
	var n float64
	for i := range v {
		v[i] = r.NormFloat64()
		n += v[i] * v[i]
	}
	for i := range v {
		v[i] /= math.Sqrt(n)
	}
	return v
}

// indiceConVettori loads n documents with d-dimensional vectors given as
// []interface{}, the way JSON and gob decode them, and returns the live heap
// after loading, in MB.
func indiceConVettori(n, d int, s Settings) (*InvertedIndex, float64) {
	r := rand.New(rand.NewSource(20260925))
	runtime.GC()
	var prima runtime.MemStats
	runtime.ReadMemStats(&prima)
	idx := NewInvertedIndex()
	for i := 0; i < n; i++ {
		v := vettoreCasuale(r, d)
		lista := make([]interface{}, d)
		for j, x := range v {
			lista[j] = x
		}
		id := fmt.Sprintf("doc_%d", i)
		idx.AddDocument(id, map[string]interface{}{
			"id":      id,
			"title":   fmt.Sprintf("The great adventure of document number %d in the world", i),
			"_vector": lista,
		}, s)
	}
	runtime.GC()
	var dopo runtime.MemStats
	runtime.ReadMemStats(&dopo)
	return idx, float64(dopo.HeapAlloc-prima.HeapAlloc) / (1 << 20)
}

// BenchmarkRicercaIbrida measures a hybrid search over 10,000 documents with
// 1,024-dimensional vectors, a lexical query that matches all of them. See
// risultati/esperimenti/2026-09-25_costo-vettori/ in the thesis repository.
func BenchmarkRicercaIbrida(b *testing.B) {
	for _, modo := range []string{"", HybridUnion} {
		nome := "rerank"
		if modo != "" {
			nome = modo
		}
		b.Run(nome, func(b *testing.B) {
			s := DefaultSettings()
			s.SearchableFields = []string{"title"}
			s.HybridMode = modo
			idx, mb := indiceConVettori(10000, 1024, s)
			q := vettoreCasuale(rand.New(rand.NewSource(1)), 1024)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				idx.Search("great adventure world", s, "0", q)
			}
			b.ReportMetric(mb, "MB-heap")
		})
	}
}
