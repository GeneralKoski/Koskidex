package eval

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	// Il nome resta in coda anche nella collisione: e' quello che i grafici
	// cercano con un glob.
	for _, p := range []string{a, b} {
		if !strings.HasSuffix(p, "_scifact-bm25.json") {
			t.Fatalf("il nome dell'esecuzione deve restare in coda: %s", p)
		}
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

// The result files live inside the repository. Counting them as uncommitted
// code would flag every run after the first one of a batch as dirty.
func TestProvenienzaIgnoresTheResultFiles(t *testing.T) {
	dir := t.TempDir()
	git := func(args ...string) {
		c := exec.Command("git", append([]string{"-C", dir}, args...)...)
		c.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	git("init", "-q")
	os.MkdirAll(filepath.Join(dir, "eval", "results"), 0o755)
	os.WriteFile(filepath.Join(dir, "codice.go"), []byte("package x"), 0o644)
	os.WriteFile(filepath.Join(dir, "eval", "results", "r.json"), []byte("{}"), 0o644)
	git("add", ".")
	git("commit", "-q", "-m", "x")

	os.WriteFile(filepath.Join(dir, "eval", "results", "r.json"), []byte(`{"nuovo":1}`), 0o644)
	if p := Provenienza(dir); p["modifiche_non_committate"] != "false" {
		t.Fatalf("un risultato riscritto non e' codice modificato: %v", p)
	}

	os.WriteFile(filepath.Join(dir, "codice.go"), []byte("package y"), 0o644)
	if p := Provenienza(dir); p["modifiche_non_committate"] != "true" {
		t.Fatalf("il codice modificato va segnalato: %v", p)
	}
}
