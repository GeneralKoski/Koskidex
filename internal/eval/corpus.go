// Package eval reads IR evaluation collections and scores rankings against
// their relevance judgments.
//
// It is deliberately independent of the search engine: it talks to a Searcher
// interface, so the metrics can be tested on hand-computed numbers with no
// index involved.
//
// The on-disk format is BEIR's. See eval/corpora/c1-public/SOURCE.md.
package eval

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// maxLineBytes is the per-line ceiling for the JSONL readers. bufio.Scanner
// stops at 64 KB by default and reports no error, which would silently drop
// long documents; 8 MB is far beyond any realistic single document.
const maxLineBytes = 8 << 20

// Document is one entry of corpus.jsonl.
type Document struct {
	ID    string `json:"_id"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

// Qrels maps a query ID to the relevance of each judged document.
// Documents absent from the map are unjudged and count as relevance 0.
type Qrels map[string]map[string]int

func scannerFor(f *os.File) *bufio.Scanner {
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), maxLineBytes)
	return sc
}

// LoadCorpus reads a BEIR corpus.jsonl.
func LoadCorpus(path string) ([]Document, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var docs []Document
	sc := scannerFor(f)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var d Document
		if err := json.Unmarshal([]byte(line), &d); err != nil {
			return nil, fmt.Errorf("%s riga %d: %w", path, n, err)
		}
		if d.ID == "" {
			return nil, fmt.Errorf("%s riga %d: documento senza _id", path, n)
		}
		docs = append(docs, d)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return docs, nil
}

// LoadQueries reads a BEIR queries.jsonl into a map from query ID to text.
//
// Careful: the file holds every split. For NFCorpus that is 3,237 queries
// against 323 test ones, so the set to evaluate comes from the qrels, not
// from here.
func LoadQueries(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	queries := make(map[string]string)
	sc := scannerFor(f)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var q struct {
			ID   string `json:"_id"`
			Text string `json:"text"`
		}
		if err := json.Unmarshal([]byte(line), &q); err != nil {
			return nil, fmt.Errorf("%s riga %d: %w", path, n, err)
		}
		if q.ID == "" {
			return nil, fmt.Errorf("%s riga %d: query senza _id", path, n)
		}
		queries[q.ID] = q.Text
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return queries, nil
}

// LoadQrels reads a BEIR qrels TSV: a header line, then
// query-id <TAB> corpus-id <TAB> score.
//
// Judgments with relevance 0 are kept: they are explicit statements that a
// document is not relevant, and dropping them would make an unjudged document
// indistinguishable from one judged irrelevant.
func LoadQrels(path string) (Qrels, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	qrels := make(Qrels)
	sc := scannerFor(f)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		campi := strings.Split(line, "\t")
		if len(campi) != 3 {
			return nil, fmt.Errorf("%s riga %d: attese 3 colonne separate da tab, trovate %d", path, n, len(campi))
		}
		if n == 1 && campi[0] == "query-id" {
			continue // intestazione
		}
		rel, err := strconv.Atoi(campi[2])
		if err != nil {
			return nil, fmt.Errorf("%s riga %d: rilevanza %q non e' un intero", path, n, campi[2])
		}
		if qrels[campi[0]] == nil {
			qrels[campi[0]] = make(map[string]int)
		}
		qrels[campi[0]][campi[1]] = rel
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if len(qrels) == 0 {
		return nil, fmt.Errorf("%s: nessun giudizio letto", path)
	}
	return qrels, nil
}

// QueriesToEvaluate keeps only the queries that have at least one judgment,
// and fails if one of them has no text: evaluating a query with no judgments
// would score a guaranteed zero and drag the mean down for no reason.
func QueriesToEvaluate(queries map[string]string, qrels Qrels) (map[string]string, error) {
	out := make(map[string]string, len(qrels))
	var mancanti []string
	for qid := range qrels {
		text, ok := queries[qid]
		if !ok {
			mancanti = append(mancanti, qid)
			continue
		}
		out[qid] = text
	}
	if len(mancanti) > 0 {
		return nil, fmt.Errorf("%d query hanno giudizi ma non un testo, la prima e' %q", len(mancanti), mancanti[0])
	}
	return out, nil
}
