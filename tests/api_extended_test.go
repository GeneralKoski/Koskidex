package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/GeneralKoski/Koskidex/internal/manager"
	"github.com/GeneralKoski/Koskidex/internal/server"
	"github.com/GeneralKoski/Koskidex/internal/storage"
)

func setupTestServerWithAuth(t *testing.T, apiKey string) (*server.Server, func()) {
	dataDir, err := os.MkdirTemp("", "koskidex_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	opts := storage.Options{DataDir: dataDir}
	mgr, err := manager.NewManager(opts)
	if err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	srv := server.NewServer(mgr, apiKey, 0)

	cleanup := func() {
		mgr.Close()
		os.RemoveAll(dataDir)
	}

	return srv, cleanup
}

func TestHealthEndpoint(t *testing.T) {
	srv, cleanup := setupTestServerWithAuth(t, "secret-key")
	defer cleanup()

	// Health should work WITHOUT auth
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 for /health, got %d", w.Code)
	}

	var res map[string]string
	json.NewDecoder(w.Body).Decode(&res)
	if res["status"] != "ok" {
		t.Fatalf("Expected status 'ok', got %q", res["status"])
	}
}

func TestAuthRequired(t *testing.T) {
	srv, cleanup := setupTestServerWithAuth(t, "secret-key")
	defer cleanup()

	// Request without auth should fail
	req := httptest.NewRequest("GET", "/indexes", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 without auth, got %d", w.Code)
	}

	// Request with auth should succeed
	req = httptest.NewRequest("GET", "/indexes", nil)
	req.Header.Set("Authorization", "Bearer secret-key")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 with auth, got %d", w.Code)
	}
}

func TestDuplicateIndexCreation(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	body := `{"name": "dup_index"}`
	req := httptest.NewRequest("POST", "/indexes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201, got %d", w.Code)
	}

	// Create again
	req = httptest.NewRequest("POST", "/indexes", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("Expected 409 for duplicate, got %d", w.Code)
	}
}

func TestNonExistentIndex(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	req := httptest.NewRequest("GET", "/indexes/nonexistent", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected 404, got %d", w.Code)
	}
}

func TestSingleDocumentAdd(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create index
	req := httptest.NewRequest("POST", "/indexes", bytes.NewBufferString(`{"name": "single_doc"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Add single doc (not array)
	doc := `{"id": "1", "title": "Single Document"}`
	req = httptest.NewRequest("POST", "/indexes/single_doc/documents", bytes.NewBufferString(doc))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("Expected 202 for single doc, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Verify the document exists
	req = httptest.NewRequest("GET", "/indexes/single_doc/documents/1", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", w.Code)
	}
}

func TestPagination(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create index and add docs
	req := httptest.NewRequest("POST", "/indexes", bytes.NewBufferString(`{"name": "pag_test"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var docs []map[string]interface{}
	for i := 0; i < 10; i++ {
		docs = append(docs, map[string]interface{}{
			"id":    string(rune('a' + i)),
			"title": "test document",
		})
	}
	body, _ := json.Marshal(docs)
	req = httptest.NewRequest("POST", "/indexes/pag_test/documents", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Search with limit=3
	req = httptest.NewRequest("GET", "/indexes/pag_test/search?q=test&limit=3", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var res map[string]interface{}
	json.NewDecoder(w.Body).Decode(&res)

	hits := res["hits"].([]interface{})
	if len(hits) != 3 {
		t.Fatalf("Expected 3 hits with limit=3, got %d", len(hits))
	}
	if res["total_hits"].(float64) != 10 {
		t.Fatalf("Expected total_hits=10, got %v", res["total_hits"])
	}
	if res["limit"].(float64) != 3 {
		t.Fatalf("Expected limit=3 in response, got %v", res["limit"])
	}
}

func TestSearchFilter(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Create index
	req := httptest.NewRequest("POST", "/indexes", bytes.NewBufferString(`{"name": "filter_test"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	docs := `[
		{"id": "1", "title": "The Matrix", "genre": "sci-fi", "year": 1999},
		{"id": "2", "title": "The Godfather", "genre": "drama", "year": 1972},
		{"id": "3", "title": "The Dark Knight", "genre": "action", "year": 2008}
	]`
	req = httptest.NewRequest("POST", "/indexes/filter_test/documents", bytes.NewBufferString(docs))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	// Search with genre filter
	req = httptest.NewRequest("GET", "/indexes/filter_test/search?q=The&filter=genre=sci-fi", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var res map[string]interface{}
	json.NewDecoder(w.Body).Decode(&res)

	if res["total_hits"].(float64) != 1 {
		t.Fatalf("Expected 1 hit with genre=sci-fi filter, got %v", res["total_hits"])
	}
}
