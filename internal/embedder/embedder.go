// Package embedder turns text into vectors with a model server, so that the
// engine gets a document's _vector and a query's vector without the client
// computing them. Only Ollama for now: a local model keeps every number
// reproducible and the documents on the machine.
package embedder

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/GeneralKoski/Koskidex/internal/engine"
)

const (
	SourceOllama     = "ollama"
	defaultOllamaURL = "http://localhost:11434"
	// bloccoOllama is how many texts go in one request.
	bloccoOllama = 64
)

// Embedder turns texts into vectors, one per text, in the same order.
type Embedder interface {
	Embed(ctx context.Context, testi []string) ([][]float64, error)
	// Nome identifies the model, for the cache: vectors of two models must
	// never mix.
	Nome() string
}

// Valida checks the settings the API receives.
func Valida(s engine.EmbedderSettings) error {
	switch s.Source {
	case "":
		return nil
	case SourceOllama:
		if strings.TrimSpace(s.Model) == "" {
			return errors.New("embedder: model is required")
		}
		return nil
	}
	return fmt.Errorf("embedder: unknown source %q, use %q", s.Source, SourceOllama)
}

// New returns the embedder the settings name.
func New(s engine.EmbedderSettings, client *http.Client) (Embedder, error) {
	if err := Valida(s); err != nil {
		return nil, err
	}
	if s.Source == "" {
		return nil, errors.New("embedder: none configured")
	}
	url := strings.TrimRight(s.URL, "/")
	if url == "" {
		url = defaultOllamaURL
	}
	return &ollama{url: url, model: s.Model, client: client}, nil
}

type ollama struct {
	url, model string
	client     *http.Client
}

func (o *ollama) Nome() string { return SourceOllama + "/" + o.model }

// Embed calls /api/embed, a block of texts at a time. Ollama returns the
// vectors already normalised.
func (o *ollama) Embed(ctx context.Context, testi []string) ([][]float64, error) {
	out := make([][]float64, 0, len(testi))
	for i := 0; i < len(testi); i += bloccoOllama {
		blocco := testi[i:min(i+bloccoOllama, len(testi))]
		corpo, err := json.Marshal(map[string]interface{}{"model": o.model, "input": blocco})
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.url+"/api/embed", bytes.NewReader(corpo))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := o.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("embedder: %w", err)
		}
		dati, err := io.ReadAll(io.LimitReader(resp.Body, 256<<20))
		_ = resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("embedder: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("embedder: ollama answered %d: %s", resp.StatusCode, estratto(dati))
		}
		var r struct {
			Embeddings [][]float64 `json:"embeddings"`
		}
		if err := json.Unmarshal(dati, &r); err != nil {
			return nil, fmt.Errorf("embedder: unreadable ollama answer: %w", err)
		}
		if len(r.Embeddings) != len(blocco) {
			return nil, fmt.Errorf("embedder: %d vectors for %d texts", len(r.Embeddings), len(blocco))
		}
		out = append(out, r.Embeddings...)
	}
	return out, nil
}

func estratto(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 200 {
		s = s[:200] + "..."
	}
	return s
}
