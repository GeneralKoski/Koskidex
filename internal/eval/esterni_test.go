package eval

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func rapporto(t *testing.T, contenuto string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "rapporto.json")
	if err := os.WriteFile(path, []byte(contenuto), 0o600); err != nil {
		t.Fatal(err)
	}

	return path
}

// Il formato di app:eval-run-queries di Documentale: una voce per riga del file
// di query, con gli id gia' tradotti in quelli del corpus.
const rapportoElasticsearch = `{
  "ran_at": "2026-09-25T09:00:00Z",
  "config": {"motore": "elasticsearch", "label": "albo-metadata"},
  "query": [
    {"query": "determina 1223 crispiano", "ids": ["doc-0002", "doc-0007"], "ms": 12.5},
    {"query": "ordinanza 187 ", "ids": [], "ms": 8.1}
  ]
}`

func TestExternalRunReadsAnAppReport(t *testing.T) {
	run, err := LoadExternalRun(rapporto(t, rapportoElasticsearch))
	if err != nil {
		t.Fatal(err)
	}
	if run.Name != "elasticsearch" {
		t.Fatalf("il nome viene dal motore del rapporto, e' %q", run.Name)
	}
	if got := run.ByQuery["determina 1223 crispiano"]; len(got) != 2 || got[0] != "doc-0002" || got[1] != "doc-0007" {
		t.Fatalf("ranking letto male: %v", got)
	}
	// Lo spazio in coda nel rapporto non deve separare la query dalla sua gemella nel file BEIR.
	if got, ok := run.ByQuery["ordinanza 187"]; !ok || len(got) != 0 {
		t.Fatalf("una query senza risultati c'e', ripulita dagli spazi, e ha un ranking vuoto: %v %v", got, ok)
	}
}

func TestExternalRunRejectsSomethingThatIsNotAReport(t *testing.T) {
	// L'ultimo e' un rapporto di scripts/compare: ha le query ma non gli ids, e
	// letto come un rapporto dell'app darebbe ranking tutti vuoti, in silenzio.
	compare := `{"query": [{"query": "delibera", "esiti": {"any-bm25": {"ids": ["doc-0001"]}}}]}`
	for _, c := range []string{`non e' json`, `{"config": {}}`, `{"query": []}`, compare} {
		if _, err := LoadExternalRun(rapporto(t, c)); err == nil {
			t.Fatalf("%q deve dare errore", c)
		}
	}
}

// Due righe uguali nel file di query danno due ranking: se sono diversi, quale
// dei due entra nel pool sarebbe una scelta fatta in silenzio.
func TestExternalRunRefusesConflictingDuplicates(t *testing.T) {
	uguali := `{"config": {"motore": "elasticsearch"}, "query": [
	  {"query": "delibera", "ids": ["doc-0001"]}, {"query": "delibera", "ids": ["doc-0001"]}]}`
	if _, err := LoadExternalRun(rapporto(t, uguali)); err != nil {
		t.Fatalf("due righe uguali con lo stesso ranking vanno bene: %v", err)
	}
	diverse := `{"config": {"motore": "elasticsearch"}, "query": [
	  {"query": "delibera", "ids": ["doc-0001"]}, {"query": "delibera", "ids": ["doc-0002"]}]}`
	if _, err := LoadExternalRun(rapporto(t, diverse)); err == nil {
		t.Fatal("la stessa query con due ranking diversi deve dare errore")
	}
}

func TestAlignGivesEachQueryIDTheRankingOfItsText(t *testing.T) {
	run, err := LoadExternalRun(rapporto(t, rapportoElasticsearch))
	if err != nil {
		t.Fatal(err)
	}
	domande := map[string]string{"q-1": "determina 1223 crispiano", "q-2": " determina 1223 crispiano ", "q-3": "ordinanza 187"}
	corpus := map[string]bool{"doc-0002": true, "doc-0007": true}

	got, err := run.Align(domande, corpus)
	if err != nil {
		t.Fatal(err)
	}
	if len(got["q-1"]) != 2 || len(got["q-2"]) != 2 || got["q-2"][0] != "doc-0002" {
		t.Fatalf("due id con lo stesso testo hanno lo stesso ranking: %v", got)
	}
	if r, ok := got["q-3"]; !ok || len(r) != 0 {
		t.Fatalf("una query a vuoto resta, con un ranking vuoto: %v", got)
	}
}

// Un run che non ha eseguito una query non ha niente da mettere nel pool per
// lei: senza errore, quella query verrebbe giudicata solo sui documenti di
// Koskidex, e il confronto partirebbe truccato contro Elasticsearch.
func TestAlignRefusesAQueryTheRunDidNotExecute(t *testing.T) {
	run, err := LoadExternalRun(rapporto(t, rapportoElasticsearch))
	if err != nil {
		t.Fatal(err)
	}
	_, err = run.Align(map[string]string{"q-1": "ordinanza 187", "q-9": "delibera di giunta"}, map[string]bool{})
	if err == nil || !strings.Contains(err.Error(), "delibera di giunta") {
		t.Fatalf("deve dire quale query manca: %v", err)
	}
}

// Elasticsearch indicizza per Document::id, il corpus usa doc-0001: un rapporto
// con gli id sbagliati darebbe un pool di documenti che non esistono.
func TestAlignRefusesIDsOutsideTheCorpus(t *testing.T) {
	grezzi := `{"config": {"motore": "elasticsearch"}, "query": [{"query": "delibera", "ids": ["doc-0001", "7"]}]}`
	run, err := LoadExternalRun(rapporto(t, grezzi))
	if err != nil {
		t.Fatal(err)
	}
	_, err = run.Align(map[string]string{"q-1": "delibera"}, map[string]bool{"doc-0001": true})
	if err == nil || !strings.Contains(err.Error(), `"7"`) {
		t.Fatalf("deve dire quale id non sta nel corpus: %v", err)
	}
}
