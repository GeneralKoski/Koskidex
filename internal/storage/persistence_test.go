package storage

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/GeneralKoski/Koskidex/internal/engine"
)

func tempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return dir
}

func TestSaveAndLoad(t *testing.T) {
	dir := tempDir(t)
	p := NewPersistence(Options{DataDir: dir})

	data := map[string]IndexData{
		"test_index": {
			Settings: engine.DefaultSettings(),
			Docs: []DocRecord{
				{ID: "1", Data: map[string]interface{}{"title": "Hello"}},
				{ID: "2", Data: map[string]interface{}{"title": "World"}},
			},
		},
	}

	p.Save(data)
	// Wait for debounce to flush
	time.Sleep(2 * time.Second)
	p.Wait()

	// Verify file exists
	if _, err := os.Stat(filepath.Join(dir, "koskidex.db")); os.IsNotExist(err) {
		t.Fatal("database file was not created")
	}

	// Load and verify
	p2 := NewPersistence(Options{DataDir: dir})
	defer p2.Wait()

	var loadedName string
	var loadedDocs []DocRecord
	err := p2.LoadIndexes(func(name string, docs []DocRecord, settings engine.Settings) {
		loadedName = name
		loadedDocs = docs
	})
	if err != nil {
		t.Fatal("LoadIndexes failed:", err)
	}
	if loadedName != "test_index" {
		t.Fatalf("expected index name 'test_index', got %q", loadedName)
	}
	if len(loadedDocs) != 2 {
		t.Fatalf("expected 2 docs, got %d", len(loadedDocs))
	}
}

func TestLoadNonExistent(t *testing.T) {
	dir := tempDir(t)
	p := NewPersistence(Options{DataDir: dir})
	defer p.Wait()

	var called bool
	err := p.LoadIndexes(func(name string, docs []DocRecord, settings engine.Settings) {
		called = true
	})
	if err != nil {
		t.Fatal("LoadIndexes should not error on missing file:", err)
	}
	if called {
		t.Fatal("callback should not be called for empty db")
	}
}

func TestCorruptedFile(t *testing.T) {
	dir := tempDir(t)
	// Write garbage
	os.WriteFile(filepath.Join(dir, "koskidex.db"), []byte("not a gob file"), 0644)

	p := NewPersistence(Options{DataDir: dir})
	defer p.Wait()

	err := p.LoadIndexes(func(name string, docs []DocRecord, settings engine.Settings) {})
	if err == nil {
		t.Fatal("expected error loading corrupted file")
	}
}

func TestConcurrentSaves(t *testing.T) {
	dir := tempDir(t)
	p := NewPersistence(Options{DataDir: dir})

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			data := map[string]IndexData{
				"idx": {
					Settings: engine.DefaultSettings(),
					Docs:     []DocRecord{{ID: "1", Data: map[string]interface{}{"n": float64(n)}}},
				},
			}
			p.Save(data)
		}(i)
	}
	wg.Wait()
	time.Sleep(2 * time.Second)
	p.Wait()

	// Just verify no panic and file is valid
	p2 := NewPersistence(Options{DataDir: dir})
	defer p2.Wait()

	err := p2.LoadIndexes(func(name string, docs []DocRecord, settings engine.Settings) {})
	if err != nil {
		t.Fatal("failed to load after concurrent saves:", err)
	}
}
