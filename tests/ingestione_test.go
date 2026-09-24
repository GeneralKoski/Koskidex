package tests

import (
	"bytes"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strconv"
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

func seminaFatture(t *testing.T, srv *server.Server, n int) {
	t.Helper()
	richiesta(t, srv, "POST", "/indexes", `{"name":"documents"}`)
	var b strings.Builder
	b.WriteString("[")
	for i := 1; i <= n; i++ {
		if i > 1 {
			b.WriteString(",")
		}
		// Un titolo su due ha un refuso, così i punteggi non sono tutti uguali
		// anche con il punteggio predefinito, che guarda i refusi e non la
		// frequenza.
		parola := "fattura"
		if i%2 == 1 {
			parola = "fatura"
		}
		b.WriteString(`{"id": ` + strconv.Itoa(i) + `, "title": "` + parola + ` numero ` + strconv.Itoa(i) + `"}`)
	}
	b.WriteString("]")
	if w := richiesta(t, srv, "POST", "/indexes/documents/documents", b.String()); w.Code != http.StatusAccepted {
		t.Fatalf("semina fallita: %d %s", w.Code, w.Body.String())
	}
}

// Documentale chiede fino a 10.000 risultati e usa l'intero insieme di id per
// filtrare con whereIn. Con il tetto a 1.000 i risultati oltre sparivano.
func TestSearchReturnsMoreThanAThousandResults(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	seminaFatture(t, srv, 1200)

	res := searchV2(t, srv, "/indexes/documents/search?q=fattura&limit=10000&ids_only=true")
	if n := len(res["hits"].([]interface{})); n != 1200 {
		t.Fatalf("attesi 1200 risultati, restituiti %d (total_hits %v)", n, res["total_hits"])
	}
	if res["limit"].(float64) != 10000 {
		t.Fatalf("il limite richiesto va rispettato fino a 10000, è %v", res["limit"])
	}
	if res := searchV2(t, srv, "/indexes/documents/search?q=fattura&limit=50000&ids_only=true"); res["limit"].(float64) != 10000 {
		t.Fatalf("oltre 10000 il limite va fermato a 10000, è %v", res["limit"])
	}
}

// Con ids_only la risposta porta solo id e punteggio: costruire 10.000
// documenti con le evidenziazioni per usarne solo gli id è lavoro buttato.
func TestIDsOnlyReturnsJustIDsAndScores(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	seminaFatture(t, srv, 5)

	for _, url := range []string{"/indexes/documents/search?q=fattura&ids_only=true", ""} {
		var res map[string]interface{}
		if url != "" {
			res = searchV2(t, srv, url)
		} else {
			w := richiesta(t, srv, "POST", "/indexes/documents/search", `{"q": "fattura", "ids_only": true}`)
			_ = json.NewDecoder(w.Body).Decode(&res)
		}
		for _, h := range res["hits"].([]interface{}) {
			hit := h.(map[string]interface{})
			if len(hit) != 2 || hit["id"] == nil || hit["score"] == nil {
				t.Fatalf("con ids_only un risultato deve avere solo id e score: %v", hit)
			}
		}
	}

	// La cache non deve restituire la forma sbagliata: stessa query, senza ids_only.
	res := searchV2(t, srv, "/indexes/documents/search?q=fattura")
	if _, ok := res["hits"].([]interface{})[0].(map[string]interface{})["document"]; !ok {
		t.Fatal("senza ids_only il risultato deve portare il documento")
	}
}

// Il punteggio serve al confronto finale fra i motori, e c'era già: veniva
// calcolato e buttato via nella risposta.
func TestEveryHitCarriesItsScoreInRankingOrder(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	seminaFatture(t, srv, 30)

	res := searchV2(t, srv, "/indexes/documents/search?q=fattura&limit=30")
	precedente := math.Inf(1)
	punteggi := map[string]float64{}
	distinti := map[float64]bool{}
	for _, h := range res["hits"].([]interface{}) {
		hit := h.(map[string]interface{})
		s, ok := hit["score"].(float64)
		if !ok {
			t.Fatalf("ogni risultato deve avere un punteggio numerico: %v", h)
		}
		if s > precedente {
			t.Fatalf("senza sort i risultati devono essere in ordine di punteggio: %v dopo %v", s, precedente)
		}
		precedente = s
		punteggi[hit["id"].(string)] = s
		distinti[s] = true
	}
	if len(distinti) < 2 {
		t.Fatalf("metà dei titoli ha un refuso, i punteggi non possono essere tutti uguali: %v", distinti)
	}

	soliID := searchV2(t, srv, "/indexes/documents/search?q=fattura&limit=30&ids_only=true")
	for _, h := range soliID["hits"].([]interface{}) {
		hit := h.(map[string]interface{})
		if hit["score"].(float64) != punteggi[hit["id"].(string)] {
			t.Fatalf("con ids_only il punteggio di %v deve essere lo stesso della risposta completa: %v invece di %v",
				hit["id"], hit["score"], punteggi[hit["id"].(string)])
		}
	}
}
