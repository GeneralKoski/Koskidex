package engine

import (
	"math"
	"testing"
)

// === 1. Dynamic Typo Tolerance (Fuzziness) ===

func TestMaxTyposFuzzinessExplicit(t *testing.T) {
	ts := TypoSettings{Enabled: true, MinWordLengthOneTypo: 4, MinWordLengthTwoTypos: 8}

	if got := MaxTypos("hello", ts, "0"); got != 0 {
		t.Errorf("fuzziness=0: want 0, got %d", got)
	}
	if got := MaxTypos("hello", ts, "1"); got != 1 {
		t.Errorf("fuzziness=1: want 1, got %d", got)
	}
	if got := MaxTypos("hello", ts, "2"); got != 2 {
		t.Errorf("fuzziness=2: want 2, got %d", got)
	}
}

func TestMaxTyposFuzzinessAUTO(t *testing.T) {
	ts := TypoSettings{Enabled: true, MinWordLengthOneTypo: 4, MinWordLengthTwoTypos: 8}

	// AUTO (empty or "AUTO"): <4 chars = 0, 4-7 chars = 1, >=8 chars = 2
	if got := MaxTypos("cat", ts, ""); got != 0 {
		t.Errorf("3 chars AUTO: want 0, got %d", got)
	}
	if got := MaxTypos("hello", ts, "AUTO"); got != 1 {
		t.Errorf("5 chars AUTO: want 1, got %d", got)
	}
	if got := MaxTypos("interstellar", ts, ""); got != 2 {
		t.Errorf("12 chars AUTO: want 2, got %d", got)
	}
}

func TestMaxTyposExplicitOverridesDisabled(t *testing.T) {
	ts := TypoSettings{Enabled: false}

	// Explicit fuzziness should work even when typo tolerance is disabled
	if got := MaxTypos("hello", ts, "1"); got != 1 {
		t.Errorf("fuzziness=1 with Enabled=false: want 1, got %d", got)
	}
	if got := MaxTypos("hello", ts, "2"); got != 2 {
		t.Errorf("fuzziness=2 with Enabled=false: want 2, got %d", got)
	}
	// But disabled + no explicit fuzziness = 0
	if got := MaxTypos("hello", ts, ""); got != 0 {
		t.Errorf("empty fuzziness with Enabled=false: want 0, got %d", got)
	}
}

func TestSearchFuzziness0(t *testing.T) {
	idx := NewInvertedIndex()
	s := DefaultSettings()
	idx.AddDocument("1", map[string]interface{}{"title": "The Matrix"}, s)

	// fuzziness=0 should NOT find "matrx" (typo)
	docs, _ := idx.Search("matrx", s, "0", nil)
	if len(docs) != 0 {
		t.Errorf("fuzziness=0 should return no results for typo, got %v", docs)
	}

	// fuzziness=0 should find exact "matrix"
	docs, _ = idx.Search("matrix", s, "0", nil)
	if len(docs) != 1 {
		t.Errorf("fuzziness=0 should find exact match, got %v", docs)
	}
}

func TestSearchFuzziness1(t *testing.T) {
	idx := NewInvertedIndex()
	s := DefaultSettings()
	idx.AddDocument("1", map[string]interface{}{"title": "The Matrix"}, s)

	// fuzziness=1 should find "matrx" (1 deletion)
	docs, _ := idx.Search("matrx", s, "1", nil)
	if len(docs) != 1 || docs[0] != "1" {
		t.Errorf("fuzziness=1 should find 'matrx' -> Matrix, got %v", docs)
	}
}

// === 2. Field Weighting ===

func TestFieldWeightingBoost(t *testing.T) {
	idx := NewInvertedIndex()
	s := DefaultSettings()
	s.FieldWeights = map[string]float64{"name": 5, "description": 1}

	idx.AddDocument("1", map[string]interface{}{"name": "apple", "description": "a fruit"}, s)
	idx.AddDocument("2", map[string]interface{}{"name": "banana", "description": "apple colored"}, s)

	docs, _ := idx.Search("apple", s, "AUTO", nil)
	if len(docs) < 2 {
		t.Fatalf("expected 2 results, got %d", len(docs))
	}
	// doc1 has "apple" in name (weight 5), doc2 has "apple" in description (weight 1)
	// doc1 should rank higher
	if docs[0] != "1" {
		t.Errorf("expected doc1 (name match, weight 5) to rank first, got %v", docs)
	}
}

func TestFieldWeightingDefaultWeight(t *testing.T) {
	idx := NewInvertedIndex()
	s := DefaultSettings()
	// No field weights set — should default to 1.0 for all fields

	idx.AddDocument("1", map[string]interface{}{"title": "hello world"}, s)
	docs, _ := idx.Search("hello", s, "AUTO", nil)
	if len(docs) != 1 {
		t.Errorf("expected 1 result with default weights, got %d", len(docs))
	}
}

// === 3. Geospatial Search (Haversine) ===

func TestHaversineDistance(t *testing.T) {
	// Milan to Rome ~477 km
	milan := [2]float64{45.4642, 9.1900}
	rome := [2]float64{41.9028, 12.4964}

	dist := haversineDistance(milan[0], milan[1], rome[0], rome[1])
	expected := 477000.0 // ~477 km in meters

	if math.Abs(dist-expected) > 10000 { // 10 km tolerance
		t.Errorf("Milan-Rome distance: got %.0f m, expected ~%.0f m", dist, expected)
	}
}

func TestHaversineDistanceSamePoint(t *testing.T) {
	dist := haversineDistance(45.0, 9.0, 45.0, 9.0)
	if dist != 0 {
		t.Errorf("same point distance should be 0, got %f", dist)
	}
}

func TestGeoFilter(t *testing.T) {
	// Milan coordinates
	doc := map[string]interface{}{
		"name": "Duomo di Milano",
		"_geo": map[string]interface{}{"lat": 45.4642, "lng": 9.1900},
	}

	// Filter: distance from a point 1km away, radius 5000m — should pass
	filters := ParseFilters("distance(_geo,45.4650,9.1910)<5000")
	if !ApplyFilters(doc, filters) {
		t.Error("geo filter should pass for nearby point")
	}

	// Filter: distance from Rome (~477km), radius 10000m — should fail
	filters = ParseFilters("distance(_geo,41.9028,12.4964)<10000")
	if ApplyFilters(doc, filters) {
		t.Error("geo filter should fail for distant point")
	}
}

func TestGeoFilterMissingField(t *testing.T) {
	doc := map[string]interface{}{"name": "No geo"}
	filters := ParseFilters("distance(_geo,45.0,9.0)<5000")
	if ApplyFilters(doc, filters) {
		t.Error("geo filter should fail when _geo field is missing")
	}
}

// === 4. Faceted Search ===
// Facets are computed in the handler layer, but we can test the building blocks

func TestFacetsAggregation(t *testing.T) {
	idx := NewInvertedIndex()
	s := DefaultSettings()

	idx.AddDocument("1", map[string]interface{}{"title": "The Matrix", "genre": "Sci-Fi"}, s)
	idx.AddDocument("2", map[string]interface{}{"title": "Interstellar", "genre": "Sci-Fi"}, s)
	idx.AddDocument("3", map[string]interface{}{"title": "The Godfather", "genre": "Crime"}, s)
	idx.AddDocument("4", map[string]interface{}{"title": "Goodfellas", "genre": "Crime"}, s)
	idx.AddDocument("5", map[string]interface{}{"title": "Inception", "genre": "Action"}, s)

	docs, _ := idx.Search("the", s, "AUTO", nil)

	// Simulate facet computation (same logic as handler)
	facets := make(map[string]int)
	for _, id := range docs {
		if doc, ok := idx.GetDocument(id); ok {
			if val, ok := doc["genre"]; ok {
				if strVal, ok := val.(string); ok {
					facets[strVal]++
				}
			}
		}
	}

	// "the" matches Matrix, Godfather — so Sci-Fi:1, Crime:1
	if facets["Sci-Fi"] != 1 {
		t.Errorf("expected Sci-Fi:1, got %d", facets["Sci-Fi"])
	}
	if facets["Crime"] != 1 {
		t.Errorf("expected Crime:1, got %d", facets["Crime"])
	}
}

// === 5. Vector Search (Cosine Similarity) ===

func TestCosineSimilarityIdentical(t *testing.T) {
	a := []float64{1, 2, 3}
	sim := cosineSimilarity(a, a)
	if math.Abs(sim-1.0) > 1e-9 {
		t.Errorf("identical vectors should have similarity 1.0, got %f", sim)
	}
}

func TestCosineSimilarityOrthogonal(t *testing.T) {
	a := []float64{1, 0, 0}
	b := []float64{0, 1, 0}
	sim := cosineSimilarity(a, b)
	if math.Abs(sim) > 1e-9 {
		t.Errorf("orthogonal vectors should have similarity 0, got %f", sim)
	}
}

func TestCosineSimilarityOpposite(t *testing.T) {
	a := []float64{1, 2, 3}
	b := []float64{-1, -2, -3}
	sim := cosineSimilarity(a, b)
	if math.Abs(sim-(-1.0)) > 1e-9 {
		t.Errorf("opposite vectors should have similarity -1.0, got %f", sim)
	}
}

func TestCosineSimilarityZeroVector(t *testing.T) {
	a := []float64{0, 0, 0}
	b := []float64{1, 2, 3}
	sim := cosineSimilarity(a, b)
	if sim != 0 {
		t.Errorf("zero vector similarity should be 0, got %f", sim)
	}
}

func TestToFloat64Array(t *testing.T) {
	// []float64 passthrough
	arr, ok := toFloat64Array([]float64{1.0, 2.0, 3.0})
	if !ok || len(arr) != 3 || arr[0] != 1.0 {
		t.Error("[]float64 should pass through")
	}

	// []interface{} with float64 values (JSON deserialization)
	arr, ok = toFloat64Array([]interface{}{1.0, 2.0, 3.0})
	if !ok || len(arr) != 3 || arr[0] != 1.0 {
		t.Error("[]interface{} with float64 should convert")
	}

	// []interface{} with non-numeric values
	_, ok = toFloat64Array([]interface{}{"not", "numbers"})
	if ok {
		t.Error("non-numeric []interface{} should fail")
	}

	// unsupported type
	_, ok = toFloat64Array("not an array")
	if ok {
		t.Error("string should fail")
	}
}

func TestVectorOnlySearch(t *testing.T) {
	idx := NewInvertedIndex()
	s := DefaultSettings()

	idx.AddDocument("1", map[string]interface{}{
		"title":   "Similar",
		"_vector": []interface{}{1.0, 0.0, 0.0},
	}, s)
	idx.AddDocument("2", map[string]interface{}{
		"title":   "Different",
		"_vector": []interface{}{0.0, 1.0, 0.0},
	}, s)

	// Vector-only search (no text query) — should find doc closest to query vector
	queryVec := []float64{1.0, 0.0, 0.0}
	docs, _ := idx.Search("", s, "AUTO", queryVec)

	if len(docs) < 2 {
		t.Fatalf("expected 2 results, got %d", len(docs))
	}
	if docs[0] != "1" {
		t.Errorf("expected doc1 (identical vector) to rank first, got %v", docs)
	}
}

func TestHybridSearch(t *testing.T) {
	idx := NewInvertedIndex()
	s := DefaultSettings()

	idx.AddDocument("1", map[string]interface{}{
		"title":   "The Matrix",
		"_vector": []interface{}{0.9, 0.1, 0.0},
	}, s)
	idx.AddDocument("2", map[string]interface{}{
		"title":   "Matrix Reloaded",
		"_vector": []interface{}{0.1, 0.9, 0.0},
	}, s)

	// Hybrid: text "matrix" + vector close to doc1
	queryVec := []float64{1.0, 0.0, 0.0}
	docs, _ := idx.Search("matrix", s, "AUTO", queryVec)

	if len(docs) < 2 {
		t.Fatalf("expected 2 results, got %d", len(docs))
	}
	// doc1 should rank higher: text match + high vector similarity
	if docs[0] != "1" {
		t.Errorf("expected doc1 to rank first in hybrid search, got %v", docs)
	}
}

// === 6. Explicit Sorting ===
// Sorting is in the handler layer, but we test that Search returns correct ranked results

func TestSearchReturnsScoreBasedRanking(t *testing.T) {
	idx := NewInvertedIndex()
	s := DefaultSettings()

	idx.AddDocument("1", map[string]interface{}{"title": "apple banana"}, s)
	idx.AddDocument("2", map[string]interface{}{"title": "apple"}, s)

	docs, _ := idx.Search("apple banana", s, "AUTO", nil)

	if len(docs) < 1 {
		t.Fatal("expected results")
	}
	// doc1 matches both terms, should rank higher
	if docs[0] != "1" {
		t.Errorf("expected doc1 (2 word matches) first, got %v", docs)
	}
}

// === 7. WAL Operations ===

func TestWALOperationStruct(t *testing.T) {
	// Verify the search works correctly after simulating WAL replay
	// (add docs, delete some, verify state)
	idx := NewInvertedIndex()
	s := DefaultSettings()

	// Simulate WAL replay: add 3, delete 1
	idx.AddDocument("1", map[string]interface{}{"title": "Keep This"}, s)
	idx.AddDocument("2", map[string]interface{}{"title": "Delete This"}, s)
	idx.AddDocument("3", map[string]interface{}{"title": "Also Keep"}, s)
	idx.DeleteDocument("2")

	if idx.GetDocCount() != 2 {
		t.Errorf("expected 2 docs after delete, got %d", idx.GetDocCount())
	}

	docs, _ := idx.Search("delete", s, "AUTO", nil)
	if len(docs) != 0 {
		t.Errorf("deleted doc should not appear in search, got %v", docs)
	}

	docs, _ = idx.Search("keep", s, "AUTO", nil)
	if len(docs) != 2 {
		t.Errorf("expected 2 results for 'keep', got %d", len(docs))
	}
}
