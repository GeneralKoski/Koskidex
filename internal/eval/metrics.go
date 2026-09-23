package eval

import "math"

// Relevance judgments for a single query: document ID to relevance grade.
// A document missing from the map is unjudged and counts as 0.
type relevance = map[string]int

// NDCG computes nDCG@k the way trec_eval's ndcg_cut does, which is the number
// BEIR publishes and therefore the only one comparable with the references in
// eval/corpora/c1-public/SOURCE.md.
//
// Two details decide whether the number is comparable at all:
//
//   - the gain is LINEAR, the relevance value itself, not 2^rel-1. Verified
//     against trec_eval's m_ndcg_cut.c, which computes gain / log2(i+2) with
//     gain taken straight from the qrels. The two formulations agree on binary
//     judgments, so SciFact cannot tell them apart and NFCorpus can: getting
//     this wrong would have shifted only the graded collection, silently.
//   - unjudged documents score 0 but still consume a position, so padding the
//     ranking with junk is penalised rather than ignored.
//
// The ideal DCG comes from the best k judgments available for the query, so a
// query with fewer than k relevant documents can still reach 1.0. Returns 0
// when the query has no relevant document at all, since there is nothing a
// ranking could have done.
func NDCG(ranked []string, rel relevance, k int) float64 {
	if k <= 0 {
		return 0
	}

	var dcg float64
	for i, docID := range ranked {
		if i >= k {
			break
		}
		if g := rel[docID]; g > 0 {
			dcg += float64(g) / math.Log2(float64(i+2))
		}
	}

	var idcg float64
	for i, g := range topGrades(rel, k) {
		idcg += float64(g) / math.Log2(float64(i+2))
	}
	if idcg == 0 {
		return 0
	}
	return dcg / idcg
}

// Recall computes recall@k over the documents judged relevant, meaning grade
// greater than zero. Returns 0 when the query has none.
func Recall(ranked []string, rel relevance, k int) float64 {
	if k <= 0 {
		return 0
	}

	totali := CountRelevant(rel)
	if totali == 0 {
		return 0
	}

	var trovati int
	for i, docID := range ranked {
		if i >= k {
			break
		}
		if rel[docID] > 0 {
			trovati++
		}
	}
	return float64(trovati) / float64(totali)
}

// MRR is the reciprocal rank of the first relevant document within the first k
// positions, or 0 if none appears there.
func MRR(ranked []string, rel relevance, k int) float64 {
	if k <= 0 {
		return 0
	}
	for i, docID := range ranked {
		if i >= k {
			break
		}
		if rel[docID] > 0 {
			return 1 / float64(i+1)
		}
	}
	return 0
}

// CountRelevant counts the documents judged relevant, grade above zero.
// Judgments of exactly 0 are explicit statements of non-relevance and do not
// count.
func CountRelevant(rel relevance) int {
	var n int
	for _, g := range rel {
		if g > 0 {
			n++
		}
	}
	return n
}

// topGrades returns the k highest relevance grades, descending. It counts
// grades into buckets rather than sorting, because a query in NFCorpus can
// carry hundreds of judgments and the grades themselves are a handful of small
// integers.
func topGrades(rel relevance, k int) []int {
	var massimo int
	for _, g := range rel {
		if g > massimo {
			massimo = g
		}
	}
	if massimo == 0 {
		return nil
	}

	conteggi := make([]int, massimo+1)
	for _, g := range rel {
		if g > 0 {
			conteggi[g]++
		}
	}

	out := make([]int, 0, k)
	for g := massimo; g >= 1 && len(out) < k; g-- {
		for n := 0; n < conteggi[g] && len(out) < k; n++ {
			out = append(out, g)
		}
	}
	return out
}
