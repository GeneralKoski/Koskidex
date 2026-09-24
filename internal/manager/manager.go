package manager

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
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
		case "DELETE_DOCS":
			if idx, ok := mgr.indexes[op.Index]; ok {
				for _, id := range op.DocIDs {
					idx.Engine.DeleteDocument(id)
				}
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
//
// Every document needs an "id" (or "_id"): a non-empty string or an integer.
// If any of them lacks one, nothing is added and an *IDNonValidiError says
// which. skipped is therefore always 0; it stays in the signature because the
// HTTP response has reported {added, skipped} since the CHANGELOG entry that
// introduced it, and clients may read it.
func (m *Manager) AddDocuments(indexName string, docs []map[string]interface{}) (added, skipped int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	idx, exists := m.indexes[indexName]
	if !exists {
		return 0, 0, ErrIndexNotFound
	}

	// Tutti gli id si controllano prima di toccare l'indice: con un id non
	// valido non entra niente, invece di un blocco caricato a meta'.
	ids := make([]string, len(docs))
	var sbagliati []int
	for i, doc := range docs {
		idVal, ok := doc["id"]
		if !ok {
			idVal = doc["_id"] // fallback
		}
		id, valido := idCanonico(idVal)
		if !valido {
			sbagliati = append(sbagliati, i)
			continue
		}
		ids[i] = id
	}
	if len(sbagliati) > 0 {
		return 0, 0, &IDNonValidiError{Posizioni: sbagliati}
	}

	for i, doc := range docs {
		idStr := ids[i]
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

// DeleteDocuments removes many documents in one step: one WAL record and one
// sync for the whole list, instead of one per id. Ids are strings or integers,
// as in AddDocuments; if any of them is unusable nothing is deleted and an
// *IDNonValidiError says which. Ids that are not in the index are not an
// error. deleted counts the documents that were actually there.
func (m *Manager) DeleteDocuments(indexName string, rawIDs []interface{}) (deleted int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	idx, exists := m.indexes[indexName]
	if !exists {
		return 0, ErrIndexNotFound
	}

	var sbagliati []int
	var presenti []string
	visti := make(map[string]bool, len(rawIDs))
	for i, v := range rawIDs {
		id, valido := idCanonico(v)
		if !valido {
			sbagliati = append(sbagliati, i)
			continue
		}
		if visti[id] {
			continue
		}
		visti[id] = true
		if _, ok := idx.Engine.GetDocument(id); ok {
			presenti = append(presenti, id)
		}
	}
	if len(sbagliati) > 0 {
		return 0, &IDNonValidiError{Posizioni: sbagliati}
	}
	if len(presenti) == 0 {
		return 0, nil
	}

	if err := m.persistence.AppendWAL(storage.WALOperation{Op: "DELETE_DOCS", Index: indexName, DocIDs: presenti}); err != nil {
		return 0, fmt.Errorf("WAL write failed: %w", err)
	}
	for _, id := range presenti {
		idx.Engine.DeleteDocument(id)
	}
	m.invalidateCache(indexName)
	return len(presenti), m.triggerSaveLocked()
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

// IDNonValidiError dice quali documenti di una richiesta non hanno un id
// utilizzabile, per posizione nella richiesta a partire da 0.
type IDNonValidiError struct {
	Posizioni []int
}

func (e *IDNonValidiError) Error() string {
	parti := make([]string, 0, len(e.Posizioni))
	for i, p := range e.Posizioni {
		if i == 10 {
			parti = append(parti, fmt.Sprintf("e altri %d", len(e.Posizioni)-10))
			break
		}
		parti = append(parti, fmt.Sprintf("posizione %d", p))
	}
	return "id mancante o non valido nei documenti in " + strings.Join(parti, ", ") +
		": l'id deve essere una stringa non vuota o un numero intero"
}

// idCanonico accetta una stringa non vuota o un numero intero. JSON decodifica
// ogni numero come float64, e fmt.Sprint scriverebbe 1000000 come "1e+06":
// un id che cambia forma non si ritrova piu'. Un numero con la virgola non e'
// un id, e oltre 2^53 un float64 non rappresenta piu' ogni intero.
func idCanonico(v interface{}) (string, bool) {
	switch x := v.(type) {
	case string:
		return x, x != ""
	case float64:
		if x != math.Trunc(x) || math.IsInf(x, 0) || math.Abs(x) > 1<<53 {
			return "", false
		}
		return strconv.FormatInt(int64(x), 10), true
	}
	return "", false
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
