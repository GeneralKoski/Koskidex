package eval

import (
	"os"
	"path/filepath"
	"testing"
)

// The archive is the history the charts are drawn from. Two runs with the same
// name on the same second must both survive.
func TestArchiviaNeverOverwrites(t *testing.T) {
	t.Setenv(VarArchivio, t.TempDir())

	a, err := Archivia("prove", "scifact-bm25", []byte(`{"n":1}`))
	if err != nil {
		t.Fatal(err)
	}
	b, err := Archivia("prove", "scifact-bm25", []byte(`{"n":2}`))
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatalf("due esecuzioni sono finite nello stesso file: %s", a)
	}
	primo, _ := os.ReadFile(a)
	if string(primo) != `{"n":1}` {
		t.Fatalf("il primo file e' stato sovrascritto: %s", primo)
	}
	if filepath.Dir(a) != filepath.Join(os.Getenv(VarArchivio), "prove") {
		t.Fatalf("file fuori dalla sottocartella: %s", a)
	}
}

// Without the variable nothing is written and nothing fails: the caller has
// to be able to tell and to say it.
func TestArchiviaWithoutTheVariableReturnsAnEmptyPath(t *testing.T) {
	t.Setenv(VarArchivio, "")
	p, err := Archivia("prove", "x", []byte("{}"))
	if err != nil || p != "" {
		t.Fatalf("attesi percorso vuoto e nessun errore, ottenuti %q e %v", p, err)
	}
}

func TestImprontaFileChangesWithTheContent(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	os.WriteFile(a, []byte("uno"), 0o644)
	os.WriteFile(b, []byte("due"), 0o644)
	ia, _ := ImprontaFile(a)
	ib, _ := ImprontaFile(b)
	if ia == "" || ia == ib {
		t.Fatalf("impronte non valide: %q e %q", ia, ib)
	}
}
