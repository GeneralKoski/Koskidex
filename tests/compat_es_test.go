package tests

import (
	"encoding/json"
	"testing"

	"github.com/GeneralKoski/Koskidex/internal/server"
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

// Un risultato che non dice quale versione del motore l'ha prodotto non si
// rifà: /health la riporta, e chi misura la salva accanto ai numeri.
func TestHealthReportsTheVersion(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	server.Version = "1902826"
	defer func() { server.Version = "dev" }()

	var h map[string]interface{}
	_ = json.NewDecoder(richiesta(t, srv, "GET", "/health", "").Body).Decode(&h)
	if h["version"] != "1902826" {
		t.Fatalf("/health deve riportare la versione: %v", h)
	}
}

// Una specifica di minimum_should_match sbagliata si rifiuta quando arriva, e
// l'indice tiene le impostazioni che aveva: altrimenti la ricerca dovrebbe
// scegliere da sola cosa farne.
func TestAMalformedMinimumShouldMatchIsRefused(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	richiesta(t, srv, "POST", "/indexes", `{"name":"documents"}`)
	if w := richiesta(t, srv, "PUT", "/indexes/documents/settings", `{"minimum_should_match": "2<-25% 9<-3"}`); w.Code != 200 {
		t.Fatalf("una specifica valida va accettata: %d %s", w.Code, w.Body.String())
	}
	if w := richiesta(t, srv, "PUT", "/indexes/documents/settings", `{"minimum_should_match": "tre quarti"}`); w.Code != 400 {
		t.Fatalf("una specifica sbagliata va rifiutata con 400: %d %s", w.Code, w.Body.String())
	}
	var s map[string]interface{}
	_ = json.NewDecoder(richiesta(t, srv, "GET", "/indexes/documents/settings", "").Body).Decode(&s)
	if s["minimum_should_match"] != "2<-25% 9<-3" {
		t.Fatalf("dopo il rifiuto restano le impostazioni di prima: %v", s["minimum_should_match"])
	}
}
