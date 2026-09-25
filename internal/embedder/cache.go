package embedder

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

// Cache keeps every vector computed, by model and text, so that re-adding the
// same documents does not call the model again. With a file it survives a
// restart: Documentale rebuilds its index from scratch, and ten thousand acts
// take minutes to embed.
type Cache struct {
	mu      sync.Mutex
	vettori map[string][]float64
	file    *os.File
}

type rigaCache struct {
	K string    `json:"k"`
	V []float64 `json:"v"`
}

// NewCache keeps the vectors in memory only.
func NewCache() *Cache {
	return &Cache{vettori: make(map[string][]float64)}
}

// OpenCache loads the vectors in path, one JSON line each, and appends the new
// ones there. A line that does not parse, as the last one after a crash in
// the middle of a write, is skipped: that vector is computed again.
func OpenCache(path string) (*Cache, error) {
	c := NewCache()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 64<<20)
	for sc.Scan() {
		var r rigaCache
		if json.Unmarshal(sc.Bytes(), &r) == nil && r.K != "" {
			c.vettori[r.K] = r.V
		}
	}
	if err := sc.Err(); err != nil {
		_ = f.Close()
		return nil, err
	}
	c.file = f
	return c, nil
}

// Len is how many vectors the cache holds.
func (c *Cache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.vettori)
}

// Close closes the file, if any.
func (c *Cache) Close() error {
	if c.file == nil {
		return nil
	}
	return c.file.Close()
}

func chiave(e Embedder, testo string) string {
	h := sha256.Sum256([]byte(e.Nome() + "\x00" + testo))
	return hex.EncodeToString(h[:])
}

// Embed returns a vector for each text, calling the model only for the texts
// it has not seen, each once. The lock is not held during the call.
func (c *Cache) Embed(ctx context.Context, e Embedder, testi []string) ([][]float64, error) {
	chiavi := make([]string, len(testi))
	var mancanti []string
	visti := make(map[string]bool)
	c.mu.Lock()
	for i, t := range testi {
		chiavi[i] = chiave(e, t)
		if _, ok := c.vettori[chiavi[i]]; !ok && !visti[chiavi[i]] {
			visti[chiavi[i]] = true
			mancanti = append(mancanti, t)
		}
	}
	c.mu.Unlock()

	if len(mancanti) > 0 {
		nuovi, err := e.Embed(ctx, mancanti)
		if err != nil {
			return nil, err
		}
		c.mu.Lock()
		err = c.aggiungiLocked(e, mancanti, nuovi)
		c.mu.Unlock()
		if err != nil {
			return nil, err
		}
	}

	out := make([][]float64, len(testi))
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, k := range chiavi {
		out[i] = c.vettori[k]
	}
	return out, nil
}

func (c *Cache) aggiungiLocked(e Embedder, testi []string, vettori [][]float64) error {
	var buf []byte
	for i, t := range testi {
		k := chiave(e, t)
		c.vettori[k] = vettori[i]
		if c.file != nil {
			riga, err := json.Marshal(rigaCache{K: k, V: vettori[i]})
			if err != nil {
				return err
			}
			buf = append(append(buf, riga...), '\n')
		}
	}
	if c.file != nil && len(buf) > 0 {
		if _, err := c.file.Write(buf); err != nil {
			return fmt.Errorf("embedder cache: %w", err)
		}
	}
	return nil
}
