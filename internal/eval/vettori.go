package eval

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/GeneralKoski/Koskidex/internal/embedder"
	"github.com/GeneralKoski/Koskidex/internal/engine"
)

// CalcolaVettori computes, through the cache at cachePath, the vectors of
// every document (from TestoDocumento) and of every query text, and returns
// what a run records about them: model, digest, milliseconds, new vectors.
func CalcolaVettori(impostazioni engine.EmbedderSettings, cachePath string, docs []Document, domande map[string]string) (Vettori, map[string]string, error) {
	ctx := context.Background()
	client := &http.Client{Timeout: 10 * time.Minute}
	e, err := embedder.New(impostazioni, client)
	if err != nil {
		return Vettori{}, nil, err
	}
	digest, err := embedder.Digest(ctx, impostazioni, client)
	if err != nil {
		return Vettori{}, nil, err
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		return Vettori{}, nil, err
	}
	cache, err := embedder.OpenCache(cachePath)
	if err != nil {
		return Vettori{}, nil, err
	}
	defer func() { _ = cache.Close() }()
	prima := cache.Len()

	t0 := time.Now()
	testi := make([]string, len(docs))
	for i, d := range docs {
		testi[i] = TestoDocumento(d)
	}
	const blocco = 256
	v := Vettori{Documenti: make(map[string][]float64, len(docs)), Query: make(map[string][]float64, len(domande))}
	for i := 0; i < len(testi); i += blocco {
		fine := min(i+blocco, len(testi))
		out, err := cache.Embed(ctx, e, testi[i:fine])
		if err != nil {
			return Vettori{}, nil, err
		}
		for j, vettore := range out {
			v.Documenti[docs[i+j].ID] = vettore
		}
		fmt.Printf("\rvettori dei documenti: %d/%d", fine, len(testi))
	}
	fmt.Println()
	var qtesti []string
	for _, q := range domande {
		qtesti = append(qtesti, q)
	}
	sort.Strings(qtesti)
	out, err := cache.Embed(ctx, e, qtesti)
	if err != nil {
		return Vettori{}, nil, err
	}
	for i, q := range qtesti {
		v.Query[q] = out[i]
	}

	return v, map[string]string{
		"embedder":        e.Nome(),
		"embedder_digest": digest,
		"vettori_ms":      fmt.Sprintf("%.0f", float64(time.Since(t0).Nanoseconds())/1e6),
		"vettori_nuovi":   fmt.Sprint(cache.Len() - prima),
	}, nil
}
