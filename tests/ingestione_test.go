package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GeneralKoski/Koskidex/internal/server"
)

func richiesta(t *testing.T, srv *server.Server, metodo, url, corpo string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(metodo, url, bytes.NewBufferString(corpo))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	return w
}

func documentiNellIndice(t *testing.T, srv *server.Server, indice string) float64 {
	t.Helper()
	w := richiesta(t, srv, "GET", "/indexes/"+indice, "")
	var r map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&r)
	return r["docs"].(float64)
}

// Documentale identifica i documenti con Document::id, un intero. Prima veniva
// scartato in silenzio: risposta 202 "Documents added" con skipped 1.
func TestAnIntegerIDIsAcceptedAsItsCanonicalString(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	richiesta(t, srv, "POST", "/indexes", `{"name":"documents"}`)

	w := richiesta(t, srv, "POST", "/indexes/documents/documents", `[{"id": 7, "title": "Fattura manutenzione"}]`)
	if w.Code != http.StatusAccepted {
		t.Fatalf("atteso 202, ottenuto %d: %s", w.Code, w.Body.String())
	}
	var r map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&r)
	if r["added"].(float64) != 1 {
		t.Fatalf("il documento con id intero deve essere aggiunto: %v", r)
	}

	// "7", non "7.0": JSON decodifica i numeri come float64.
	if w := richiesta(t, srv, "GET", "/indexes/documents/documents/7", ""); w.Code != http.StatusOK {
		t.Fatalf("il documento deve rispondere all'id 7: HTTP %d", w.Code)
	}
	res := searchV2(t, srv, "/indexes/documents/search?q=fattura")
	if res["total_hits"].(float64) != 1 {
		t.Fatalf("il documento deve essere cercabile: %v", res)
	}
}

// Un id che non si puo' usare deve fermare la richiesta con un errore che dice
// dove, e non deve entrare niente: meta' blocco caricato e meta' no e' il modo
// piu' difficile da scoprire di perdere dati.
func TestAnInvalidIDRejectsTheWholeRequest(t *testing.T) {
	casi := map[string]string{
		"id mancante":   `[{"id": 1, "title": "a"}, {"title": "senza id"}]`,
		"id vuoto":      `[{"id": 1, "title": "a"}, {"id": "", "title": "b"}]`,
		"id non intero": `[{"id": 1, "title": "a"}, {"id": 7.5, "title": "b"}]`,
		"id oggetto":    `[{"id": 1, "title": "a"}, {"id": {"x": 1}, "title": "b"}]`,
		"id booleano":   `[{"id": 1, "title": "a"}, {"id": true, "title": "b"}]`,
	}
	for nome, corpo := range casi {
		t.Run(nome, func(t *testing.T) {
			srv, cleanup := setupTestServer(t)
			defer cleanup()
			richiesta(t, srv, "POST", "/indexes", `{"name":"documents"}`)

			w := richiesta(t, srv, "POST", "/indexes/documents/documents", corpo)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("atteso 400, ottenuto %d: %s", w.Code, w.Body.String())
			}
			if !strings.Contains(w.Body.String(), "posizione 1") {
				t.Fatalf("l'errore deve indicare la posizione del documento sbagliato: %s", w.Body.String())
			}
			if n := documentiNellIndice(t, srv, "documents"); n != 0 {
				t.Fatalf("con un id non valido non deve entrare niente, invece ci sono %v documenti", n)
			}
		})
	}
}

// Lo scenario provato dal vivo il 24/09/2026, con i campi come li manda
// Documentale: "enel" sta solo in subjects, e dava zero risultati.
func TestDocumentaleListFieldsAreSearchableOverHTTP(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	richiesta(t, srv, "POST", "/indexes", `{"name":"documents"}`)
	richiesta(t, srv, "PUT", "/indexes/documents/settings",
		`{"searchable_fields":["name","tags","summary","subjects","notes","additional_data"],
		  "field_weights":{"name":5,"tags":4,"summary":3,"subjects":3,"notes":2,"additional_data":1}}`)
	w := richiesta(t, srv, "POST", "/indexes/documents/documents", `[
		{"id": 8, "name": "Contratto fornitura", "tags": ["contratto"], "summary": "Fornitura energia elettrica",
		 "subjects": ["Enel Energia"], "notes": "", "additional_data": ["Ufficio Acquisti", 1223]}]`)
	if w.Code != http.StatusAccepted {
		t.Fatalf("atteso 202, ottenuto %d: %s", w.Code, w.Body.String())
	}

	for _, q := range []string{"enel", "acquisti", "1223", "contratto"} {
		if res := searchV2(t, srv, "/indexes/documents/search?q="+q+"&fuzziness=0"); res["total_hits"].(float64) != 1 {
			t.Errorf("%q va trovata: %v", q, res["total_hits"])
		}
	}
}
