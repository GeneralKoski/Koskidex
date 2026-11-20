package tests

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func impostazioniDi(t *testing.T, w interface{ Bytes() []byte }) map[string]interface{} {
	t.Helper()
	var s map[string]interface{}
	if err := json.Unmarshal(w.Bytes(), &s); err != nil {
		t.Fatalf("impostazioni illeggibili: %v", err)
	}
	return s
}

func TestPatchSettingsKeepsTheOthers(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	richiesta(t, srv, "POST", "/indexes", `{"name": "p"}`)
	if w := richiesta(t, srv, "PUT", "/indexes/p/settings", `{"searchable_fields": ["title"], "stop_words": {"the": true}, "hybrid_mode": "union"}`); w.Code != http.StatusOK {
		t.Fatalf("PUT: %d %s", w.Code, w.Body.String())
	}

	if w := richiesta(t, srv, "PATCH", "/indexes/p/settings", `{"vector_min_similarity": 0.55, "field_weights": {"title": 5}}`); w.Code != http.StatusOK {
		t.Fatalf("PATCH: %d %s", w.Code, w.Body.String())
	}
	s := impostazioniDi(t, richiesta(t, srv, "GET", "/indexes/p/settings", "").Body)
	if s["vector_min_similarity"] != 0.55 || s["hybrid_mode"] != "union" {
		t.Fatalf("PATCH deve cambiare solo le chiavi inviate: %v", s)
	}
	if sw, _ := s["stop_words"].(map[string]interface{}); sw["the"] != true {
		t.Fatalf("stop_words perse dal PATCH: %v", s["stop_words"])
	}
	if f, _ := s["searchable_fields"].([]interface{}); len(f) != 1 || f[0] != "title" {
		t.Fatalf("searchable_fields persi dal PATCH: %v", s["searchable_fields"])
	}

	if w := richiesta(t, srv, "PATCH", "/indexes/p/settings", `{"hybrid_mode": null}`); w.Code != http.StatusOK {
		t.Fatalf("PATCH null: %d", w.Code)
	}
	if s := impostazioniDi(t, richiesta(t, srv, "GET", "/indexes/p/settings", "").Body); s["hybrid_mode"] != "" || s["vector_min_similarity"] != 0.55 {
		t.Fatalf("null deve azzerare solo hybrid_mode: %v", s)
	}

	// PUT resta una sostituzione intera.
	richiesta(t, srv, "PUT", "/indexes/p/settings", `{"searchable_fields": ["title"]}`)
	if s := impostazioniDi(t, richiesta(t, srv, "GET", "/indexes/p/settings", "").Body); s["vector_min_similarity"] != 0.0 || s["stop_words"] != nil && len(s["stop_words"].(map[string]interface{})) != 0 {
		t.Fatalf("PUT deve sostituire tutto: %v", s)
	}
}

func TestPatchSettingsRefusesBadInput(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	richiesta(t, srv, "POST", "/indexes", `{"name": "p"}`)
	richiesta(t, srv, "PUT", "/indexes/p/settings", `{"searchable_fields": ["title"]}`)

	for _, corpo := range []string{
		`[1, 2]`,
		`{"stop_words": ["the"]}`,
		`{"minimum_should_match": "tre quarti"}`,
	} {
		if w := richiesta(t, srv, "PATCH", "/indexes/p/settings", corpo); w.Code != http.StatusBadRequest {
			t.Fatalf("%s: atteso 400, ottenuto %d %s", corpo, w.Code, w.Body.String())
		}
	}
	if f, _ := impostazioniDi(t, richiesta(t, srv, "GET", "/indexes/p/settings", "").Body)["searchable_fields"].([]interface{}); len(f) != 1 {
		t.Fatalf("un PATCH rifiutato non deve cambiare niente: %v", f)
	}
	if w := richiesta(t, srv, "PATCH", "/indexes/manca/settings", `{}`); w.Code != http.StatusNotFound {
		t.Fatalf("indice inesistente: atteso 404, ottenuto %d", w.Code)
	}
}

func TestStopWordsAsMapAreAccepted(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	richiesta(t, srv, "POST", "/indexes", `{"name": "s"}`)
	if w := richiesta(t, srv, "PUT", "/indexes/s/settings", `{"stop_words": {"the": true, "a": true, "is": true}}`); w.Code != http.StatusOK {
		t.Fatalf("stop_words come nel README: %d %s", w.Code, w.Body.String())
	}
}

func TestSettingsErrorNamesTheField(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	richiesta(t, srv, "POST", "/indexes", `{"name": "e"}`)

	for _, metodo := range []string{"PUT", "PATCH"} {
		w := richiesta(t, srv, metodo, "/indexes/e/settings", `{"stop_words": ["the"]}`)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("%s: atteso 400, ottenuto %d", metodo, w.Code)
		}
		var corpo map[string]string
		if err := json.Unmarshal(w.Body.Bytes(), &corpo); err != nil {
			t.Fatalf("%s: risposta illeggibile: %s", metodo, w.Body.String())
		}
		if !strings.Contains(corpo["error"], "stop_words") || !strings.Contains(corpo["error"], "object") {
			t.Fatalf("%s: l'errore deve dire quale campo e cosa si aspetta: %q", metodo, corpo["error"])
		}
	}
}
