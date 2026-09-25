package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GeneralKoski/Koskidex/internal/engine"
)

func TestWALAppendAndRead(t *testing.T) {
	dir := t.TempDir()
	p := NewPersistence(Options{DataDir: dir})
	defer p.Wait()

	ops := []WALOperation{
		{Op: "CREATE_INDEX", Index: "movies"},
		{Op: "ADD_DOC", Index: "movies", DocID: "1", DocData: map[string]interface{}{"title": "The Matrix"}},
		{Op: "ADD_DOC", Index: "movies", DocID: "2", DocData: map[string]interface{}{"title": "Inception"}},
		{Op: "DELETE_DOC", Index: "movies", DocID: "1"},
	}

	for _, op := range ops {
		if err := p.AppendWAL(op); err != nil {
			t.Fatalf("AppendWAL failed: %v", err)
		}
	}

	readOps, err := p.ReadWAL()
	if err != nil {
		t.Fatalf("ReadWAL failed: %v", err)
	}
	if len(readOps) != 4 {
		t.Fatalf("expected 4 WAL ops, got %d", len(readOps))
	}
	if readOps[0].Op != "CREATE_INDEX" || readOps[0].Index != "movies" {
		t.Errorf("first op mismatch: %+v", readOps[0])
	}
	if readOps[3].Op != "DELETE_DOC" || readOps[3].DocID != "1" {
		t.Errorf("last op mismatch: %+v", readOps[3])
	}
}

func TestWALTruncateOnSave(t *testing.T) {
	dir := t.TempDir()
	p := NewPersistence(Options{DataDir: dir})

	// Append some WAL entries
	_ = p.AppendWAL(WALOperation{Op: "CREATE_INDEX", Index: "test"})
	_ = p.AppendWAL(WALOperation{Op: "ADD_DOC", Index: "test", DocID: "1", DocData: map[string]interface{}{"title": "Hello"}})

	// Trigger a save — debounce fires after 1s, which truncates WAL
	p.Save(map[string]IndexData{
		"test": {
			Settings: engine.DefaultSettings(),
			Docs:     []DocRecord{{ID: "1", Data: map[string]interface{}{"title": "Hello"}}},
		},
	})

	time.Sleep(2 * time.Second)
	p.Wait()

	// WAL should be truncated after save
	ops, err := p.ReadWAL()
	if err != nil {
		t.Fatalf("ReadWAL failed: %v", err)
	}
	if len(ops) != 0 {
		t.Errorf("WAL should be empty after save+truncate, got %d ops", len(ops))
	}
}

func TestWALFileCreated(t *testing.T) {
	dir := t.TempDir()
	p := NewPersistence(Options{DataDir: dir})
	defer p.Wait()

	walPath := filepath.Join(dir, "operations.log")
	if _, err := os.Stat(walPath); os.IsNotExist(err) {
		t.Fatal("WAL file should be created on init")
	}
}

func TestWALReadEmpty(t *testing.T) {
	dir := t.TempDir()
	p := NewPersistence(Options{DataDir: dir})
	defer p.Wait()

	ops, err := p.ReadWAL()
	if err != nil {
		t.Fatalf("ReadWAL on empty WAL should not error: %v", err)
	}
	if len(ops) != 0 {
		t.Errorf("expected 0 ops from empty WAL, got %d", len(ops))
	}
}

func TestWALReadNonExistent(t *testing.T) {
	dir := t.TempDir()
	// Don't create persistence — just try to read WAL directly
	p := &Persistence{walPath: filepath.Join(dir, "nonexistent.log")}

	ops, err := p.ReadWAL()
	if err != nil {
		t.Fatalf("ReadWAL on missing file should not error: %v", err)
	}
	if len(ops) != 0 {
		t.Errorf("expected 0 ops from missing WAL, got %d", len(ops))
	}
}

func TestWALSettingsOperation(t *testing.T) {
	dir := t.TempDir()
	p := NewPersistence(Options{DataDir: dir})
	defer p.Wait()

	s := engine.DefaultSettings()
	s.FieldWeights = map[string]float64{"title": 5.0}

	err := p.AppendWAL(WALOperation{Op: "UPDATE_SETTINGS", Index: "movies", Settings: &s})
	if err != nil {
		t.Fatalf("AppendWAL with settings failed: %v", err)
	}

	ops, _ := p.ReadWAL()
	if len(ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(ops))
	}
	if ops[0].Settings == nil {
		t.Fatal("settings should not be nil")
	}
	if ops[0].Settings.FieldWeights["title"] != 5.0 {
		t.Errorf("expected field weight 5.0, got %f", ops[0].Settings.FieldWeights["title"])
	}
}

// Un blocco di operazioni si scrive in una volta sola, con una sola
// sincronizzazione: su macOS ogni Sync è un F_FULLFSYNC da qualche
// millisecondo, e 10.000 documenti costavano mezzo minuto. Le righe restano
// quelle di sempre, una per operazione, e il replay non cambia.
func TestWALBatchAppendsEveryOperationInOrder(t *testing.T) {
	dir := t.TempDir()
	p := NewPersistence(Options{DataDir: dir})
	defer p.Wait()

	_ = p.AppendWAL(WALOperation{Op: "CREATE_INDEX", Index: "atti"})
	blocco := []WALOperation{
		{Op: "ADD_DOC", Index: "atti", DocID: "1", DocData: map[string]interface{}{"t": "uno"}},
		{Op: "ADD_DOC", Index: "atti", DocID: "2", DocData: map[string]interface{}{"t": "due"}},
		{Op: "ADD_DOC", Index: "atti", DocID: "3", DocData: map[string]interface{}{"t": "tre"}},
	}
	if err := p.AppendWALBatch(blocco); err != nil {
		t.Fatalf("AppendWALBatch: %v", err)
	}

	letti, err := p.ReadWAL()
	if err != nil {
		t.Fatal(err)
	}
	if len(letti) != 4 {
		t.Fatalf("attese 4 operazioni, lette %d", len(letti))
	}
	for i, id := range []string{"1", "2", "3"} {
		if letti[i+1].Op != "ADD_DOC" || letti[i+1].DocID != id {
			t.Fatalf("operazione %d: attesa ADD_DOC %s, letta %+v", i+1, id, letti[i+1])
		}
	}
}

func TestWALBatchCostsOneSync(t *testing.T) {
	dir := t.TempDir()
	p := NewPersistence(Options{DataDir: dir})
	defer p.Wait()

	blocco := make([]WALOperation, 200)
	for i := range blocco {
		blocco[i] = WALOperation{Op: "ADD_DOC", Index: "atti", DocID: fmt.Sprint(i)}
	}
	prima := p.syncs
	if err := p.AppendWALBatch(blocco); err != nil {
		t.Fatal(err)
	}
	if n := p.syncs - prima; n != 1 {
		t.Fatalf("200 operazioni in un blocco devono costare una sincronizzazione, ne sono costate %d", n)
	}
}
