package tests

import (
	"net/http"
	"testing"
)

// L'indice dei prodotti e delle zone di Korsia, configurato via HTTP come lo
// configurerà l'app.
func TestSynonymGroupsAreASettingOverHTTP(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	richiesta(t, srv, "POST", "/indexes", `{"name":"korsia"}`)
	richiesta(t, srv, "POST", "/indexes/korsia/documents", `[
		{"id": "1", "name": "Coca-Cola", "category_label": "Bevande"},
		{"id": "2", "name": "Soft drink al limone", "category_label": "Bibite"}]`)

	w := richiesta(t, srv, "PUT", "/indexes/korsia/settings",
		`{"searchable_fields": ["name", "category_label"], "synonym_groups": [["bibite", "bevande", "soft drink"]]}`)
	if w.Code != http.StatusOK {
		t.Fatalf("aggiornamento impostazioni fallito: %d %s", w.Code, w.Body.String())
	}
	for _, q := range []string{"bibite", "bevande", "soft%20drink"} {
		if n := len(searchV2(t, srv, "/indexes/korsia/search?q="+q)["hits"].([]interface{})); n != 2 {
			t.Errorf("%s: %d risultati, ne servono 2", q, n)
		}
	}
}

func TestSynonymFieldsAreASettingOverHTTP(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	richiesta(t, srv, "POST", "/indexes", `{"name":"korsia"}`)
	richiesta(t, srv, "POST", "/indexes/korsia/documents", `[
		{"id": "1", "name": "Lambrusco", "category_label": "Vini", "zone_names": "Bevande e vini"},
		{"id": "2", "name": "Coca-Cola", "category_label": "Bevande", "zone_names": "Bevande e vini"}]`)
	w := richiesta(t, srv, "PUT", "/indexes/korsia/settings", `{
		"searchable_fields": ["name", "category_label", "zone_names"],
		"synonym_groups": [["bibite", "bevande"]],
		"synonym_fields": ["name", "category_label"]}`)
	if w.Code != http.StatusOK {
		t.Fatalf("aggiornamento impostazioni fallito: %d %s", w.Code, w.Body.String())
	}
	hits := searchV2(t, srv, "/indexes/korsia/search?q=bibite")["hits"].([]interface{})
	if len(hits) != 1 || hits[0].(map[string]interface{})["id"] != "2" {
		t.Fatalf("\"bibite\" deve trovare solo la Coca-Cola, non il Lambrusco: %v", hits)
	}
}

func TestBoostsAreASettingOverHTTP(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	richiesta(t, srv, "POST", "/indexes", `{"name":"korsia"}`)
	richiesta(t, srv, "POST", "/indexes/korsia/documents", `[
		{"id": "1", "kind": "product", "name": "Pepe nero"},
		{"id": "2", "kind": "zone", "name": "Pepe"}]`)
	if w := richiesta(t, srv, "PUT", "/indexes/korsia/settings", `{"boosts": [{"filter": "kind=zone"}]}`); w.Code != http.StatusBadRequest {
		t.Fatalf("un boost senza fattore va rifiutato: %d", w.Code)
	}
	if w := richiesta(t, srv, "PUT", "/indexes/korsia/settings", `{"searchable_fields": ["name"], "boosts": [{"filter": "kind=product", "factor": 2}]}`); w.Code != http.StatusOK {
		t.Fatalf("aggiornamento impostazioni fallito: %d %s", w.Code, w.Body.String())
	}
	hits := searchV2(t, srv, "/indexes/korsia/search?q=pepe")["hits"].([]interface{})
	if hits[0].(map[string]interface{})["id"] != "1" {
		t.Fatalf("il prodotto, raddoppiato, deve venire prima: %v", hits)
	}
}
