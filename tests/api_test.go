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

func setupTestServer(t *testing.T) (*server.Server, func()) {
	// Create a temporary data dir
	dataDir, err := os.MkdirTemp("", "koskidex_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	opts := storage.Options{DataDir: dataDir}
	mgr, err := manager.NewManager(opts)
	if err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	srv := server.NewServer(mgr, "", 0)

	cleanup := func() {
		mgr.Close()
		os.RemoveAll(dataDir)
	}

	return srv, cleanup
}

func TestAPIIntegration(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// 1. Create Index
	reqBody := `{"name": "test_index"}`
	req := httptest.NewRequest("POST", "/indexes", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created, got %d. Body: %s", w.Code, w.Body.String())
	}

	// 2. Add Documents
	docs := `[
		{"id": "1", "title": "The Matrix", "genre": "sci-fi"},
		{"id": "2", "title": "The Godfather", "genre": "drama"}
	]`
	req = httptest.NewRequest("POST", "/indexes/test_index/documents", bytes.NewBufferString(docs))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Fatalf("Expected 202 Accepted, got %d. Body: %s", w.Code, w.Body.String())
	}

	// 3. Search Document Exact Matches
	req = httptest.NewRequest("GET", "/indexes/test_index/search?q=Matrix", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
	}

	var searchRes map[string]interface{}
	json.NewDecoder(w.Body).Decode(&searchRes)

	if searchRes["total_hits"].(float64) != 1 {
		t.Fatalf("Expected 1 hit for 'Matrix', got %v", searchRes["total_hits"])
	}

	// 4. Delete Index
	req = httptest.NewRequest("DELETE", "/indexes/test_index", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for delete, got %d", w.Code)
	}
}
