package engine

import (
	"strconv"
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
	mu         sync.RWMutex
	index      map[string][]Posting
	docs       map[string]map[string]interface{} // docID -> original document
	docToTerms map[string][]string               // docID -> list of terms in it (for fast deletion)
	prefixMap  map[string][]string               // first 2 chars -> list of terms for fuzzy search

	// Collection statistics BM25 needs. Maintained while indexing rather than
	// derived at query time: df would cost O(postings) for a common term, and
	// docToTerms cannot give the length because it is deduplicated.
	docFreq    map[string]int // term -> number of documents containing it
	docLengths map[string]int // docID -> number of indexed tokens, occurrences included
	totalLen   int            // sum of docLengths, for the average
}

// NewInvertedIndex creates a new inverted index
func NewInvertedIndex() *InvertedIndex {
	return &InvertedIndex{
		index:      make(map[string][]Posting),
		docs:       make(map[string]map[string]interface{}),
		docToTerms: make(map[string][]string),
		prefixMap:  make(map[string][]string),
		docFreq:    make(map[string]int),
		docLengths: make(map[string]int),
	}
}

// AddDocument adds a document to the index
func (idx *InvertedIndex) AddDocument(docID string, doc map[string]interface{}, settings Settings) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.addDocumentLocked(docID, doc, settings)
}

// Reindex rebuilds the entire index in place using the given settings.
// It holds the write lock for the whole rebuild, so concurrent reads and
// writes are serialized safely and no documents can be lost mid-rebuild.
func (idx *InvertedIndex) Reindex(settings Settings) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	docs := idx.docs
	idx.index = make(map[string][]Posting)
	idx.docs = make(map[string]map[string]interface{})
	idx.docToTerms = make(map[string][]string)
	idx.prefixMap = make(map[string][]string)
	idx.docFreq = make(map[string]int)
	idx.docLengths = make(map[string]int)
	idx.totalLen = 0

	for docID, doc := range docs {
		idx.addDocumentLocked(docID, doc, settings)
	}
}

// distanzaFraElementi separa le posizioni di due elementi consecutivi di un
// campo lista, come position_increment_gap in Elasticsearch: gli elementi di
// una lista non sono una frase, e l'ultima parola di uno non deve risultare
// accanto alla prima del successivo.
const distanzaFraElementi = 100

// tokenizzaValore trasforma in token il valore di un campo. Una stringa si
// tokenizza com'e'; una lista elemento per elemento, stringhe e numeri, con
// distanzaFraElementi fra un elemento e l'altro; un numero da solo solo se il
// campo e' stato dichiarato cercabile, perche' senza campi dichiarati prezzi e
// anni di un catalogo non devono diventare cercabili. Il resto (booleani,
// oggetti) non e' testo. Il secondo valore dice se il campo ha prodotto testo.
func tokenizzaValore(v interface{}, field string, settings Settings, esplicito bool) ([]Token, bool) {
	switch x := v.(type) {
	case string:
		return Tokenize(x, field, settings), true
	case float64:
		if !esplicito {
			return nil, false
		}
		return Tokenize(strconv.FormatFloat(x, 'f', -1, 64), field, settings), true
	case []string:
		elementi := make([]interface{}, len(x))
		for i, s := range x {
			elementi[i] = s
		}
		return tokenizzaValore(elementi, field, settings, esplicito)
	case []interface{}:
		var out []Token
		base := 0
		for _, e := range x {
			var testo string
			switch y := e.(type) {
			case string:
				testo = y
			case float64:
				testo = strconv.FormatFloat(y, 'f', -1, 64)
			default:
				continue
			}
			tok := Tokenize(testo, field, settings)
			for _, tk := range tok {
				tk.Position += base
				out = append(out, tk)
			}
			if len(tok) > 0 {
				base = out[len(out)-1].Position + distanzaFraElementi
			}
		}
		return out, true
	}
	return nil, false
}

// addDocumentLocked indexes a document. The caller must hold idx.mu.
func (idx *InvertedIndex) addDocumentLocked(docID string, doc map[string]interface{}, settings Settings) {
	// Purge any previous version of this document first, so re-adding the same
	// id (update) doesn't leave stale postings behind or double-count terms.
	idx.deleteDocumentLocked(docID)

	// Store document
	idx.docs[docID] = doc

	// Determine searchable fields
	var fields []string
	esplicito := len(settings.SearchableFields) > 0
	if esplicito {
		fields = settings.SearchableFields
	} else {
		for k, v := range doc {
			switch v.(type) {
			case string, []interface{}, []string:
				fields = append(fields, k)
			}
		}
	}

	// Distinct terms in this document, recorded once for O(terms) deletion.
	docTerms := make(map[string]bool)

	// Tokenize searchable fields
	for _, field := range fields {
		val, ok := doc[field]
		if ok {
			tokens, ok := tokenizzaValore(val, field, settings, esplicito)
			if ok {

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

					// One posting is one occurrence: this is the document
					// length BM25 normalises by.
					idx.docLengths[docID]++
					idx.totalLen++

					// Track distinct terms per document for fast deletion, and
					// take the chance to count document frequency: this branch
					// runs exactly once per (document, term).
					if !docTerms[t.Term] {
						docTerms[t.Term] = true
						idx.docToTerms[docID] = append(idx.docToTerms[docID], t.Term)
						idx.docFreq[t.Term]++
					}

					// Substring Indexing (Bigrams):
					// Instead of just the first 2 chars, we index all 2-char slices.
					// This allows matching "amsung" to "samsung".
					runes := []rune(t.Term)
					if len(runes) < 2 {
						idx.addToPrefixMap(t.Term, t.Term)
					} else {
						for i := 0; i <= len(runes)-2; i++ {
							substring := string(runes[i : i+2])
							idx.addToPrefixMap(substring, t.Term)
						}
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

	tokens := Tokenize(query, "", settings)
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

// GetDocument returns a document by ID. The returned map is the live indexed
// document shared with the index (not a copy) to stay allocation-free on the
// search hot path: callers MUST treat it as read-only and never mutate it, or
// they race concurrent readers/writers of the index.
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
	SearchableFields []string        `json:"searchable_fields"`
	DisplayedFields  []string        `json:"displayed_fields"`
	RankingRules     []string        `json:"ranking_rules"`
	StopWords        map[string]bool `json:"stop_words"`
	// Stemmer sceglie l'analizzatore dei termini. Vuoto = nessuno, cioe' il
	// comportamento di sempre.
	Stemmer       string              `json:"stemmer"`
	Synonyms      map[string][]string `json:"synonyms"`
	TypoTolerance TypoSettings        `json:"typo_tolerance"`
	FieldWeights  map[string]float64  `json:"field_weights"`
	Sitemap       SitemapSettings     `json:"sitemap"`
	RetrievalMode string              `json:"retrieval_mode"`
	ScoringMode   string              `json:"scoring_mode"`
	BM25K1        float64             `json:"bm25_k1"`
	BM25B         float64             `json:"bm25_b"`
	// SubstringMatch adds the documents with an indexed term that contains the
	// whole query, as Elasticsearch's wildcard *query* on a text field. Off by
	// default.
	SubstringMatch bool `json:"substring_match"`

	// Matching closer to Elasticsearch's multi_match as Documentale runs it
	// (best_fields, operator and, fuzziness AUTO, prefix_length 1). All off by
	// default, so an index saved earlier keeps its behaviour. The typo
	// thresholds of AUTO are TypoTolerance 3 and 6.
	//
	// DisablePrefixSearch drops the match of any term that starts with the
	// query word, whatever the distance: only exact and typo matches remain.
	DisablePrefixSearch bool `json:"disable_prefix_search"`
	// PrefixLength is how many leading characters of a typo match must be
	// exact, as Elasticsearch's prefix_length.
	PrefixLength int `json:"prefix_length"`
	// AllTermsInOneField requires, when every term is required, one field of
	// the document that holds every term, as best_fields with operator and.
	AllTermsInOneField bool `json:"all_terms_in_one_field"`
	// Tokenizer chooses the word boundaries. Empty = split on anything that is
	// not a letter or a digit, the behaviour up to now. TokenizerStandard keeps
	// word-internal punctuation as Elasticsearch's standard tokenizer does.
	Tokenizer string `json:"tokenizer"`
}

// TokenizerStandard is Elasticsearch's standard tokenizer (Unicode UAX#29) for
// the punctuation that matters in Latin text: one apostrophe, dot or colon
// between two letters ("dell'illuminazione", "d.lgs"), one apostrophe, dot,
// comma or semicolon between two digits ("14.01.2026", "3,5"), and the
// underscore next to a letter or a digit stay inside the word.
const TokenizerStandard = "standard"

// How a term's contribution to a document's score is computed.
//
//	ScoringLegacy ("legacy", the default) - (10 - typos + 2*exact) * weight
//	ScoringBM25   ("bm25")                - Robertson/Lucene BM25
//
// An empty value means ScoringLegacy, for the same reason RetrievalMode has
// one: settings are persisted, and an index saved earlier must not change
// behaviour on upgrade.
const (
	ScoringLegacy = "legacy"
	ScoringBM25   = "bm25"

	// Lucene's defaults. Not calibrated here on purpose: calibration is its
	// own phase, and a number tuned before it is measured is not a result.
	DefaultBM25K1 = 1.2
	DefaultBM25B  = 0.75
)

// How many of the query terms a document has to carry to be retrieved.
//
//	RetrievalAll ("all", the default) - conjunctive: every term is required.
//	RetrievalAny ("any")              - disjunctive: one term is enough, and a
//	                                    document matching more scores more.
//
// An empty value means RetrievalAll. Settings are persisted to disk, so an
// index saved before this field existed must keep behaving as it did.
const (
	RetrievalAll = "all"
	RetrievalAny = "any"
)

type SitemapSettings struct {
	BaseUrl    string `json:"base_url"`
	UrlField   string `json:"url_field"`
	ChangeFreq string `json:"changefreq"`
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
		FieldWeights:  make(map[string]float64),
		RetrievalMode: RetrievalAll,
		ScoringMode:   ScoringLegacy,
		BM25K1:        DefaultBM25K1,
		BM25B:         DefaultBM25B,
		Sitemap: SitemapSettings{
			ChangeFreq: "weekly",
		},
	}
}

// DeleteDocument removes a document from the index
func (idx *InvertedIndex) DeleteDocument(docID string) {
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.deleteDocumentLocked(docID)
}

// deleteDocumentLocked removes a document from the index. Caller must hold idx.mu.
func (idx *InvertedIndex) deleteDocumentLocked(docID string) {
	// 1. Get terms associated with this document for targeted removal
	terms, ok := idx.docToTerms[docID]
	if !ok {
		return
	}

	// 2. Remove from inverted index. terms is deduplicated, so each iteration
	// is one document leaving that term's posting list: exactly one decrement.
	for _, term := range terms {
		idx.docFreq[term]--
		if idx.docFreq[term] <= 0 {
			delete(idx.docFreq, term)
		}

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
	idx.totalLen -= idx.docLengths[docID]
	delete(idx.docLengths, docID)
	delete(idx.docs, docID)
	delete(idx.docToTerms, docID)
}

// DocFrequency returns how many documents contain a term.
func (idx *InvertedIndex) DocFrequency(term string) int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.docFreq[term]
}

// VocabularySize returns how many distinct terms the index holds: the cost of
// a scan of the vocabulary, as SubstringMatch does, grows with it.
func (idx *InvertedIndex) VocabularySize() int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return len(idx.index)
}

// DocLength returns a document's length in indexed tokens, occurrences
// included.
func (idx *InvertedIndex) DocLength(docID string) int {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.docLengths[docID]
}

// AverageDocLength returns the mean document length, 0 on an empty index.
// Callers inside Search must use averageDocLengthLocked instead: taking the
// read lock again can deadlock against a waiting writer.
func (idx *InvertedIndex) AverageDocLength() float64 {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	return idx.averageDocLengthLocked()
}

func (idx *InvertedIndex) averageDocLengthLocked() float64 {
	if len(idx.docLengths) == 0 {
		return 0
	}
	return float64(idx.totalLen) / float64(len(idx.docLengths))
}

func (idx *InvertedIndex) addToPrefixMap(prefix, term string) {
	for _, t := range idx.prefixMap[prefix] {
		if t == term {
			return
		}
	}
	idx.prefixMap[prefix] = append(idx.prefixMap[prefix], term)
}

// GetAllDocs returns all documents in the index. The outer map is a fresh copy
// safe to range over, but the inner document maps are shared with the index and
// MUST be treated as read-only (see GetDocument).
func (idx *InvertedIndex) GetAllDocs() map[string]map[string]interface{} {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	docsCopy := make(map[string]map[string]interface{})
	for k, v := range idx.docs {
		docsCopy[k] = v
	}
	return docsCopy
}
