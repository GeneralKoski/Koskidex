package engine

import (
	"sync"
)

// Posting represents a single occurrence of a term in a document
type Posting struct {
	DocID    string
	Field    string
	Position int
	TF       float64 // term frequency (calculated later)
}

// InvertedIndex maps terms to their occurrences in documents
// and also stores the documents themselves.
type InvertedIndex struct {
	mu           sync.RWMutex
	index        map[string][]Posting
	docs         map[string]map[string]interface{} // docID -> original document
	docToTerms   map[string][]string               // docID -> list of terms in it (for fast deletion)
	prefixMap    map[string][]string               // first 2 chars -> list of terms for fuzzy search
}

// NewInvertedIndex creates a new inverted index
func NewInvertedIndex() *InvertedIndex {
	return &InvertedIndex{
		index:      make(map[string][]Posting),
		docs:       make(map[string]map[string]interface{}),
		docToTerms: make(map[string][]string),
		prefixMap:  make(map[string][]string),
	}
}

// AddDocument adds a document to the index
func (idx *InvertedIndex) AddDocument(docID string, doc map[string]interface{}, settings Settings) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Store document
	idx.docs[docID] = doc

	// Determine searchable fields
	var fields []string
	if len(settings.SearchableFields) > 0 {
		fields = settings.SearchableFields
	} else {
		for k, v := range doc {
			if _, ok := v.(string); ok {
				fields = append(fields, k)
			}
		}
	}

	// Tokenize searchable fields
	for _, field := range fields {
		val, ok := doc[field]
		if ok {
			strVal, ok := val.(string)
			if ok {
				// Tokenize searchable fields
				tokens := Tokenize(strVal, field, settings.StopWords)

				// Token Expansion for Synonyms
				expandedTokens := make([]Token, 0, len(tokens))
				for _, t := range tokens {
					expandedTokens = append(expandedTokens, t)
					if syns, ok := settings.Synonyms[t.Term]; ok {
						for _, syn := range syns {
							expandedTokens = append(expandedTokens, Token{
								Term:     syn,
								Position: t.Position,
								Field:    field,
							})
						}
					}
				}
				
				// Group by term to calculate basic TF
				termCounts := make(map[string]int)
				for _, t := range expandedTokens {
					termCounts[t.Term]++
					post := Posting{
						DocID:    docID,
						Field:    field,
						Position: t.Position,
					}
					idx.index[t.Term] = append(idx.index[t.Term], post)
					
					// Track terms per document for fast deletion
					idx.docToTerms[docID] = append(idx.docToTerms[docID], t.Term)
					
					// Add to prefix map
					prefix := getPrefix(t.Term)
					idx.addToPrefixMap(prefix, t.Term)
				}

				}
			}
		}
	}
}

// SearchExact finds documents containing all the exact terms (AND logic for simplicity initially)
func (idx *InvertedIndex) SearchExact(query string, settings Settings) []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	tokens := Tokenize(query, "", settings.StopWords)
	if len(tokens) == 0 {
		return nil
	}

	// Basic AND search
	docIDCounts := make(map[string]int)
	for _, t := range tokens {
		postings := idx.index[t.Term]
		
		seenDocsForTerm := make(map[string]bool)
		for _, p := range postings {
			if !seenDocsForTerm[p.DocID] {
				docIDCounts[p.DocID]++
				seenDocsForTerm[p.DocID] = true
			}
		}
	}

	var results []string
	requiredMatches := len(tokens)
	for docID, count := range docIDCounts {
		if count == requiredMatches {
			results = append(results, docID)
		}
	}

	return results
}

// GetDocument returns a document by ID
func (idx *InvertedIndex) GetDocument(docID string) (map[string]interface{}, bool) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	doc, ok := idx.docs[docID]
	return doc, ok
}

// GetDocCount returns number of documents
func (idx *InvertedIndex) GetDocCount() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.docs)
}

// Settings defines per-index configuration
type Settings struct {
	SearchableFields []string            `json:"searchable_fields"`
	DisplayedFields  []string            `json:"displayed_fields"`
	RankingRules     []string            `json:"ranking_rules"`
	StopWords        map[string]bool     `json:"stop_words"`
	Synonyms         map[string][]string `json:"synonyms"`
	TypoTolerance    TypoSettings        `json:"typo_tolerance"`
}

type TypoSettings struct {
	Enabled               bool `json:"enabled"`
	MinWordLengthOneTypo  int  `json:"min_word_length_one_typo"`
	MinWordLengthTwoTypos int  `json:"min_word_length_two_typos"`
}

// DefaultSettings returns sane defaults
func DefaultSettings() Settings {
	return Settings{
		SearchableFields: nil,
		DisplayedFields:  nil,
		RankingRules:     []string{"exactness", "typo", "proximity", "attribute"},
		StopWords:        make(map[string]bool),
		Synonyms:         make(map[string][]string),
		TypoTolerance: TypoSettings{
			Enabled:               true,
			MinWordLengthOneTypo:  4,
			MinWordLengthTwoTypos: 8,
		},
	}
}

// DeleteDocument removes a document from the index
func (idx *InvertedIndex) DeleteDocument(docID string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// 1. Get terms associated with this document for targeted removal
	terms, ok := idx.docToTerms[docID]
	if !ok {
		return
	}

	// 2. Remove from inverted index
	for _, term := range terms {
		postings := idx.index[term]
		newPostings := make([]Posting, 0, len(postings))
		for _, p := range postings {
			if p.DocID != docID {
				newPostings = append(newPostings, p)
			}
		}
		if len(newPostings) == 0 {
			delete(idx.index, term)
		} else {
			idx.index[term] = newPostings
		}
	}

	// 3. Cleanup document maps
	delete(idx.docs, docID)
	delete(idx.docToTerms, docID)
}

func getPrefix(term string) string {
	runes := []rune(term)
	if len(runes) >= 2 {
		return string(runes[:2])
	}
	return term
}

func (idx *InvertedIndex) addToPrefixMap(prefix, term string) {
	for _, t := range idx.prefixMap[prefix] {
		if t == term {
			return
		}
	}
	idx.prefixMap[prefix] = append(idx.prefixMap[prefix], term)
}

// GetAllDocs returns all documents in the index
func (idx *InvertedIndex) GetAllDocs() map[string]map[string]interface{} {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	
	docsCopy := make(map[string]map[string]interface{})
	for k, v := range idx.docs {
		docsCopy[k] = v
	}
	return docsCopy
}
