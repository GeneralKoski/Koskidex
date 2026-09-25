package eval

import (
	"strings"

	"github.com/GeneralKoski/Koskidex/internal/engine"
)

// campoUnico is the single searchable field the evaluation corpus is indexed
// into. The published references are "flat" runs: title and text concatenated
// into one field. Indexing them separately with different weights, as the
// frozen baseline corpus does, would make the number incomparable.
const campoUnico = "text"

// KoskidexSearcher adapts the engine to the Searcher interface, configured so
// that what is measured is comparable with the published BM25 references.
//
// Two settings are not negotiable and are enforced here rather than left to
// the caller, because both fail silently: typo tolerance is switched off, and
// title and text go into one field with one weight. See
// eval/corpora/c1-public/SOURCE.md.
type KoskidexSearcher struct {
	idx      *engine.InvertedIndex
	settings engine.Settings
	vettori  Vettori
}

// Vettori are the embeddings of a run with vectors: one per document, given
// to the engine as _vector, and one per query text, passed with the search as
// a client passes it over HTTP. Computing them is the caller's job, so the
// engine sees exactly what it would see from Documentale.
type Vettori struct {
	Documenti map[string][]float64
	Query     map[string][]float64
}

// TestoDocumento is the text a document is indexed with, and embedded from:
// title and text in one field.
func TestoDocumento(d Document) string {
	return strings.TrimSpace(d.Title + " " + d.Text)
}

// NewKoskidexSearcher indexes the corpus and returns a searcher over it.
// Applying extra settings, such as switching a ranking flag on, is done
// through modifica, which runs before the documents are indexed.
func NewKoskidexSearcher(docs []Document, modifica func(*engine.Settings)) *KoskidexSearcher {
	return NewKoskidexSearcherConVettori(docs, modifica, Vettori{})
}

// NewKoskidexSearcherConVettori is NewKoskidexSearcher with vectors: a
// document or a query without one is searched as without vectors.
func NewKoskidexSearcherConVettori(docs []Document, modifica func(*engine.Settings), vettori Vettori) *KoskidexSearcher {
	s := engine.DefaultSettings()
	s.SearchableFields = []string{campoUnico}
	s.FieldWeights = map[string]float64{campoUnico: 1.0}
	s.TypoTolerance.Enabled = false

	if modifica != nil {
		modifica(&s)
	}

	idx := engine.NewInvertedIndex()
	for _, d := range docs {
		doc := map[string]interface{}{campoUnico: TestoDocumento(d)}
		if v, ok := vettori.Documenti[d.ID]; ok {
			doc["_vector"] = v
		}
		idx.AddDocument(d.ID, doc, s)
	}

	return &KoskidexSearcher{idx: idx, settings: s, vettori: vettori}
}

// Search returns at most k document IDs, best first, and how many documents
// matched in total.
//
// Fuzziness is pinned to "0", which MaxTypos treats as an explicit zero
// regardless of the settings. Relying on TypoTolerance.Enabled alone would not
// be enough: an explicit fuzziness overrides it, so the two are set together.
func (k *KoskidexSearcher) Search(query string, limite int) ([]string, int) {
	ids, _ := k.idx.Search(query, k.settings, "0", k.vettori.Query[query])
	trovati := len(ids)
	if trovati > limite {
		return ids[:limite], trovati
	}
	return ids, trovati
}

// Settings exposes the configuration actually used, so a run can record it
// instead of assuming it.
func (k *KoskidexSearcher) Settings() engine.Settings {
	return k.settings
}
