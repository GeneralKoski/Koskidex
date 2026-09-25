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
	ordina(tutti)
	if len(tutti) > k {
		tutti = tutti[:k]
	}
	return tutti
}

// ordina sorts by value, highest first, ties by id.
func ordina(l []vicino) {
	sort.Slice(l, func(i, j int) bool {
		if l[i].sim != l[j].sim {
			return l[i].sim > l[j].sim
		}
		return l[i].id < l[j].id
	})
}

func (s Settings) pesoVettore() float64 {
	if s.VectorWeight != nil {
		return *s.VectorWeight
	}
	return 20
}

func (s Settings) alfaFusione() float64 {
	if s.FusionAlpha != nil {
		return *s.FusionAlpha
	}
	return 0.5
}

func (s Settings) vettoriK() int {
	if s.VectorTopK > 0 {
		return s.VectorTopK
	}
	return vectorTopKDefault
}

// fondiLocked replaces the lexical scores in docMatches with the fusion of
// the two lists: the lexical candidates by score, and the VectorTopK documents
// closest to the query vector, which join the candidates. The caller holds
// idx.mu.
func (idx *InvertedIndex) fondiLocked(docMatches map[string]*SearchMatch, q []float64, settings Settings) {
	lessicali := make([]vicino, 0, len(docMatches))
	for id, m := range docMatches {
		lessicali = append(lessicali, vicino{id: id, sim: m.Score})
	}
	ordina(lessicali)
	vettoriali := idx.viciniLocked(q, settings.vettoriK())

	fusi := make(map[string]float64, len(lessicali)+len(vettoriali))
	switch settings.FusionMode {
	case FusionRRF:
		for i, v := range lessicali {
			fusi[v.id] += 1 / float64(rrfK+i+1)
		}
		for i, v := range vettoriali {
			fusi[v.id] += 1 / float64(rrfK+i+1)
		}
	case FusionConvex:
		a := settings.alfaFusione()
		for id, x := range normalizza(lessicali) {
			fusi[id] += a * x
		}
		for id, x := range normalizza(vettoriali) {
			fusi[id] += (1 - a) * x
		}
	}
	for id, punteggio := range fusi {
		if m, ok := docMatches[id]; ok {
			m.Score = punteggio
		} else {
			docMatches[id] = &SearchMatch{DocID: id, Score: punteggio}
		}
	}
}

// normalizza maps a list min-max onto [0, 1]. When every value is the same,
// a single candidate included, each gets 1: there is nothing to tell them
// apart, and being in the list is all the list says.
func normalizza(l []vicino) map[string]float64 {
	out := make(map[string]float64, len(l))
	if len(l) == 0 {
		return out
	}
	massimo, minimo := l[0].sim, l[len(l)-1].sim
	for _, v := range l {
		if massimo == minimo {
			out[v.id] = 1
		} else {
			out[v.id] = (v.sim - minimo) / (massimo - minimo)
		}
	}
	return out
}
