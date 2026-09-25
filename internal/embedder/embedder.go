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
		if s.Context < 0 {
			return errors.New("embedder: context must not be negative")
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
	return &ollama{url: url, model: s.Model, contesto: s.Context, client: client}, nil
}

type ollama struct {
	url, model string
	contesto   int
	client     *http.Client
}

// Nome carries the context when one is set: the same text read up to a
// different length is a different vector.
func (o *ollama) Nome() string {
	if o.contesto > 0 {
		return fmt.Sprintf("%s/%s@%d", SourceOllama, o.model, o.contesto)
	}
	return SourceOllama + "/" + o.model
}

// Embed calls /api/embed, a block of texts at a time. Ollama returns the
// vectors already normalised.
func (o *ollama) Embed(ctx context.Context, testi []string) ([][]float64, error) {
	out := make([][]float64, 0, len(testi))
	for i := 0; i < len(testi); i += bloccoOllama {
		blocco := testi[i:min(i+bloccoOllama, len(testi))]
		richiesta := map[string]interface{}{"model": o.model, "input": blocco}
		if o.contesto > 0 {
			richiesta["options"] = map[string]int{"num_ctx": o.contesto, "num_batch": o.contesto}
		}
		corpo, err := json.Marshal(richiesta)
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

// Digest is the digest Ollama reports for the model, the version a result
// records: a name like bge-m3 can point to other weights tomorrow.
func Digest(ctx context.Context, s engine.EmbedderSettings, client *http.Client) (string, error) {
	e, err := New(s, client)
	if err != nil {
		return "", err
	}
	o := e.(*ollama)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, o.url+"/api/tags", nil)
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("embedder: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	var r struct {
		Models []struct {
			Name   string `json:"name"`
			Digest string `json:"digest"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return "", fmt.Errorf("embedder: unreadable ollama answer: %w", err)
	}
	for _, m := range r.Models {
		if m.Name == o.model || m.Name == o.model+":latest" {
			return m.Digest, nil
		}
	}
	return "", fmt.Errorf("embedder: model %q not found in ollama", o.model)
}

func estratto(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 200 {
		s = s[:200] + "..."
	}
	return s
}
