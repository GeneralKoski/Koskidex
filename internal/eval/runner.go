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
	// Search returns document IDs, best first, at most k of them, and the
	// total number of documents that matched before the cutoff.
	//
	// The total is returned alongside instead of being derivable from the
	// slice because once the slice is truncated the information is gone, and
	// it is the only order-independent way to check that a ranking change did
	// not also change what gets retrieved.
	Search(query string, k int) (ids []string, matched int)
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
	// Candidates is how many documents matched in total, before the cutoff.
	Candidates int `json:"candidates"`
	Relevant   int `json:"relevant"`
	// Top is the head of the ranking, kept only when asked for with RunTop.
	Top []string `json:"top,omitempty"`
}

// Results is what gets written to a versioned file. Every number in the thesis
// has to come from one of these, never from a run nobody recorded.
type Results struct {
	Run string `json:"run"`
	// Config is every setting that shaped the run, plus where the code came
	// from. The run label alone is not enough: two files called
	// scifact-bm25.json written on different days are not the same run.
	Config        map[string]string `json:"config,omitempty"`
	Collection    string            `json:"collection"`
	RanAt         string            `json:"ran_at"`
	GoVersion     string            `json:"go_version"`
	Queries       int               `json:"queries"`
	SkippedNoRel  int               `json:"skipped_no_relevant"`
	ZeroResults   int               `json:"zero_results"`
	MeanNDCG10    float64           `json:"mean_ndcg@10"`
	MeanRecall100 float64           `json:"mean_recall@100"`
	MeanMRR10     float64           `json:"mean_mrr@10"`
	Elapsed       string            `json:"elapsed"`
	PerQuery      []QueryResult     `json:"per_query"`
	// Timings are kept apart from PerQuery on purpose. Latencies change on
	// every run; the metrics must not. With the two mixed, a diff between two
	// result files would be all noise and would hide the one score that moved.
	Timings Timings `json:"timings"`
}

// Timings are in milliseconds, as numbers, so they can be plotted without
// parsing. IndexMs is filled in by the caller, who owns the indexing.
type Timings struct {
	IndexMs    float64            `json:"index_ms"`
	ElapsedMs  float64            `json:"elapsed_ms"`
	PerQueryMs map[string]float64 `json:"per_query_ms"`
}

// Nanosecond precision: Microseconds() truncates, and a query faster than a
// microsecond would be recorded as taking zero time.
func millisecondi(d time.Duration) float64 {
	return float64(d.Nanoseconds()) / 1e6
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
	return RunTop(s, nomeRun, collezione, queries, qrels, 0)
}

// RunTop is Run that also keeps the first top documents of every ranking, to
// see which documents overtake the relevant one. With top 0 it is Run.
func RunTop(s Searcher, nomeRun, collezione string, queries map[string]string, qrels Qrels, top int) Results {
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
		Timings:    Timings{PerQueryMs: make(map[string]float64, len(ids))},
	}

	var sommaNDCG, sommaRecall, sommaMRR float64
	for _, qid := range ids {
		rel := qrels[qid]
		rilevanti := CountRelevant(rel)
		if rilevanti == 0 {
			out.SkippedNoRel++
			continue
		}

		t0 := time.Now()
		ranked, candidati := s.Search(queries[qid], profondita)
		out.Timings.PerQueryMs[qid] = millisecondi(time.Since(t0))
		if len(ranked) == 0 {
			out.ZeroResults++
		}

		q := QueryResult{
			QueryID:    qid,
			NDCG10:     NDCG(ranked, rel, NDCGCut),
			Recall100:  Recall(ranked, rel, RecallCut),
			MRR10:      MRR(ranked, rel, MRRCut),
			Retrieved:  len(ranked),
			Candidates: candidati,
			Relevant:   rilevanti,
		}
		if top > 0 {
			q.Top = append([]string{}, ranked[:min(top, len(ranked))]...)
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
	out.Timings.ElapsedMs = millisecondi(time.Since(inizio))

	return out
}

// WriteJSON writes the results in a stable, readable layout, so two files can
// be diffed line by line.
func (r Results) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}
