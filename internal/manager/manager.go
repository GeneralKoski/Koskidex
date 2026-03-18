package manager

import (
	"errors"
	"sync"

	"github.com/GeneralKoski/Koskidex/internal/engine"
	"github.com/GeneralKoski/Koskidex/internal/storage"
)

var (
	ErrIndexNotFound      = errors.New("index not found")
	ErrIndexAlreadyExists = errors.New("index already exists")
)

// Index wrapper including the engine and its name
type Index struct {
	Name     string
	Engine   *engine.InvertedIndex
	Settings engine.Settings
}

type CacheInvalidator interface {
	InvalidatePrefix(prefix string)
}

// Manager manages multiple indexes
type Manager struct {
	mu               sync.RWMutex
	indexes          map[string]*Index
	storageOpts      storage.Options
	persistence      *storage.Persistence
	cacheInvalidator CacheInvalidator
}

func (m *Manager) SetCacheInvalidator(c CacheInvalidator) {
	m.cacheInvalidator = c
}

func (m *Manager) invalidateCache(indexName string) {
	if m.cacheInvalidator != nil {
		m.cacheInvalidator.InvalidatePrefix(indexName + "|")
	}
}

// NewManager initializes a new Manager and loads existing indexes from disk
func NewManager(opts storage.Options) (*Manager, error) {
	p := storage.NewPersistence(opts)

	mgr := &Manager{
		indexes:     make(map[string]*Index),
		storageOpts: opts,
		persistence: p,
	}

	// For simplicity in Phase 1, we just return a new empty manager
	// Later persistence layer can load from disk here
	err := p.LoadIndexes(func(name string, d []storage.DocRecord, settings engine.Settings) {
		idx := &Index{
			Name:     name,
			Engine:   engine.NewInvertedIndex(),
			Settings: settings,
		}
		for _, doc := range d {
			idx.Engine.AddDocument(doc.ID, doc.Data, settings)
		}
		mgr.indexes[name] = idx
	})

	if err != nil {
		return nil, err
	}

	return mgr, nil
}

// CreateIndex creates a new index
func (m *Manager) CreateIndex(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.indexes[name]; exists {
		return ErrIndexAlreadyExists
	}

	m.indexes[name] = &Index{
		Name:   name,
		Engine: engine.NewInvertedIndex(),
		Settings: engine.Settings{
			SearchableFields: nil, // index all fields by default
			StopWords:        make(map[string]bool),
		},
	}

	return m.triggerSaveLocked()
}

// GetIndex returns an index by name
func (m *Manager) GetIndex(name string) (*Index, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	idx, exists := m.indexes[name]
	if !exists {
		return nil, ErrIndexNotFound
	}

	return idx, nil
}

// ListIndexes returns a list of all index names
func (m *Manager) ListIndexes() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var names []string
	for name := range m.indexes {
		names = append(names, name)
	}
	return names
}

// DeleteIndex removes an index
func (m *Manager) DeleteIndex(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.indexes[name]; !exists {
		return ErrIndexNotFound
	}
	delete(m.indexes, name)

	m.invalidateCache(name)
	return m.triggerSaveLocked()
}

// AddDocuments adds documents to an index and saves to disk
func (m *Manager) AddDocuments(indexName string, docs []map[string]interface{}) error {
	idx, err := m.GetIndex(indexName)
	if err != nil {
		return err
	}

	for _, doc := range docs {
		idVal, ok := doc["id"]
		if !ok {
			idVal = doc["_id"] // fallback
		}
		if idStr, ok := idVal.(string); ok {
			idx.Engine.AddDocument(idStr, doc, idx.Settings)
		}
	}

	m.invalidateCache(indexName)
	return m.triggerSave()
}

// DeleteDocument removes a single document from an index
func (m *Manager) DeleteDocument(indexName, docID string) error {
	idx, err := m.GetIndex(indexName)
	if err != nil {
		return err
	}

	idx.Engine.DeleteDocument(docID)
	m.invalidateCache(indexName)
	return m.triggerSave()
}

// UpdateSettings updates index configuration
func (m *Manager) UpdateSettings(indexName string, settings engine.Settings) error {
	idx, err := m.GetIndex(indexName)
	if err != nil {
		return err
	}

	m.mu.Lock()
	idx.Settings = settings
	m.mu.Unlock()

	// Re-indexing is technically needed for synonyms/searchable fields changes
	// For simplicity, we just save and advise users to re-add docs or we could trigger re-indexing here
	// In a real product, we'd iterate over all docs and re-index them.

	// Re-index all docs with new settings
	allDocs := idx.Engine.GetAllDocs()
	newEngine := engine.NewInvertedIndex()
	for id, doc := range allDocs {
		newEngine.AddDocument(id, doc, settings)
	}

	m.mu.Lock()
	idx.Engine = newEngine
	m.mu.Unlock()

	m.invalidateCache(indexName)
	return m.triggerSave()
}

// Trigger Save saves all data to disk using debounced save in the persistence layer.
func (m *Manager) triggerSave() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.triggerSaveLocked()
}

func (m *Manager) triggerSaveLocked() error {
	// Extract data to save
	saveData := make(map[string]storage.IndexData)
	for name, idx := range m.indexes {
		var docs []storage.DocRecord
		// Get all docs doesn't exist directly on engine, we'd add it or retrieve them
		// For simplicity, we assume engine exposes docs or we extract them
		idxDocs := idx.Engine.GetAllDocs() // We need to add this method to engine
		for id, data := range idxDocs {
			docs = append(docs, storage.DocRecord{ID: id, Data: data})
		}
		saveData[name] = storage.IndexData{
			Settings: idx.Settings,
			Docs:     docs,
		}
	}

	m.persistence.Save(saveData)
	return nil
}

func (m *Manager) Close() {
	m.persistence.Wait()
}
