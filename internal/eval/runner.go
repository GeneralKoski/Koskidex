package eval

import (
	"encoding/json"
	"io"
	"runtime"
	"sort"
	"time"
)

// Searcher is whatever produces a ranking. Keeping the runner behind this
// interface is what lets the metrics be tested with no index involved, and
// what will let a future run compare two rankers without touching this file.
type Searcher interface {
	// Search returns document IDs, best first, at most k of them.
	Search(query string, k int) []string
}

// Cutoffs for the reported metrics. nDCG@10 is the number BEIR publishes and
// the one the references in SOURCE.md refer to.
const (
	NDCGCut   = 10
	RecallCut = 100
	MRRCut    = 10
)

// QueryResult is one query's scores, kept per query so a mean that moves can
// be traced back to which queries moved it.
type QueryResult struct {
	QueryID   string  `json:"query_id"`
	NDCG10    float64 `json:"ndcg@10"`
	Recall100 float64 `json:"recall@100"`
	MRR10     float64 `json:"mrr@10"`
	Retrieved int     `json:"retrieved"`
	Relevant  int     `json:"relevant"`
}

// Results is what gets written to a versioned file. Every number in the thesis
// has to come from one of these, never from a run nobody recorded.
type Results struct {
	Run           string        `json:"run"`
	Collection    string        `json:"collection"`
	RanAt         string        `json:"ran_at"`
	GoVersion     string        `json:"go_version"`
	Queries       int           `json:"queries"`
	SkippedNoRel  int           `json:"skipped_no_relevant"`
	ZeroResults   int           `json:"zero_results"`
	MeanNDCG10    float64       `json:"mean_ndcg@10"`
	MeanRecall100 float64       `json:"mean_recall@100"`
	MeanMRR10     float64       `json:"mean_mrr@10"`
	Elapsed       string        `json:"elapsed"`
	PerQuery      []QueryResult `json:"per_query"`
}

// Run scores a Searcher over every judged query and returns the means.
//
// Queries whose judgments contain no relevant document are skipped and
// counted, which is what trec_eval does: there is no ranking that could score
// above zero on them, so averaging them in would only dilute the result by an
// amount that depends on the collection rather than on the ranker.
//
// Queries are processed in sorted ID order so that two runs of the same
// configuration produce byte-identical files and a diff between two result
// files shows only what actually changed.
//
// ZeroResults counts the queries that came back empty. It is reported on its
// own because a mean cannot distinguish a ranker that orders badly from one
// that retrieves nothing, and the two call for opposite fixes.
func Run(s Searcher, nomeRun, collezione string, queries map[string]string, qrels Qrels) Results {
	inizio := time.Now()

	ids := make([]string, 0, len(queries))
	for id := range queries {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	// Retrieve as deep as the deepest cutoff, once, and score every metric off
	// the same ranking: asking the engine twice could give two different lists.
	profondita := RecallCut
	if NDCGCut > profondita {
		profondita = NDCGCut
	}

	out := Results{
		Run:        nomeRun,
		Collection: collezione,
		RanAt:      time.Now().UTC().Format(time.RFC3339),
		GoVersion:  runtime.Version(),
		PerQuery:   make([]QueryResult, 0, len(ids)),
	}

	var sommaNDCG, sommaRecall, sommaMRR float64
	for _, qid := range ids {
		rel := qrels[qid]
		rilevanti := CountRelevant(rel)
		if rilevanti == 0 {
			out.SkippedNoRel++
			continue
		}

		ranked := s.Search(queries[qid], profondita)
		if len(ranked) == 0 {
			out.ZeroResults++
		}

		q := QueryResult{
			QueryID:   qid,
			NDCG10:    NDCG(ranked, rel, NDCGCut),
			Recall100: Recall(ranked, rel, RecallCut),
			MRR10:     MRR(ranked, rel, MRRCut),
			Retrieved: len(ranked),
			Relevant:  rilevanti,
		}
		out.PerQuery = append(out.PerQuery, q)

		sommaNDCG += q.NDCG10
		sommaRecall += q.Recall100
		sommaMRR += q.MRR10
	}

	out.Queries = len(out.PerQuery)
	if out.Queries > 0 {
		n := float64(out.Queries)
		out.MeanNDCG10 = sommaNDCG / n
		out.MeanRecall100 = sommaRecall / n
		out.MeanMRR10 = sommaMRR / n
	}
	out.Elapsed = time.Since(inizio).Round(time.Millisecond).String()

	return out
}

// WriteJSON writes the results in a stable, readable layout, so two files can
// be diffed line by line.
func (r Results) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}
