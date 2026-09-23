package eval

import (
	"os"
	"path/filepath"
	"testing"
)

// The real collections are not in git: they come from eval/corpora/fetch.sh.
// Without them these tests skip, so a fresh clone stays green.
func corpusDir(t *testing.T, nome string) string {
	t.Helper()
	dir := filepath.Join("..", "..", "eval", "corpora", "c1-public", nome)
	if _, err := os.Stat(dir); err != nil {
		t.Skipf("%s non scaricato, lancia eval/corpora/fetch.sh", nome)
	}
	return dir
}

// Counts checked by hand against the downloaded files on 2026-09-23 and
// written into eval/corpora/c1-public/SOURCE.md. If one of these moves, either
// the loader broke or the collection changed at the source, and both are
// things to find out before measuring anything.
func TestRealCollectionsMatchTheRecordedCounts(t *testing.T) {
	casi := []struct {
		nome             string
		documenti        int
		queryTotali      int
		queryGiudicate   int
		giudizi          int
		relevanzaMassima int
	}{
		{"scifact", 5183, 1109, 300, 339, 1},
		{"nfcorpus", 3633, 3237, 323, 12334, 2},
	}

	for _, c := range casi {
		t.Run(c.nome, func(t *testing.T) {
			dir := corpusDir(t, c.nome)

			docs, err := LoadCorpus(filepath.Join(dir, "corpus.jsonl"))
			if err != nil {
				t.Fatal(err)
			}
			if len(docs) != c.documenti {
				t.Errorf("documenti: attesi %d, letti %d", c.documenti, len(docs))
			}

			queries, err := LoadQueries(filepath.Join(dir, "queries.jsonl"))
			if err != nil {
				t.Fatal(err)
			}
			if len(queries) != c.queryTotali {
				t.Errorf("query nel file: attese %d, lette %d", c.queryTotali, len(queries))
			}

			qrels, err := LoadQrels(filepath.Join(dir, "qrels", "test.tsv"))
			if err != nil {
				t.Fatal(err)
			}
			if len(qrels) != c.queryGiudicate {
				t.Errorf("query giudicate: attese %d, lette %d", c.queryGiudicate, len(qrels))
			}

			var giudizi, massima int
			for _, perDoc := range qrels {
				for _, rel := range perDoc {
					giudizi++
					if rel > massima {
						massima = rel
					}
				}
			}
			if giudizi != c.giudizi {
				t.Errorf("giudizi: attesi %d, letti %d", c.giudizi, giudizi)
			}
			if massima != c.relevanzaMassima {
				t.Errorf("rilevanza massima: attesa %d, letta %d", c.relevanzaMassima, massima)
			}

			// This is the trap SOURCE.md warns about: for NFCorpus the query
			// file holds ten times the queries that have judgments.
			daValutare, err := QueriesToEvaluate(queries, qrels)
			if err != nil {
				t.Fatal(err)
			}
			if len(daValutare) != c.queryGiudicate {
				t.Errorf("query da valutare: attese %d, ottenute %d", c.queryGiudicate, len(daValutare))
			}
		})
	}
}

// Every judged document has to exist in the corpus, otherwise recall has a
// ceiling below 1 and no ranking can ever reach it.
func TestEveryJudgedDocumentExistsInTheCorpus(t *testing.T) {
	for _, nome := range []string{"scifact", "nfcorpus"} {
		t.Run(nome, func(t *testing.T) {
			dir := corpusDir(t, nome)

			docs, err := LoadCorpus(filepath.Join(dir, "corpus.jsonl"))
			if err != nil {
				t.Fatal(err)
			}
			esiste := make(map[string]bool, len(docs))
			for _, d := range docs {
				esiste[d.ID] = true
			}

			qrels, err := LoadQrels(filepath.Join(dir, "qrels", "test.tsv"))
			if err != nil {
				t.Fatal(err)
			}

			var mancanti int
			var esempio string
			for qid, perDoc := range qrels {
				for docID, rel := range perDoc {
					if rel > 0 && !esiste[docID] {
						mancanti++
						if esempio == "" {
							esempio = qid + " -> " + docID
						}
					}
				}
			}
			if mancanti > 0 {
				t.Errorf("%d documenti giudicati rilevanti non sono nel corpus, il primo e' %s", mancanti, esempio)
			}
		})
	}
}
