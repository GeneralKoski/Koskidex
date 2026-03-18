package storage

import (
	"bytes"
	"encoding/gob"
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

type Persistence struct {
	opts       Options
	filePath   string
	saveCh     chan map[string]IndexData
	wg         sync.WaitGroup
}

func NewPersistence(opts Options) *Persistence {
	os.MkdirAll(opts.DataDir, 0755)

	p := &Persistence{
		opts:     opts,
		filePath: filepath.Join(opts.DataDir, "koskidex.db"),
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
