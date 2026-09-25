package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/GeneralKoski/Koskidex/internal/server"
)

// Un Ollama finto: il vettore di un testo è [lunghezza, numero di "a"], e ogni
// testo ricevuto si conta.
func ollamaFintoAPI(t *testing.T, testi *int64) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Input []string `json:"input"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		atomic.AddInt64(testi, int64(len(req.Input)))
		out := make([][]float64, len(req.Input))
		for i, s := range req.Input {
			out[i] = []float64{float64(len(s)), float64(strings.Count(s, "a"))}
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"embeddings": out})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func chiedi(t *testing.T, srv *server.Server, metodo, path, corpo string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(metodo, path, bytes.NewBufferString(corpo))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	return w
}

func indiceConEmbedder(t *testing.T, srv *server.Server, url string) {
	t.Helper()
	if w := chiedi(t, srv, "POST", "/indexes", `{"name":"atti"}`); w.Code != http.StatusCreated {
		t.Fatalf("creazione: %d %s", w.Code, w.Body)
	}
	impostazioni := fmt.Sprintf(`{"searchable_fields":["name"],"ranking_rules":["exactness"],
		"embedder":{"source":"ollama","model":"bge-m3","url":%q}}`, url)
	if w := chiedi(t, srv, "PUT", "/indexes/atti/settings", impostazioni); w.Code != http.StatusOK {
		t.Fatalf("impostazioni: %d %s", w.Code, w.Body)
	}
}

func idsDi(t *testing.T, w *httptest.ResponseRecorder) []string {
	t.Helper()
	var r struct {
		Hits []struct {
			ID string `json:"id"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &r); err != nil {
		t.Fatalf("risposta illeggibile: %s", w.Body)
	}
	var ids []string
	for _, h := range r.Hits {
		ids = append(ids, h.ID)
	}
	return ids
}

func TestTheEmbedderGivesDocumentsTheirVector(t *testing.T) {
	var testi int64
	ollama := ollamaFintoAPI(t, &testi)
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	indiceConEmbedder(t, srv, ollama.URL)

	w := chiedi(t, srv, "POST", "/indexes/atti/documents",
		`[{"id":"1","name":"noleggio auto"},{"id":"2","name":"altro","_vector":[9,9]},{"id":"3","name":""}]`)
	if w.Code != http.StatusAccepted {
		t.Fatalf("aggiunta: %d %s", w.Code, w.Body)
	}
	vettore := func(id string) []interface{} {
		var d map[string]interface{}
		_ = json.Unmarshal(chiedi(t, srv, "GET", "/indexes/atti/documents/"+id, "").Body.Bytes(), &d)
		if doc, ok := d["document"].(map[string]interface{}); ok {
			d = doc
		}
		v, _ := d["_vector"].([]interface{})
		return v
	}
	if v := vettore("1"); len(v) != 2 || v[0].(float64) != 13 {
		t.Fatalf("il documento 1 doveva avere il vettore di \"noleggio auto\": %v", v)
	}
	if v := vettore("2"); len(v) != 2 || v[0].(float64) != 9 {
		t.Fatalf("un vettore mandato dal client non si ricalcola: %v", v)
	}
	if v := vettore("3"); v != nil {
		t.Fatalf("un documento senza testo non ha vettore: %v", v)
	}
	if testi != 1 {
		t.Fatalf("al modello doveva arrivare un solo testo, ne sono arrivati %d", testi)
	}
}

// Due documenti che il lessicale non distingue: con hybrid decide il vettore
// della query, calcolato dal server.
func TestHybridSearchEmbedsTheQuery(t *testing.T) {
	var testi int64
	ollama := ollamaFintoAPI(t, &testi)
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	indiceConEmbedder(t, srv, ollama.URL)
	chiedi(t, srv, "POST", "/indexes/atti/documents",
		`[{"id":"1","name":"noleggio aaaaaaa"},{"id":"2","name":"noleggio auto"}]`)

	lessicale := idsDi(t, chiedi(t, srv, "GET", "/indexes/atti/search?q=noleggio", ""))
	ibrida := idsDi(t, chiedi(t, srv, "GET", "/indexes/atti/search?q=noleggio&hybrid=true", ""))
	if len(ibrida) != 2 || ibrida[0] != "2" {
		t.Fatalf("con hybrid il documento 2 (vettore più vicino a quello della query) va primo: %v", ibrida)
	}
	if lessicale[0] != "1" {
		t.Fatalf("senza hybrid l'ordine è quello di sempre, e il test va rivisto: %v", lessicale)
	}
	prima := testi
	chiedi(t, srv, "POST", "/indexes/atti/search", `{"q":"noleggio","hybrid":true,"limit":5}`)
	if testi != prima {
		t.Fatalf("il vettore di una query già vista viene dalla cache: %d testi in più", testi-prima)
	}
}

func TestHybridWithoutAnEmbedderIsRefused(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	chiedi(t, srv, "POST", "/indexes", `{"name":"atti"}`)
	if w := chiedi(t, srv, "GET", "/indexes/atti/search?q=noleggio&hybrid=true", ""); w.Code != http.StatusBadRequest {
		t.Fatalf("atteso 400, ottenuto %d %s", w.Code, w.Body)
	}
}

func TestAnUnreachableModelAddsNothing(t *testing.T) {
	var testi int64
	ollama := ollamaFintoAPI(t, &testi)
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	indiceConEmbedder(t, srv, ollama.URL)
	ollama.Close()

	if w := chiedi(t, srv, "POST", "/indexes/atti/documents", `[{"id":"1","name":"noleggio"}]`); w.Code != http.StatusBadGateway {
		t.Fatalf("atteso 502, ottenuto %d %s", w.Code, w.Body)
	}
	var info map[string]interface{}
	_ = json.Unmarshal(chiedi(t, srv, "GET", "/indexes/atti", "").Body.Bytes(), &info)
	if info["docs"].(float64) != 0 {
		t.Fatalf("senza vettori non entra niente: %v documenti", info["docs"])
	}
	if w := chiedi(t, srv, "GET", "/indexes/atti/search?q=noleggio&hybrid=true", ""); w.Code != http.StatusBadGateway {
		t.Fatalf("ricerca ibrida col modello spento: atteso 502, ottenuto %d", w.Code)
	}
}

func TestAMalformedEmbedderIsRefused(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	chiedi(t, srv, "POST", "/indexes", `{"name":"atti"}`)
	for _, e := range []string{`{"source":"openai","model":"x"}`, `{"source":"ollama"}`} {
		if w := chiedi(t, srv, "PUT", "/indexes/atti/settings", `{"embedder":`+e+`}`); w.Code != http.StatusBadRequest {
			t.Errorf("%s: atteso 400, ottenuto %d", e, w.Code)
		}
	}
}

// Senza embedder nelle impostazioni, il modello non si chiama mai.
func TestWithoutAnEmbedderNothingIsCalled(t *testing.T) {
	var testi int64
	ollamaFintoAPI(t, &testi)
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	chiedi(t, srv, "POST", "/indexes", `{"name":"atti"}`)
	chiedi(t, srv, "POST", "/indexes/atti/documents", `[{"id":"1","name":"noleggio"}]`)
	var d map[string]interface{}
	_ = json.Unmarshal(chiedi(t, srv, "GET", "/indexes/atti/documents/1", "").Body.Bytes(), &d)
	if strings.Contains(fmt.Sprint(d), "_vector") || testi != 0 {
		t.Fatalf("niente vettori e niente chiamate senza embedder: %v, %d testi", d, testi)
	}
}
