package embedder

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/GeneralKoski/Koskidex/internal/engine"
)

// ollamaFinto answers /api/embed as Ollama does, with a vector that depends on
// the text, and counts the requests and the texts.
func ollamaFinto(t *testing.T, richieste, testi *int64) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/embed" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		var req struct {
			Model string   `json:"model"`
			Input []string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Model != "bge-m3" {
			http.Error(w, `{"error":"model not found"}`, http.StatusNotFound)
			return
		}
		atomic.AddInt64(richieste, 1)
		atomic.AddInt64(testi, int64(len(req.Input)))
		out := make([][]float64, len(req.Input))
		for i, s := range req.Input {
			out[i] = []float64{float64(len(s)), float64(strings.Count(s, "a"))}
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"model": req.Model, "embeddings": out})
	}))
}

func impostazioni(url string) engine.EmbedderSettings {
	return engine.EmbedderSettings{Source: SourceOllama, Model: "bge-m3", URL: url}
}

func TestOllamaEmbedsInBlocksAndKeepsTheOrder(t *testing.T) {
	var richieste, testi int64
	srv := ollamaFinto(t, &richieste, &testi)
	defer srv.Close()
	e, err := New(impostazioni(srv.URL), srv.Client())
	if err != nil {
		t.Fatal(err)
	}

	in := make([]string, 130)
	for i := range in {
		in[i] = strings.Repeat("a", i+1)
	}
	v, err := e.Embed(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if len(v) != 130 || v[0][0] != 1 || v[129][0] != 130 {
		t.Fatalf("vettori fuori ordine: %d, primo %v, ultimo %v", len(v), v[0], v[len(v)-1])
	}
	if richieste != 3 {
		t.Fatalf("attese 3 richieste da 64, ottenute %d", richieste)
	}
}

func TestOllamaErrorsAreReported(t *testing.T) {
	var r, n int64
	srv := ollamaFinto(t, &r, &n)
	defer srv.Close()
	e, _ := New(engine.EmbedderSettings{Source: SourceOllama, Model: "inesistente", URL: srv.URL}, srv.Client())
	if _, err := e.Embed(context.Background(), []string{"x"}); err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("atteso un errore con il 404 di Ollama, ottenuto %v", err)
	}
	srv.Close()
	e, _ = New(impostazioni(srv.URL), http.DefaultClient)
	if _, err := e.Embed(context.Background(), []string{"x"}); err == nil {
		t.Fatal("con Ollama spento serve un errore")
	}
}

func TestSettingsAreValidated(t *testing.T) {
	casi := map[string]engine.EmbedderSettings{
		"sorgente sconosciuta": {Source: "openai", Model: "x"},
		"modello mancante":     {Source: SourceOllama},
	}
	for nome, s := range casi {
		if Valida(s) == nil {
			t.Errorf("%s: attesa una validazione fallita", nome)
		}
	}
	if Valida(engine.EmbedderSettings{}) != nil || Valida(impostazioni("")) != nil {
		t.Fatal("vuoto e ollama con modello vanno accettati")
	}
	e, _ := New(impostazioni(""), nil)
	if e.(*ollama).url != "http://localhost:11434" {
		t.Fatalf("URL predefinito sbagliato: %s", e.(*ollama).url)
	}
}

func TestCacheCallsTheModelOncePerText(t *testing.T) {
	var richieste, testi int64
	srv := ollamaFinto(t, &richieste, &testi)
	defer srv.Close()
	e, _ := New(impostazioni(srv.URL), srv.Client())
	c := NewCache()

	v, err := c.Embed(context.Background(), e, []string{"casa", "albo", "casa"})
	if err != nil {
		t.Fatal(err)
	}
	if testi != 2 || v[0][1] != 2 || v[1][1] != 1 || v[2][1] != 2 {
		t.Fatalf("attesi 2 testi al modello e i vettori al loro posto: %d, %v", testi, v)
	}
	if _, err := c.Embed(context.Background(), e, []string{"albo", "casa"}); err != nil {
		t.Fatal(err)
	}
	if richieste != 1 {
		t.Fatalf("i testi già visti non devono tornare al modello: %d richieste", richieste)
	}

	// Un altro modello, un'altra chiave.
	altro := &ollama{url: srv.URL, model: "altro", client: srv.Client()}
	if chiave(altro, "casa") == chiave(e, "casa") {
		t.Fatal("due modelli non devono condividere i vettori")
	}
}

func TestCacheFileSurvivesARestartAndABrokenLine(t *testing.T) {
	var richieste, testi int64
	srv := ollamaFinto(t, &richieste, &testi)
	defer srv.Close()
	e, _ := New(impostazioni(srv.URL), srv.Client())
	path := filepath.Join(t.TempDir(), "embeddings.jsonl")

	c, err := OpenCache(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Embed(context.Background(), e, []string{"delibera", "determina"}); err != nil {
		t.Fatal(err)
	}
	_ = c.Close()

	// Una scrittura interrotta a metà.
	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	_, _ = f.WriteString(`{"k":"abc","v":[1,`)
	_ = f.Close()

	c, err = OpenCache(path)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if c.Len() != 2 {
		t.Fatalf("attesi 2 vettori riletti, trovati %d", c.Len())
	}
	v, err := c.Embed(context.Background(), e, []string{"determina"})
	if err != nil || v[0][0] != 9 {
		t.Fatalf("vettore riletto sbagliato: %v %v", v, err)
	}
	if richieste != 1 {
		t.Fatalf("dopo il riavvio il modello non va richiamato: %d richieste", richieste)
	}
}

func TestDocumentText(t *testing.T) {
	doc := map[string]interface{}{
		"id": "7", "name": "Determina 1223", "tags": []interface{}{"noleggio", "veicoli"},
		"notes": "", "anno": 2026.0, "_vector": []interface{}{1.0},
	}
	if got := TestoDocumento(doc, []string{"name", "tags", "notes", "manca"}); got != "Determina 1223\nnoleggio, veicoli" {
		t.Fatalf("testo con i campi: %q", got)
	}
	if got := TestoDocumento(doc, nil); got != "Determina 1223\nnoleggio, veicoli" {
		t.Fatalf("testo senza campi: %q", got)
	}
	if got := TestoDocumento(map[string]interface{}{"n": []interface{}{2026.0, 3.5}}, []string{"n"}); got != "2026, 3.5" {
		t.Fatalf("numeri: %q", got)
	}
}
