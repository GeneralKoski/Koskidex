package eval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func scrivi(t *testing.T, nome, contenuto string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), nome)
	if err := os.WriteFile(path, []byte(contenuto), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadCorpus(t *testing.T) {
	path := scrivi(t, "corpus.jsonl", `{"_id":"d1","title":"Primo","text":"testo uno","metadata":{"url":"x"}}
{"_id":"d2","title":"Secondo","text":"testo due"}
`)
	docs, err := LoadCorpus(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 {
		t.Fatalf("attesi 2 documenti, ottenuti %d", len(docs))
	}
	if docs[0].ID != "d1" || docs[0].Title != "Primo" || docs[0].Text != "testo uno" {
		t.Fatalf("primo documento letto male: %+v", docs[0])
	}
}

// Blank lines must not become phantom documents.
func TestLoadCorpusSkipsBlankLines(t *testing.T) {
	path := scrivi(t, "corpus.jsonl", "{\"_id\":\"d1\",\"text\":\"a\"}\n\n\n{\"_id\":\"d2\",\"text\":\"b\"}\n")
	docs, err := LoadCorpus(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 2 {
		t.Fatalf("attesi 2 documenti, ottenuti %d", len(docs))
	}
}

// A document with no _id is a corrupt corpus, not something to index anyway:
// it would be unmatchable against any judgment.
func TestLoadCorpusRejectsMissingID(t *testing.T) {
	path := scrivi(t, "corpus.jsonl", `{"title":"senza id","text":"a"}`+"\n")
	if _, err := LoadCorpus(path); err == nil {
		t.Fatal("atteso errore per documento senza _id")
	}
}

// bufio.Scanner stops at 64 KB by default and reports no error. A long
// document has to come through whole, not be silently dropped.
func TestLoadCorpusHandlesLongLines(t *testing.T) {
	lungo := strings.Repeat("parola ", 40000) // ~280 KB
	path := scrivi(t, "corpus.jsonl", `{"_id":"d1","text":"`+lungo+`"}`+"\n")
	docs, err := LoadCorpus(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Fatalf("atteso 1 documento, ottenuti %d", len(docs))
	}
	if len(docs[0].Text) < 200000 {
		t.Fatalf("il testo e' stato troncato: %d byte", len(docs[0].Text))
	}
}

func TestLoadQrels(t *testing.T) {
	path := scrivi(t, "test.tsv", "query-id\tcorpus-id\tscore\nq1\td1\t2\nq1\td2\t1\nq2\td3\t1\n")
	qrels, err := LoadQrels(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(qrels) != 2 {
		t.Fatalf("attese 2 query, ottenute %d", len(qrels))
	}
	if qrels["q1"]["d1"] != 2 || qrels["q1"]["d2"] != 1 || qrels["q2"]["d3"] != 1 {
		t.Fatalf("giudizi letti male: %v", qrels)
	}
}

// A zero judgment says "this document is not relevant", which is not the same
// as saying nothing about it. It has to survive the read.
func TestLoadQrelsKeepsZeroJudgments(t *testing.T) {
	path := scrivi(t, "test.tsv", "query-id\tcorpus-id\tscore\nq1\td1\t0\n")
	qrels, err := LoadQrels(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, giudicato := qrels["q1"]["d1"]; !giudicato {
		t.Fatal("il giudizio 0 e' stato scartato")
	}
}

func TestLoadQrelsRejectsMalformedLines(t *testing.T) {
	casi := map[string]string{
		"due colonne":       "query-id\tcorpus-id\tscore\nq1\td1\n",
		"rilevanza non int": "query-id\tcorpus-id\tscore\nq1\td1\tmolto\n",
		"separato da spazi": "query-id\tcorpus-id\tscore\nq1 d1 1\n",
	}
	for nome, contenuto := range casi {
		t.Run(nome, func(t *testing.T) {
			if _, err := LoadQrels(scrivi(t, "test.tsv", contenuto)); err == nil {
				t.Fatal("atteso errore")
			}
		})
	}
}

// A file with only a header means the wrong split was picked up, which would
// otherwise show up as a mean over zero queries.
func TestLoadQrelsRejectsEmptyFile(t *testing.T) {
	path := scrivi(t, "test.tsv", "query-id\tcorpus-id\tscore\n")
	if _, err := LoadQrels(path); err == nil {
		t.Fatal("atteso errore per file senza giudizi")
	}
}

func TestQueriesToEvaluateKeepsOnlyJudgedQueries(t *testing.T) {
	queries := map[string]string{"q1": "prima", "q2": "seconda", "q3": "senza giudizi"}
	qrels := Qrels{"q1": {"d1": 1}, "q2": {"d2": 1}}

	da, err := QueriesToEvaluate(queries, qrels)
	if err != nil {
		t.Fatal(err)
	}
	if len(da) != 2 {
		t.Fatalf("attese 2 query, ottenute %d: %v", len(da), da)
	}
	if _, c := da["q3"]; c {
		t.Fatal("q3 non ha giudizi e non deve essere valutata")
	}
}

// A judgment pointing at a query with no text means corpus and qrels do not
// belong together. Scoring it as zero would quietly lower every mean.
func TestQueriesToEvaluateRejectsJudgedQueryWithoutText(t *testing.T) {
	_, err := QueriesToEvaluate(map[string]string{"q1": "prima"}, Qrels{"q1": {"d1": 1}, "q9": {"d2": 1}})
	if err == nil {
		t.Fatal("atteso errore per query giudicata ma senza testo")
	}
}
