package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/general-koski/koskidex/internal/engine"
	"github.com/general-koski/koskidex/internal/manager"
)

type createIndexReq struct {
	Name string `json:"name"`
}

func (s *Server) handleCreateIndex(w http.ResponseWriter, r *http.Request) {
	var req createIndexReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if req.Name == "" {
		sendError(w, http.StatusBadRequest, "Index name is required")
		return
	}

	err := s.mgr.CreateIndex(req.Name)
	if err == manager.ErrIndexAlreadyExists {
		sendError(w, http.StatusConflict, "Index already exists")
		return
	} else if err != nil {
		sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	sendJSON(w, http.StatusCreated, map[string]string{"name": req.Name, "message": "Index created"})
}

func (s *Server) handleListIndexes(w http.ResponseWriter, r *http.Request) {
	indexes := s.mgr.ListIndexes()
	sendJSON(w, http.StatusOK, map[string]interface{}{"indexes": indexes})
}

func (s *Server) handleGetIndex(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	idx, err := s.mgr.GetIndex(name)
	if err == manager.ErrIndexNotFound {
		sendError(w, http.StatusNotFound, "Index not found")
		return
	} else if err != nil {
		sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"name": name,
		"docs": idx.Engine.GetDocCount(),
	})
}

func (s *Server) handleDeleteIndex(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	err := s.mgr.DeleteIndex(name)
	if err == manager.ErrIndexNotFound {
		sendError(w, http.StatusNotFound, "Index not found")
		return
	} else if err != nil {
		sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	sendJSON(w, http.StatusOK, map[string]string{"message": "Index deleted"})
}

func (s *Server) handleAddDocuments(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")

	var docs []map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&docs); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid JSON payload (expected array of objects)")
		return
	}

	if err := s.mgr.AddDocuments(name, docs); err != nil {
		if err == manager.ErrIndexNotFound {
			sendError(w, http.StatusNotFound, "Index not found")
			return
		}
		sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	sendJSON(w, http.StatusAccepted, map[string]interface{}{"message": "Documents added", "count": len(docs)})
}

func (s *Server) handleGetDocument(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	id := r.PathValue("id")

	idx, err := s.mgr.GetIndex(name)
	if err == manager.ErrIndexNotFound {
		sendError(w, http.StatusNotFound, "Index not found")
		return
	}

	doc, ok := idx.Engine.GetDocument(id)
	if !ok {
		sendError(w, http.StatusNotFound, "Document not found")
		return
	}

	sendJSON(w, http.StatusOK, doc)
}

func (s *Server) handleDeleteDocument(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	id := r.PathValue("id")

	if err := s.mgr.DeleteDocument(name, id); err != nil {
		if err == manager.ErrIndexNotFound {
			sendError(w, http.StatusNotFound, "Index not found")
			return
		}
		sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	sendJSON(w, http.StatusOK, map[string]string{"message": "Document deleted"})
}

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	idx, err := s.mgr.GetIndex(name)
	if err != nil {
		if err == manager.ErrIndexNotFound {
			sendError(w, http.StatusNotFound, "Index not found")
			return
		}
		sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	sendJSON(w, http.StatusOK, idx.Settings)
}

func (s *Server) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var settings engine.Settings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		sendError(w, http.StatusBadRequest, "Invalid JSON settings")
		return
	}

	if err := s.mgr.UpdateSettings(name, settings); err != nil {
		if err == manager.ErrIndexNotFound {
			sendError(w, http.StatusNotFound, "Index not found")
			return
		}
		sendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	sendJSON(w, http.StatusOK, map[string]string{"message": "Settings updated"})
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	name := r.PathValue("name")
	query := r.URL.Query().Get("q")
	// For phase 1 we only support exact search, phase 2 we do typo tolerance

	if query == "" {
		sendError(w, http.StatusBadRequest, "Missing 'q' query parameter")
		return
	}

	idx, err := s.mgr.GetIndex(name)
	if err == manager.ErrIndexNotFound {
		sendError(w, http.StatusNotFound, "Index not found")
		return
	}

	docIDs, highlights := idx.Engine.Search(query, idx.Settings)

	var hits []map[string]interface{}
	for _, id := range docIDs {
		if doc, ok := idx.Engine.GetDocument(id); ok {
			// Filter DisplayedFields if configured
			displayDoc := make(map[string]interface{})
			if len(idx.Settings.DisplayedFields) > 0 {
				for _, f := range idx.Settings.DisplayedFields {
					if val, ok := doc[f]; ok {
						displayDoc[f] = val
					}
				}
				// Always include ID
				if _, ok := displayDoc["id"]; !ok {
					displayDoc["id"] = id
				}
			} else {
				displayDoc = doc
			}

			// Apply highlighting
			highlightsMap := make(map[string]string)
			for k, v := range displayDoc {
				if strVal, isStr := v.(string); isStr {
					if terms, hasHl := highlights[id]; hasHl {
						highlightsMap[k] = engine.Highlight(strVal, terms)
					}
				}
			}

			hits = append(hits, map[string]interface{}{
				"id":         id,
				"document":   displayDoc,
				"highlights": highlightsMap,
			})
		}
	}

	elapsed := time.Since(start)

	sendJSON(w, http.StatusOK, map[string]interface{}{
		"query":              query,
		"hits":               hits,
		"total_hits":         len(hits),
		"processing_time_ms": elapsed.Milliseconds(),
	})
}
