package eval

import (
	"bytes"
	"encoding/json"
	"testing"
)

// searcherFinto returns a fixed ranking per query, so the runner can be tested
// with no index and with results known in advance.
type searcherFinto struct {
	perQuery map[string][]string
	chiamate []string
	ultimoK  int
}

func (s *searcherFinto) Search(query string, k int) []string {
	s.chiamate = append(s.chiamate, query)
	s.ultimoK = k
	r := s.perQuery[query]
	if len(r) > k {
		return r[:k]
	}
	return r
}

func TestRunComputesMeansOverJudgedQueries(t *testing.T) {
	queries := map[string]string{"q1": "prima", "q2": "seconda"}
	qrels := Qrels{
		"q1": {"d1": 1},
		"q2": {"d2": 1},
	}
	s := &searcherFinto{perQuery: map[string][]string{
		"prima":   {"d1", "x"}, // rilevante al primo posto  -> nDCG 1, MRR 1
		"seconda": {"x", "d2"}, // rilevante al secondo posto -> MRR 0.5
	}}

	r := Run(s, "prova", "finta", queries, qrels)

	if r.Queries != 2 {
		t.Fatalf("attese 2 query valutate, ottenute %d", r.Queries)
	}
	// nDCG: 1.0 e 1/log2(3) = 0.6309297535714574 -> media 0.8154648767857287
	vicino(t, r.MeanNDCG10, (1.0+1/1.5849625007211562)/2)
	vicino(t, r.MeanMRR10, (1.0+0.5)/2)
	vicino(t, r.MeanRecall100, 1.0)
}

// trec_eval skips queries with no relevant document. Averaging them in would
// dilute every mean by an amount that depends on the collection instead of on
// the ranker.
func TestRunSkipsQueriesWithNoRelevantDocument(t *testing.T) {
	queries := map[string]string{"q1": "prima", "q2": "seconda"}
	qrels := Qrels{
		"q1": {"d1": 1},
		"q2": {"d2": 0, "d3": 0}, // giudicate, ma nessuna rilevante
	}
	s := &searcherFinto{perQuery: map[string][]string{"prima": {"d1"}}}

	r := Run(s, "prova", "finta", queries, qrels)

	if r.Queries != 1 {
		t.Fatalf("attesa 1 query valutata, ottenute %d", r.Queries)
	}
	if r.SkippedNoRel != 1 {
		t.Fatalf("attesa 1 query saltata, ottenute %d", r.SkippedNoRel)
	}
	// Se q2 fosse entrata nella media, questa sarebbe 0.5 invece di 1.
	vicino(t, r.MeanNDCG10, 1.0)
}

// Two runs of the same configuration have to produce the same file, otherwise
// a diff between result files shows noise instead of changes.
func TestRunIsDeterministicInQueryOrder(t *testing.T) {
	queries := map[string]string{}
	qrels := Qrels{}
	ranking := map[string][]string{}
	for _, id := range []string{"q5", "q1", "q9", "q3", "q7"} {
		queries[id] = "testo " + id
		qrels[id] = map[string]int{"d" + id: 1}
		ranking["testo "+id] = []string{"d" + id}
	}

	var primo []string
	for giro := 0; giro < 5; giro++ {
		r := Run(&searcherFinto{perQuery: ranking}, "prova", "finta", queries, qrels)
		var ordine []string
		for _, q := range r.PerQuery {
			ordine = append(ordine, q.QueryID)
		}
		if giro == 0 {
			primo = ordine
			atteso := []string{"q1", "q3", "q5", "q7", "q9"}
			for i := range atteso {
				if ordine[i] != atteso[i] {
					t.Fatalf("atteso ordine %v, ottenuto %v", atteso, ordine)
				}
			}
			continue
		}
		for i := range primo {
			if ordine[i] != primo[i] {
				t.Fatalf("giro %d: ordine diverso, %v contro %v", giro, ordine, primo)
			}
		}
	}
}

// The engine must be asked once, deep enough for the deepest cutoff, and every
// metric scored off that one ranking: two calls could return two lists.
func TestRunAsksOnceAndDeepEnoughForRecall(t *testing.T) {
	s := &searcherFinto{perQuery: map[string][]string{"prima": {"d1"}}}
	Run(s, "prova", "finta", map[string]string{"q1": "prima"}, Qrels{"q1": {"d1": 1}})

	if len(s.chiamate) != 1 {
		t.Fatalf("attesa 1 chiamata al motore, ottenute %d", len(s.chiamate))
	}
	if s.ultimoK < RecallCut {
		t.Fatalf("il motore e' stato interrogato a profondita' %d, insufficiente per recall@%d", s.ultimoK, RecallCut)
	}
}

func TestRunWithNoQueriesDoesNotDivideByZero(t *testing.T) {
	r := Run(&searcherFinto{}, "prova", "finta", map[string]string{}, Qrels{})
	if r.Queries != 0 || r.MeanNDCG10 != 0 {
		t.Fatalf("attesi zeri, ottenuto %+v", r)
	}
}

func TestWriteJSONRoundTrips(t *testing.T) {
	r := Run(&searcherFinto{perQuery: map[string][]string{"prima": {"d1"}}},
		"legacy", "scifact", map[string]string{"q1": "prima"}, Qrels{"q1": {"d1": 1}})

	var buf bytes.Buffer
	if err := r.WriteJSON(&buf); err != nil {
		t.Fatal(err)
	}

	var riletto Results
	if err := json.Unmarshal(buf.Bytes(), &riletto); err != nil {
		t.Fatalf("il file prodotto non si rilegge: %v", err)
	}
	if riletto.Run != "legacy" || riletto.Collection != "scifact" {
		t.Fatalf("metadati persi: %+v", riletto)
	}
	if riletto.GoVersion == "" || riletto.RanAt == "" {
		t.Fatal("il file deve dire con quale Go e quando e' stato prodotto")
	}
	if len(riletto.PerQuery) != 1 {
		t.Fatalf("attesa 1 query nel file, ottenute %d", len(riletto.PerQuery))
	}
}
