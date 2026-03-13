package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/GeneralKoski/Koskidex/internal/manager"
)

// Server wraps the API routes and manager
type Server struct {
	mux    *http.ServeMux
	mgr    *manager.Manager
	apiKey string
}

// NewServer initializes the HTTP routing
func NewServer(mgr *manager.Manager, apiKey string) *Server {
	s := &Server{
		mux:    http.NewServeMux(),
		mgr:    mgr,
		apiKey: apiKey,
	}
	s.routes()
	return s
}

// ServeHTTP implements http.Handler interface
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// CORS Headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Preflight check
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Authentication
	if s.apiKey != "" {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") || strings.TrimPrefix(auth, "Bearer ") != s.apiKey {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	s.mux.ServeHTTP(w, r)
}

// routes sets up the routing
func (s *Server) routes() {
	// Indexes
	s.mux.HandleFunc("POST /indexes", s.handleCreateIndex)
	s.mux.HandleFunc("GET /indexes", s.handleListIndexes)
	s.mux.HandleFunc("GET /indexes/{name}", s.handleGetIndex)
	s.mux.HandleFunc("DELETE /indexes/{name}", s.handleDeleteIndex)

	// Documents
	s.mux.HandleFunc("POST /indexes/{name}/documents", s.handleAddDocuments)
	s.mux.HandleFunc("GET /indexes/{name}/documents/{id}", s.handleGetDocument)
	s.mux.HandleFunc("DELETE /indexes/{name}/documents/{id}", s.handleDeleteDocument)

	// Settings
	s.mux.HandleFunc("GET /indexes/{name}/settings", s.handleGetSettings)
	s.mux.HandleFunc("PUT /indexes/{name}/settings", s.handleUpdateSettings)

	// Search
	s.mux.HandleFunc("GET /indexes/{name}/search", s.handleSearch)
}

func sendJSON(w http.ResponseWriter, status int, data interface{}) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func sendError(w http.ResponseWriter, status int, message string) {
	sendJSON(w, status, map[string]string{"error": message})
}
