package tests

import (
	"encoding/json"
	"testing"
)

// Le impostazioni di compatibilità con Elasticsearch arrivano via HTTP come le
// manderà KoskidexService, e si rileggono uguali.
func TestElasticsearchCompatibilitySettingsOverHTTP(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	richiesta(t, srv, "POST", "/indexes", `{"name":"documents"}`)
	richiesta(t, srv, "PUT", "/indexes/documents/settings",
		`{"disable_prefix_search": true, "prefix_length": 1, "all_terms_in_one_field": true, "tokenizer": "standard"}`)

	var s map[string]interface{}
	_ = json.NewDecoder(richiesta(t, srv, "GET", "/indexes/documents/settings", "").Body).Decode(&s)
	if s["disable_prefix_search"] != true || s["prefix_length"] != float64(1) || s["all_terms_in_one_field"] != true || s["tokenizer"] != "standard" {
		t.Fatalf("le impostazioni devono tornare come sono state scritte: %v", s)
	}

	richiesta(t, srv, "POST", "/indexes/documents/documents", `[{"id": 1, "name": "manutenzione"}]`)
	if n := len(searchV2(t, srv, "/indexes/documents/search?q=manut")["hits"].([]interface{})); n != 0 {
		t.Fatalf("con disable_prefix_search \"manut\" non deve trovare \"manutenzione\": %d risultati", n)
	}
}
