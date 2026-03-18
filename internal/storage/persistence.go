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
	DocData  map[string]interface{} `json:"doc_data,omitempty"`
	Settings *engine.Settings       `json:"settings,omitempty"`
}

type Persistence struct {
	opts       Options
	filePath   string
	saveCh     chan map[string]IndexData
	wg         sync.WaitGroup
	walPath    string
	walFile    *os.File
	walMutex   sync.Mutex
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
		saveCh:   make(chan map[string]IndexData, 10),
	}

	p.wg.Add(1)
	go p.worker()

	return p
}

func (p *Persistence) worker() {
	defer p.wg.Done()

	var latestData map[string]IndexData
	timer := time.NewTimer(0)
	if !timer.Stop() {
		<-timer.C
	}
	timerRunning := false

	for {
		select {
		case data, ok := <-p.saveCh:
			if !ok {
				if timerRunning {
					timer.Stop()
				}
				if latestData != nil {
					p.writeToDisk(latestData)
				}
				return
			}
			latestData = data
			if timerRunning {
				timer.Stop()
			}
			timer.Reset(1 * time.Second)
			timerRunning = true

		case <-timer.C:
			timerRunning = false
			if latestData != nil {
				p.writeToDisk(latestData)
				p.truncateWAL()
				latestData = nil
			}
		}
	}
}

func (p *Persistence) writeToDisk(data map[string]IndexData) {
	slog.Debug("Saving to disk", "file", p.filePath)

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(data); err != nil {
		slog.Error("Failed to encode data", "error", err)
		return
	}

	// Write to temp file then rename for atomic write
	tmpFile := p.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, buf.Bytes(), 0644); err != nil {
		slog.Error("Failed to write temp file", "error", err)
		return
	}

	if err := os.Rename(tmpFile, p.filePath); err != nil {
		slog.Error("Failed to rename file", "error", err)
	}
}

func (p *Persistence) Save(data map[string]IndexData) {
	p.saveCh <- data
}

func (p *Persistence) AppendWAL(op WALOperation) error {
	p.walMutex.Lock()
	defer p.walMutex.Unlock()

	if p.walFile == nil {
		return nil
	}

	data, err := json.Marshal(op)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = p.walFile.Write(data)
	if err == nil {
		p.walFile.Sync()
	}
	return err
}

func (p *Persistence) truncateWAL() {
	p.walMutex.Lock()
	defer p.walMutex.Unlock()
	if p.walFile != nil {
		p.walFile.Close()
	}
	p.walFile, _ = os.OpenFile(p.walPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
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
