package storage

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/GeneralKoski/Koskidex/internal/engine"
)

type Options struct {
	DataDir string
}

type DocRecord struct {
	ID   string
	Data map[string]interface{}
}

type IndexData struct {
	Settings engine.Settings
	Docs     []DocRecord
}

type WALOperation struct {
	Op       string                 `json:"op"`
	Index    string                 `json:"index"`
	DocID    string                 `json:"doc_id,omitempty"`
	DocIDs   []string               `json:"doc_ids,omitempty"`
	DocData  map[string]interface{} `json:"doc_data,omitempty"`
	Settings *engine.Settings       `json:"settings,omitempty"`
}

type Persistence struct {
	opts     Options
	filePath string
	saveCh   chan struct{}
	wg       sync.WaitGroup
	pendMu   sync.Mutex
	pending  func() map[string]IndexData // snapshot da scrivere, nil se non c'è niente
	gen      int                         // cresce a ogni SaveFunc
	walPath  string
	walFile  *os.File
	walMutex sync.Mutex
	syncs    int // WAL syncs done, for the tests
}

func NewPersistence(opts Options) *Persistence {
	_ = os.MkdirAll(opts.DataDir, 0755)

	walPath := filepath.Join(opts.DataDir, "operations.log")
	walFile, _ := os.OpenFile(walPath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)

	p := &Persistence{
		opts:     opts,
		filePath: filepath.Join(opts.DataDir, "koskidex.db"),
		walPath:  walPath,
		walFile:  walFile,
		saveCh:   make(chan struct{}, 1),
	}

	p.wg.Add(1)
	go p.worker()

	return p
}

func (p *Persistence) worker() {
	defer p.wg.Done()

	timer := time.NewTimer(0)
	if !timer.Stop() {
		<-timer.C
	}

	for {
		select {
		case _, ok := <-p.saveCh:
			if !ok {
				timer.Stop()
				_ = p.flush() // best-effort flush on shutdown
				return
			}
			timer.Reset(1 * time.Second)

		case <-timer.C:
			if err := p.flush(); err != nil {
				// Snapshot failed: keep the pending snapshot and the WAL intact
				// so nothing is lost, and retry on the next tick.
				timer.Reset(1 * time.Second)
			}
		}
	}
}

// flush costruisce lo snapshot in attesa, lo scrive e toglie dal WAL le
// operazioni che contiene. Costruirlo qui, una volta per giro del timer, e
// non a ogni scrittura evita di copiare tutti i documenti per ogni blocco
// caricato. Le operazioni scritte nel WAL prima di offset sono già nello stato
// che snapshot legge (il manager scrive il WAL e applica l'operazione sotto
// lo stesso lock); quelle arrivate dopo restano nel WAL.
func (p *Persistence) flush() error {
	p.pendMu.Lock()
	snapshot, gen := p.pending, p.gen
	p.pendMu.Unlock()
	if snapshot == nil {
		return nil
	}

	offset := p.walSize()
	if err := p.writeToDisk(snapshot()); err != nil {
		return err
	}
	p.truncateWALBefore(offset)

	p.pendMu.Lock()
	if p.gen == gen {
		p.pending = nil
	}
	p.pendMu.Unlock()
	return nil
}

func (p *Persistence) writeToDisk(data map[string]IndexData) error {
	slog.Debug("Saving to disk", "file", p.filePath)

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(data); err != nil {
		slog.Error("Failed to encode data", "error", err)
		return err
	}

	// Write to temp file then rename for atomic write
	tmpFile := p.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, buf.Bytes(), 0644); err != nil {
		slog.Error("Failed to write temp file", "error", err)
		return err
	}

	if err := os.Rename(tmpFile, p.filePath); err != nil {
		slog.Error("Failed to rename file", "error", err)
		return err
	}
	return nil
}

// SaveFunc chiede uno snapshot: snapshot si chiama più tardi, dal worker, e
// deve leggere lo stato corrente. Non blocca mai: più richieste ravvicinate
// diventano una sola scrittura.
func (p *Persistence) SaveFunc(snapshot func() map[string]IndexData) {
	p.pendMu.Lock()
	p.pending = snapshot
	p.gen++
	p.pendMu.Unlock()
	select {
	case p.saveCh <- struct{}{}:
	default:
	}
}

// Save salva dati già pronti: le operazioni nel WAL che non contengono vanno
// perse al troncamento, quindi il manager usa SaveFunc.
func (p *Persistence) Save(data map[string]IndexData) {
	p.SaveFunc(func() map[string]IndexData { return data })
}

func (p *Persistence) AppendWAL(op WALOperation) error {
	return p.AppendWALBatch([]WALOperation{op})
}

// AppendWALBatch appends the operations in one write and one sync. A sync is
// the expensive part (F_FULLFSYNC on macOS, a few milliseconds), so a request
// that adds a thousand documents pays it once, not a thousand times. The
// lines are the same as with AppendWAL, one per operation.
func (p *Persistence) AppendWALBatch(ops []WALOperation) error {
	p.walMutex.Lock()
	defer p.walMutex.Unlock()

	if p.walFile == nil {
		return nil
	}

	var buf bytes.Buffer
	for _, op := range ops {
		data, err := json.Marshal(op)
		if err != nil {
			return err
		}
		buf.Write(data)
		buf.WriteByte('\n')
	}
	_, err := p.walFile.Write(buf.Bytes())
	if err == nil {
		_ = p.walFile.Sync()
		p.syncs++
	}
	return err
}

func (p *Persistence) walSize() int64 {
	p.walMutex.Lock()
	defer p.walMutex.Unlock()
	if p.walFile == nil {
		return 0
	}
	fi, err := p.walFile.Stat()
	if err != nil {
		return 0
	}
	return fi.Size()
}

// truncateWALBefore toglie dal WAL i primi offset byte, già nello snapshot.
// Quello che segue si riscrive in un file nuovo rinominato al suo posto, così
// un crash a metà lascia il WAL vecchio, che si può rigiocare per intero.
func (p *Persistence) truncateWALBefore(offset int64) {
	p.walMutex.Lock()
	defer p.walMutex.Unlock()
	if p.walFile == nil {
		return
	}

	var coda []byte
	if fi, err := p.walFile.Stat(); err == nil && fi.Size() > offset {
		coda = make([]byte, fi.Size()-offset)
		if _, err := p.walFile.ReadAt(coda, offset); err != nil {
			return // il WAL intero resta: rigiocarlo è sicuro
		}
	}
	if len(coda) > 0 {
		tmp := p.walPath + ".tmp"
		if err := os.WriteFile(tmp, coda, 0644); err != nil {
			return
		}
		if err := os.Rename(tmp, p.walPath); err != nil {
			return
		}
	}

	_ = p.walFile.Close()
	flag := os.O_CREATE | os.O_RDWR | os.O_APPEND
	if len(coda) == 0 {
		flag |= os.O_TRUNC
	}
	p.walFile, _ = os.OpenFile(p.walPath, flag, 0644)
}

func (p *Persistence) ReadWAL() ([]WALOperation, error) {
	data, err := os.ReadFile(p.walPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var ops []WALOperation
	lines := bytes.Split(data, []byte("\n"))
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var op WALOperation
		if err := json.Unmarshal(line, &op); err == nil {
			ops = append(ops, op)
		}
	}
	return ops, nil
}

func (p *Persistence) Wait() {
	close(p.saveCh)
	p.wg.Wait()
	p.walMutex.Lock()
	if p.walFile != nil {
		_ = p.walFile.Close()
		p.walFile = nil
	}
	p.walMutex.Unlock()
}

func (p *Persistence) LoadIndexes(callback func(string, []DocRecord, engine.Settings)) error {
	data, err := os.ReadFile(p.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No existing DB, normal
		}
		return err
	}

	dec := gob.NewDecoder(bytes.NewReader(data))
	var dbData map[string]IndexData
	if err := dec.Decode(&dbData); err != nil {
		return err
	}

	for name, indexData := range dbData {
		callback(name, indexData.Docs, indexData.Settings)
	}

	return nil
}

func init() {
	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}{})
}
