package manager

import (
	"errors"
	"fmt"
	"sort"
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

	walOps, _ := p.ReadWAL()
	for _, op := range walOps {
		switch op.Op {
		case "CREATE_INDEX":
			if _, exists := mgr.indexes[op.Index]; !exists {
				mgr.indexes[op.Index] = &Index{
					Name:     op.Index,
					Engine:   engine.NewInvertedIndex(),
					Settings: engine.DefaultSettings(),
				}
			}
		case "DELETE_INDEX":
			delete(mgr.indexes, op.Index)
		case "UPDATE_SETTINGS":
			if idx, ok := mgr.indexes[op.Index]; ok && op.Settings != nil {
				idx.Settings = *op.Settings
			}
		case "ADD_DOC":
			if idx, ok := mgr.indexes[op.Index]; ok && op.DocData != nil {
				idx.Engine.AddDocument(op.DocID, op.DocData, idx.Settings)
			}
		case "DELETE_DOC":
			if idx, ok := mgr.indexes[op.Index]; ok && op.DocID != "" {
				idx.Engine.DeleteDocument(op.DocID)
			}
		}
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
		Name:     name,
		Engine:   engine.NewInvertedIndex(),
		Settings: engine.DefaultSettings(),
	}

	if err := m.persistence.AppendWAL(storage.WALOperation{Op: "CREATE_INDEX", Index: name}); err != nil {
		delete(m.indexes, name)
		return fmt.Errorf("WAL write failed: %w", err)
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
	if err := m.persistence.AppendWAL(storage.WALOperation{Op: "DELETE_INDEX", Index: name}); err != nil {
		return fmt.Errorf("WAL write failed: %w", err)
	}
	return m.triggerSaveLocked()
}

// AddDocuments adds documents to an index and saves to disk.
// It returns how many documents were indexed and how many were skipped
// because they lacked a valid string "id" (or "_id") field.
func (m *Manager) AddDocuments(indexName string, docs []map[string]interface{}) (added, skipped int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	idx, exists := m.indexes[indexName]
	if !exists {
		return 0, 0, ErrIndexNotFound
	}

	for _, doc := range docs {
		idVal, ok := doc["id"]
		if !ok {
			idVal = doc["_id"] // fallback
		}
		idStr, ok := idVal.(string)
		if !ok || idStr == "" {
			skipped++
			continue
		}
		if err := m.persistence.AppendWAL(storage.WALOperation{Op: "ADD_DOC", Index: indexName, DocID: idStr, DocData: doc}); err != nil {
			return added, skipped, fmt.Errorf("WAL write failed: %w", err)
		}
		idx.Engine.AddDocument(idStr, doc, idx.Settings)
		added++
	}

	m.invalidateCache(indexName)
	return added, skipped, m.triggerSaveLocked()
}

// ListDocuments returns a page of documents from an index, ordered by ID,
// along with the total number of documents in the index.
func (m *Manager) ListDocuments(indexName string, limit, offset int) ([]map[string]interface{}, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	idx, exists := m.indexes[indexName]
	if !exists {
		return nil, 0, ErrIndexNotFound
	}

	all := idx.Engine.GetAllDocs()
	ids := make([]string, 0, len(all))
	for id := range all {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	total := len(ids)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}

	page := make([]map[string]interface{}, 0, end-offset)
	for _, id := range ids[offset:end] {
		page = append(page, all[id])
	}
	return page, total, nil
}

// DeleteDocument removes a single document from an index
func (m *Manager) DeleteDocument(indexName, docID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	idx, exists := m.indexes[indexName]
	if !exists {
		return ErrIndexNotFound
	}

	if err := m.persistence.AppendWAL(storage.WALOperation{Op: "DELETE_DOC", Index: indexName, DocID: docID}); err != nil {
		return fmt.Errorf("WAL write failed: %w", err)
	}
	idx.Engine.DeleteDocument(docID)
	m.invalidateCache(indexName)
	return m.triggerSaveLocked()
}

// UpdateSettings updates index configuration and re-indexes all documents.
func (m *Manager) UpdateSettings(indexName string, settings engine.Settings) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	idx, exists := m.indexes[indexName]
	if !exists {
		return ErrIndexNotFound
	}

	idx.Settings = settings

	if err := m.persistence.AppendWAL(storage.WALOperation{Op: "UPDATE_SETTINGS", Index: indexName, Settings: &settings}); err != nil {
		return fmt.Errorf("WAL write failed: %w", err)
	}

	// Re-index all docs in place with the new settings (synonyms, searchable
	// fields, etc.). Reindex holds the engine lock for the whole rebuild.
	idx.Engine.Reindex(settings)

	m.invalidateCache(indexName)
	return m.triggerSaveLocked()
}

// triggerSaveLocked saves all data to disk using the debounced save in the
// persistence layer. The caller must hold m.mu.
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
