package eval

import (
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"
)

// ExternalRun is a ranking produced outside Koskidex: Elasticsearch, as
// Documentale's app:eval-run-queries records it, with one list of corpus IDs
// per query text.
//
// It exists so that the pool is not built from Koskidex alone. A document only
// Elasticsearch retrieves would otherwise never be judged, would count as not
// relevant, and the comparison would start rigged against Elasticsearch.
type ExternalRun struct {
	Name    string
	ByQuery map[string][]string
	// Ms is the time the app measured for each query, in milliseconds.
	Ms map[string]float64
}

// LoadExternalRun reads a report of app:eval-run-queries.
func LoadExternalRun(path string) (ExternalRun, error) {
	dati, err := os.ReadFile(path)
	if err != nil {
		return ExternalRun{}, err
	}
	var r struct {
		Config struct {
			Motore string `json:"motore"`
		} `json:"config"`
		Query []struct {
			Query string    `json:"query"`
			IDs   *[]string `json:"ids"`
			Ms    float64   `json:"ms"`
		} `json:"query"`
	}
	if err := json.Unmarshal(dati, &r); err != nil {
		return ExternalRun{}, fmt.Errorf("%s: %w", path, err)
	}
	if len(r.Query) == 0 {
		return ExternalRun{}, fmt.Errorf("%s: nessuna query, non sembra un rapporto di app:eval-run-queries", path)
	}

	run := ExternalRun{Name: r.Config.Motore, ByQuery: make(map[string][]string, len(r.Query)),
		Ms: make(map[string]float64, len(r.Query))}
	for _, q := range r.Query {
		testo := strings.TrimSpace(q.Query)
		// Un ranking vuoto e un ranking assente non sono la stessa cosa: il
		// secondo vuol dire che il file non e' un rapporto dell'app.
		if q.IDs == nil {
			return ExternalRun{}, fmt.Errorf("%s: la query %q non ha ids, non sembra un rapporto di app:eval-run-queries", path, testo)
		}
		if prima, visto := run.ByQuery[testo]; visto && !slices.Equal(prima, *q.IDs) {
			return ExternalRun{}, fmt.Errorf("%s: la query %q compare due volte con due ranking diversi", path, testo)
		}
		run.ByQuery[testo] = *q.IDs
		run.Ms[testo] = q.Ms
	}

	return run, nil
}

// Align gives each query ID the ranking the run produced for its text.
//
// It refuses a query the run did not execute, and a document ID that is not in
// the corpus: both would leave the pool silently incomplete.
func (r ExternalRun) Align(queries map[string]string, corpus map[string]bool) (map[string][]string, error) {
	qids := make([]string, 0, len(queries))
	for qid := range queries {
		qids = append(qids, qid)
	}
	sort.Strings(qids)

	out := make(map[string][]string, len(queries))
	var mancanti []string
	for _, qid := range qids {
		ranking, ok := r.ByQuery[strings.TrimSpace(queries[qid])]
		if !ok {
			mancanti = append(mancanti, fmt.Sprintf("%s %q", qid, queries[qid]))

			continue
		}
		for _, id := range ranking {
			if !corpus[id] {
				return nil, fmt.Errorf("%s: la query %s ha il documento %q, che non sta nel corpus (gli id devono essere quelli del corpus, come doc-0001)", r.Name, qid, id)
			}
		}
		out[qid] = ranking
	}
	if len(mancanti) > 0 {
		return nil, fmt.Errorf("%s: %d query non eseguite, la prima e' %s", r.Name, len(mancanti), mancanti[0])
	}

	return out, nil
}

// Search makes a recorded run a Searcher, so Run scores it with the same code
// as Koskidex. A query the run did not execute comes back empty: Align is the
// place that refuses it.
func (r ExternalRun) Search(query string, k int) ([]string, int) {
	ids := r.ByQuery[strings.TrimSpace(query)]
	if len(ids) > k {
		return ids[:k], len(ids)
	}

	return ids, len(ids)
}
