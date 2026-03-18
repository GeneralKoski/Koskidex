package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/GeneralKoski/Koskidex/internal/engine"
	"github.com/GeneralKoski/Koskidex/internal/manager"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	uptime := time.Since(s.startTime).Round(time.Second).String()
	sendJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"uptime": uptime,
	})
}

func (s *Server) handleRobots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	// Prevent crawlers from exhausting the API search endpoint and wasting crawl budget
	_, _ = w.Write([]byte("User-agent: *\nDisallow: /indexes/*/search\n"))
}

func (s *Server) handleSitemap(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	idx, err := s.mgr.GetIndex(name)
	if err != nil {
		http.Error(w, "Index not found", http.StatusNotFound)
		return
	}

	baseUrl := idx.Settings.Sitemap.BaseUrl
	if baseUrl == "" {
		http.Error(w, "Sitemap base_url not configured for this index in settings", http.StatusBadRequest)
		return
	}

	urlField := idx.Settings.Sitemap.UrlField
	if urlField == "" {
		urlField = "url"
	}

	freq := idx.Settings.Sitemap.ChangeFreq
	if freq == "" {
		freq = "weekly"
	}

	w.Header().Set("Content-Type", "application/xml")
	_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>` + "\n"))
	_, _ = w.Write([]byte(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n"))

	docs := idx.Engine.GetAllDocs()
	for id, doc := range docs {
		loc := ""
		if val, ok := doc[urlField]; ok {
			loc = baseUrl + fmt.Sprintf("%v", val)
		} else {
			// fallback relative ID path
			loc = baseUrl + "/" + id
		}

		// Ensure proper formatting if baseUrl ends with / and value starts with /
		loc = strings.ReplaceAll(loc, "///", "/")
		loc = strings.ReplaceAll(loc, "https:/", "https://")
		loc = strings.ReplaceAll(loc, "http:/", "http://")

		_, _ = w.Write([]byte(fmt.Sprintf("  <url>\n    <loc>%s</loc>\n    <changefreq>%s</changefreq>\n  </url>\n", loc, freq)))
	}

	_, _ = w.Write([]byte(`</urlset>` + "\n"))
}

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

	contentType := r.Header.Get("Content-Type")

	// 1. Support Multipart File Uploads
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB max
			sendError(w, http.StatusBadRequest, "Failed to parse multipart form")
			return
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			sendError(w, http.StatusBadRequest, "File field 'file' is required for multipart uploads")
			return
		}
		defer func() { _ = file.Close() }()
		if err := json.NewDecoder(file).Decode(&docs); err != nil {
			sendError(w, http.StatusBadRequest, "Invalid JSON in uploaded file")
			return
		}
	} else {
		// 2. Support Polymorphic JSON (Single object or Array)
		var rawBody json.RawMessage
		if err := json.NewDecoder(r.Body).Decode(&rawBody); err != nil {
			sendError(w, http.StatusBadRequest, "Invalid JSON payload")
			return
		}

		// Check if it's an array or an object
		if len(rawBody) > 0 && rawBody[0] == '[' {
			if err := json.Unmarshal(rawBody, &docs); err != nil {
				sendError(w, http.StatusBadRequest, "Invalid JSON array")
				return
			}
		} else {
			var doc map[string]interface{}
			if err := json.Unmarshal(rawBody, &doc); err != nil {
				sendError(w, http.StatusBadRequest, "Invalid JSON object")
				return
			}
			docs = append(docs, doc)
		}
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

type SearchRequest struct {
	Q         string    `json:"q"`
	Vector    []float64 `json:"vector"`
	Limit     int       `json:"limit"`
	Offset    int       `json:"offset"`
	Filter    string    `json:"filter"`
	Fuzziness string    `json:"fuzziness"`
	Sort      string    `json:"sort"`
	Facets    string    `json:"facets"`
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	name := r.PathValue("name")
	
	query := r.URL.Query().Get("q")
	filterRaw := r.URL.Query().Get("filter")
	fuzziness := r.URL.Query().Get("fuzziness")
	sortParam := r.URL.Query().Get("sort")
	facetsParam := r.URL.Query().Get("facets")
	limit := 20
	offset := 0
	var vector []float64

	if l, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && l > 0 {
		limit = l
	}
	if o, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil && o >= 0 {
		offset = o
	}
	vecStr := r.URL.Query().Get("vector")
	if vecStr != "" {
		parts := strings.Split(vecStr, ",")
		for _, p := range parts {
			if f, err := strconv.ParseFloat(strings.TrimSpace(p), 64); err == nil {
				vector = append(vector, f)
			}
		}
	}

	if r.Method == http.MethodPost {
		var req SearchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			if req.Q != "" { query = req.Q }
			if req.Filter != "" { filterRaw = req.Filter }
			if req.Fuzziness != "" { fuzziness = req.Fuzziness }
			if req.Sort != "" { sortParam = req.Sort }
			if req.Facets != "" { facetsParam = req.Facets }
			if req.Limit > 0 { limit = req.Limit }
			if req.Offset >= 0 { offset = req.Offset }
			if len(req.Vector) > 0 { vector = req.Vector }
		}
	}

	if fuzziness != "" && fuzziness != "0" && fuzziness != "1" && fuzziness != "2" && fuzziness != "AUTO" {
		sendError(w, http.StatusBadRequest, "Invalid fuzziness value. Allowed: 0, 1, 2, AUTO")
		return
	}

	if limit > 1000 {
		limit = 1000
	}

	idx, err := s.mgr.GetIndex(name)
	if err == manager.ErrIndexNotFound {
		sendError(w, http.StatusNotFound, "Index not found")
		return
	}

	if query == "" && len(vector) == 0 {
		sendJSON(w, http.StatusOK, map[string]interface{}{
			"query":              query,
			"hits":               []interface{}{},
			"total_hits":         0,
			"facets":             make(map[string]map[string]int),
			"limit":              limit,
			"offset":             offset,
			"processing_time_ms": time.Since(start).Milliseconds(),
		})
		return
	}

	// Check cache
	vecKey := ""
	if len(vector) > 0 {
		vecKey = fmt.Sprintf("%v", vector)
	}
	cacheKey := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%d|%d|%s", name, query, filterRaw, fuzziness, sortParam, facetsParam, limit, offset, vecKey)
	if cached, ok := s.cache.Get(cacheKey); ok {
		if resp, ok := cached.(map[string]interface{}); ok {
			resp["processing_time_ms"] = time.Since(start).Milliseconds()
			sendJSON(w, http.StatusOK, resp)
			return
		}
	}

	docIDs, highlights := idx.Engine.Search(query, idx.Settings, fuzziness, vector)

	// Apply filters before pagination
	filters := engine.ParseFilters(filterRaw)
	if len(filters) > 0 {
		var filtered []string
		for _, id := range docIDs {
			if doc, ok := idx.Engine.GetDocument(id); ok {
				if engine.ApplyFilters(doc, filters) {
					filtered = append(filtered, id)
				}
			}
		}
		docIDs = filtered
	}

	facetsResult := make(map[string]map[string]int)
	if facetsParam != "" {
		facetFields := strings.Split(facetsParam, ",")
		for _, f := range facetFields {
			facetsResult[f] = make(map[string]int)
		}
		
		for _, id := range docIDs {
			if doc, ok := idx.Engine.GetDocument(id); ok {
				for _, f := range facetFields {
					if val, ok := doc[f]; ok {
						if strVal, isStr := val.(string); isStr {
							facetsResult[f][strVal]++
						} else {
							facetsResult[f][fmt.Sprintf("%v", val)]++
						}
					}
				}
			}
		}
	}

	if sortParam != "" {
		sortRules := strings.Split(sortParam, ",")

		docCache := make(map[string]map[string]interface{}, len(docIDs))
		for _, id := range docIDs {
			if doc, ok := idx.Engine.GetDocument(id); ok {
				docCache[id] = doc
			}
		}

		sort.SliceStable(docIDs, func(i, j int) bool {
			docI := docCache[docIDs[i]]
			docJ := docCache[docIDs[j]]

			for _, rule := range sortRules {
				parts := strings.SplitN(rule, ":", 2)
				field := parts[0]
				dir := "asc"
				if len(parts) > 1 {
					dir = strings.ToLower(parts[1])
				}

				valI, okI := docI[field]
				valJ, okJ := docJ[field]

				if okI && okJ {
					if numI, isNumI := valI.(float64); isNumI {
						if numJ, isNumJ := valJ.(float64); isNumJ {
							if numI != numJ {
								if dir == "desc" {
									return numI > numJ
								}
								return numI < numJ
							}
							continue
						}
					}
					strI := fmt.Sprintf("%v", valI)
					strJ := fmt.Sprintf("%v", valJ)
					if strI != strJ {
						if dir == "desc" {
							return strI > strJ
						}
						return strI < strJ
					}
				} else if okI && !okJ {
					return dir == "desc"
				} else if !okI && okJ {
					return dir != "desc"
				}
			}
			return false
		})
	}

	totalHits := len(docIDs)

	// Apply pagination
	if offset >= len(docIDs) {
		docIDs = nil
	} else {
		end := offset + limit
		if end > len(docIDs) {
			end = len(docIDs)
		}
		docIDs = docIDs[offset:end]
	}

	hits := []map[string]interface{}{}
	for _, id := range docIDs {
		if doc, ok := idx.Engine.GetDocument(id); ok {
			displayDoc := make(map[string]interface{})
			if len(idx.Settings.DisplayedFields) > 0 {
				for _, f := range idx.Settings.DisplayedFields {
					if val, ok := doc[f]; ok {
						displayDoc[f] = val
					}
				}
				if _, ok := displayDoc["id"]; !ok {
					displayDoc["id"] = id
				}
			} else {
				displayDoc = doc
			}

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

	resp := map[string]interface{}{
		"query":              query,
		"hits":               hits,
		"total_hits":         totalHits,
		"facets":             facetsResult,
		"limit":              limit,
		"offset":             offset,
		"processing_time_ms": elapsed.Milliseconds(),
	}

	s.cache.Put(cacheKey, resp)
	sendJSON(w, http.StatusOK, resp)
}
