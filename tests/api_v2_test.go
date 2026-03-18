package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GeneralKoski/Koskidex/internal/server"
)

func seedV2Index(t *testing.T, srv *server.Server, indexName string, docs string) {
	t.Helper()

	req := httptest.NewRequest("POST", "/indexes", bytes.NewBufferString(`{"name": "`+indexName+`"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	req = httptest.NewRequest("POST", "/indexes/"+indexName+"/documents", bytes.NewBufferString(docs))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("Failed to seed %s: %d %s", indexName, w.Code, w.Body.String())
	}
}

func searchV2(t *testing.T, srv *server.Server, url string) map[string]interface{} {
	t.Helper()
	req := httptest.NewRequest("GET", url, nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Search failed: %d %s", w.Code, w.Body.String())
	}

	var res map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&res)
	return res
}

// === Fuzziness ===

func TestAPIFuzziness0NoResults(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	seedV2Index(t, srv, "fuzz", `[
		{"id": "1", "title": "The Matrix"},
		{"id": "2", "title": "Interstellar"}
	]`)

	res := searchV2(t, srv, "/indexes/fuzz/search?q=matrx&fuzziness=0")
	if res["total_hits"].(float64) != 0 {
		t.Errorf("fuzziness=0 should return 0 for typo, got %v", res["total_hits"])
	}
}

func TestAPIFuzziness1FindsTypo(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	seedV2Index(t, srv, "fuzz1", `[{"id": "1", "title": "The Matrix"}]`)

	res := searchV2(t, srv, "/indexes/fuzz1/search?q=matrx&fuzziness=1")
	if res["total_hits"].(float64) != 1 {
		t.Errorf("fuzziness=1 should find 'matrx' -> Matrix, got %v", res["total_hits"])
	}
}

func TestAPIFuzzinessAUTO(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	seedV2Index(t, srv, "fuzzauto", `[{"id": "1", "title": "Interstellar"}]`)

	res := searchV2(t, srv, "/indexes/fuzzauto/search?q=interstllar&fuzziness=AUTO")
	if res["total_hits"].(float64) != 1 {
		t.Errorf("fuzziness=AUTO should find 'interstllar', got %v", res["total_hits"])
	}
}

func TestAPIFuzzinessInvalid(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	seedV2Index(t, srv, "fuzzinv", `[{"id": "1", "title": "The Matrix"}]`)

	req := httptest.NewRequest("GET", "/indexes/fuzzinv/search?q=matrix&fuzziness=banana", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("invalid fuzziness should return 400, got %d", w.Code)
	}
}

// === Faceted Search ===

func TestAPIFacets(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	seedV2Index(t, srv, "facets", `[
		{"id": "1", "title": "The Matrix", "genre": "Sci-Fi"},
		{"id": "2", "title": "Interstellar", "genre": "Sci-Fi"},
		{"id": "3", "title": "The Godfather", "genre": "Crime"},
		{"id": "4", "title": "Inception", "genre": "Action"},
		{"id": "5", "title": "The Dark Knight", "genre": "Action"}
	]`)

	res := searchV2(t, srv, "/indexes/facets/search?q=the&facets=genre")

	facets, ok := res["facets"].(map[string]interface{})
	if !ok {
		t.Fatal("response should contain facets object")
	}

	genreFacet, ok := facets["genre"].(map[string]interface{})
	if !ok {
		t.Fatal("facets should contain genre")
	}

	// "the" matches Matrix, Godfather, Dark Knight
	if len(genreFacet) == 0 {
		t.Error("genre facet should not be empty")
	}
}

func TestAPIFacetsEmptyWhenNotRequested(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	seedV2Index(t, srv, "nofacets", `[{"id": "1", "title": "Hello", "genre": "Test"}]`)

	res := searchV2(t, srv, "/indexes/nofacets/search?q=hello")

	facets, ok := res["facets"].(map[string]interface{})
	if !ok {
		t.Fatal("response should contain facets object")
	}
	if len(facets) != 0 {
		t.Errorf("facets should be empty when not requested, got %v", facets)
	}
}

// === Explicit Sorting ===

func TestAPISortAsc(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	seedV2Index(t, srv, "sortasc", `[
		{"id": "1", "title": "Movie Alpha", "year": 2020},
		{"id": "2", "title": "Movie Beta", "year": 2010},
		{"id": "3", "title": "Movie Gamma", "year": 2015}
	]`)

	res := searchV2(t, srv, "/indexes/sortasc/search?q=movie&sort=year:asc")

	hits := res["hits"].([]interface{})
	if len(hits) != 3 {
		t.Fatalf("expected 3 hits, got %d", len(hits))
	}

	// Should be ordered: 2010, 2015, 2020
	firstDoc := hits[0].(map[string]interface{})["document"].(map[string]interface{})
	lastDoc := hits[2].(map[string]interface{})["document"].(map[string]interface{})

	firstYear := firstDoc["year"].(float64)
	lastYear := lastDoc["year"].(float64)

	if firstYear >= lastYear {
		t.Errorf("sort=year:asc: first year %.0f should be < last year %.0f", firstYear, lastYear)
	}
}

func TestAPISortDesc(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	seedV2Index(t, srv, "sortdesc", `[
		{"id": "1", "title": "Product Alpha", "price": 100},
		{"id": "2", "title": "Product Beta", "price": 500},
		{"id": "3", "title": "Product Gamma", "price": 250}
	]`)

	res := searchV2(t, srv, "/indexes/sortdesc/search?q=product&sort=price:desc")

	hits := res["hits"].([]interface{})
	if len(hits) != 3 {
		t.Fatalf("expected 3 hits, got %d", len(hits))
	}

	firstDoc := hits[0].(map[string]interface{})["document"].(map[string]interface{})
	lastDoc := hits[2].(map[string]interface{})["document"].(map[string]interface{})

	if firstDoc["price"].(float64) <= lastDoc["price"].(float64) {
		t.Error("sort=price:desc should order highest first")
	}
}

// === POST Search ===

func TestAPIPostSearch(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	seedV2Index(t, srv, "postsearch", `[
		{"id": "1", "title": "The Matrix", "genre": "Sci-Fi", "year": 1999},
		{"id": "2", "title": "Inception", "genre": "Action", "year": 2010},
		{"id": "3", "title": "Interstellar", "genre": "Sci-Fi", "year": 2014}
	]`)

	body := `{
		"q": "the",
		"fuzziness": "AUTO",
		"sort": "year:desc",
		"facets": "genre",
		"limit": 10
	}`

	req := httptest.NewRequest("POST", "/indexes/postsearch/search", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("POST search failed: %d %s", w.Code, w.Body.String())
	}

	var res map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&res)

	if res["total_hits"].(float64) == 0 {
		t.Error("POST search should return results")
	}

	facets, ok := res["facets"].(map[string]interface{})
	if !ok || len(facets) == 0 {
		t.Error("POST search with facets should return facet data")
	}
}

func TestAPIPostSearchWithFilter(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	seedV2Index(t, srv, "postfilter", `[
		{"id": "1", "title": "Old Movie", "year": 1990},
		{"id": "2", "title": "New Movie", "year": 2020}
	]`)

	body := `{"q": "movie", "filter": "year>2000"}`
	req := httptest.NewRequest("POST", "/indexes/postfilter/search", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("POST search with filter failed: %d", w.Code)
	}

	var res map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&res)

	if res["total_hits"].(float64) != 1 {
		t.Errorf("filter year>2000 should return 1 hit, got %v", res["total_hits"])
	}
}

// === Geospatial Filter ===

func TestAPIGeoFilter(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	seedV2Index(t, srv, "geo", `[
		{"id": "1", "title": "Duomo Milano", "_geo": {"lat": 45.4642, "lng": 9.1900}},
		{"id": "2", "title": "Colosseo Roma", "_geo": {"lat": 41.8902, "lng": 12.4922}},
		{"id": "3", "title": "Torre Pisa", "_geo": {"lat": 43.7230, "lng": 10.3966}}
	]`)

	// Search near Milan (5km radius) — should only find Duomo
	res := searchV2(t, srv, "/indexes/geo/search?q=duomo+colosseo+torre&fuzziness=AUTO&filter=distance(_geo,45.4650,9.1910)<5000")

	total := res["total_hits"].(float64)
	if total != 1 {
		t.Errorf("geo filter 5km around Milan should return 1 hit, got %.0f", total)
	}
}

func TestAPIGeoFilterLargeRadius(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	seedV2Index(t, srv, "geo2", `[
		{"id": "1", "title": "Place Milano", "_geo": {"lat": 45.4642, "lng": 9.1900}},
		{"id": "2", "title": "Place Roma", "_geo": {"lat": 41.8902, "lng": 12.4922}},
		{"id": "3", "title": "Place Pisa", "_geo": {"lat": 43.7230, "lng": 10.3966}}
	]`)

	// Large radius (600km) from center of Italy — should find all 3
	res := searchV2(t, srv, "/indexes/geo2/search?q=place&filter=distance(_geo,43.0,11.0)<600000")

	total := res["total_hits"].(float64)
	if total != 3 {
		t.Errorf("geo filter 600km should return all 3, got %.0f", total)
	}
}

// === Empty query returns facets key ===

func TestAPIEmptyQueryResponse(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	seedV2Index(t, srv, "empty", `[{"id": "1", "title": "Test"}]`)

	res := searchV2(t, srv, "/indexes/empty/search?q=")

	if _, ok := res["facets"]; !ok {
		t.Error("empty query response should contain facets key")
	}
	if res["total_hits"].(float64) != 0 {
		t.Error("empty query should return 0 hits")
	}
}

// === Multi-field sorting ===

func TestAPISortMultiField(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	seedV2Index(t, srv, "multisort", `[
		{"id": "1", "title": "Item Alpha", "category": "A", "price": 200},
		{"id": "2", "title": "Item Beta", "category": "A", "price": 100},
		{"id": "3", "title": "Item Gamma", "category": "B", "price": 150}
	]`)

	res := searchV2(t, srv, "/indexes/multisort/search?q=item&sort=category:asc,price:asc")

	hits := res["hits"].([]interface{})
	if len(hits) != 3 {
		t.Fatalf("expected 3 hits, got %d", len(hits))
	}

	// Category A first (2 items), then B. Within A, price asc: 100, 200
	first := hits[0].(map[string]interface{})["document"].(map[string]interface{})
	second := hits[1].(map[string]interface{})["document"].(map[string]interface{})

	if first["category"] != "A" || second["category"] != "A" {
		t.Error("first two results should be category A")
	}
	if first["price"].(float64) >= second["price"].(float64) {
		t.Error("within same category, should sort by price asc")
	}
}

// === Multiple facet fields ===

func TestAPIMultipleFacets(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	seedV2Index(t, srv, "mfacets", `[
		{"id": "1", "title": "Apple MacBook", "category": "Laptops", "brand": "Apple"},
		{"id": "2", "title": "Apple iPhone", "category": "Phones", "brand": "Apple"},
		{"id": "3", "title": "Samsung Galaxy", "category": "Phones", "brand": "Samsung"}
	]`)

	res := searchV2(t, srv, "/indexes/mfacets/search?q=apple+samsung&facets=category,brand")

	facets := res["facets"].(map[string]interface{})

	if _, ok := facets["category"]; !ok {
		t.Error("should have category facet")
	}
	if _, ok := facets["brand"]; !ok {
		t.Error("should have brand facet")
	}
}
