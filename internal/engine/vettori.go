package engine

import "sort"

type vicino struct {
	id  string
	sim float64
}

// viciniLocked returns the k documents closest to the query vector by cosine
// similarity, closest first, ties by id. It scans every vector: O(n·d), with no
// approximate index, so its cost is what the thesis measures. A document with
// no vector, or with one of another dimension, is skipped, as in the
// re-ranking. The caller holds idx.mu.
func (idx *InvertedIndex) viciniLocked(q []float64, k int) []vicino {
	tutti := make([]vicino, 0, len(idx.docs))
	for id, doc := range idx.docs {
		v, ok := doc["_vector"]
		if !ok {
			continue
		}
		dv, ok := toFloat64Array(v)
		if !ok || len(dv) != len(q) {
			continue
		}
		tutti = append(tutti, vicino{id: id, sim: cosineSimilarity(q, dv)})
	}
	sort.Slice(tutti, func(i, j int) bool {
		if tutti[i].sim != tutti[j].sim {
			return tutti[i].sim > tutti[j].sim
		}
		return tutti[i].id < tutti[j].id
	})
	if len(tutti) > k {
		tutti = tutti[:k]
	}
	return tutti
}
