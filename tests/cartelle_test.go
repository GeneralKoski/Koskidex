package tests

import (
	"net/http"
	"testing"
)

// L'indice delle cartelle di Documentale, configurato via HTTP come lo
// configurerà il servizio: la sottostringa si accende con substring_match.
func TestSubstringMatchIsASettingOverHTTP(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	richiesta(t, srv, "POST", "/indexes", `{"name":"folders"}`)
	richiesta(t, srv, "POST", "/indexes/folders/documents", `[{"id": 1, "path": "Amministrazione/Delibere 2024"}]`)

	cerca := func() int {
		return len(searchV2(t, srv, "/indexes/folders/search?q=liber")["hits"].([]interface{}))
	}
	if n := cerca(); n != 0 {
		t.Fatalf("prima di accenderla, \"liber\" non deve trovare niente: %d", n)
	}

	w := richiesta(t, srv, "PUT", "/indexes/folders/settings",
		`{"searchable_fields": ["path"], "retrieval_mode": "any", "substring_match": true}`)
	if w.Code != http.StatusOK {
		t.Fatalf("aggiornamento impostazioni fallito: %d %s", w.Code, w.Body.String())
	}
	if n := cerca(); n != 1 {
		t.Fatalf("con substring_match \"liber\" deve trovare la cartella delle delibere: %d", n)
	}
}
