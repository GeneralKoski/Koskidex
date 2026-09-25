// Command pool prepares the relevance judgments for a collection, and turns
// them back into qrels once they are written.
//
// Two steps, because between them there is a person.
//
//	go run ./scripts/pool -corpus c.jsonl -queries q.jsonl -out giudizi.tsv \
//	    [-run rapporto-elasticsearch.json ...]
//	# si apre giudizi.tsv, si riempie la colonna "grado"
//	go run ./scripts/pool -sheet giudizi.tsv -qrels qrels/test.tsv
//
// The grade is 0 (not relevant), 1 (relevant) or 2 (exactly what was being
// looked for). Leaving it blank is an error on the way back, deliberately: a
// document nobody got to is not a document judged irrelevant.
//
// What ends up in the sheet is the pool, the union of the top results of
// several configurations. Documents no configuration retrieved are never
// judged and count as not relevant, so recall measured this way is an upper
// bound. It is the standard compromise, and it is the reason the pool is built
// from several configurations rather than from the one being promoted.
//
// -run adds the rankings of another engine, as Documentale's
// app:eval-run-queries records them, one flag per report. Without them the pool
// is Koskidex's alone, and a document only Elasticsearch finds is never judged.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GeneralKoski/Koskidex/internal/engine"
	"github.com/GeneralKoski/Koskidex/internal/eval"
)

const campo = "text"

type configurazione struct {
	recupero  string
	punteggio string
}

// Le tre configurazioni in gioco: quella di oggi e le due corrette. Il pool le
// unisce tutte, cosi' nessuna delle tre viene giudicata su documenti scelti da
// un'altra.
var configurazioni = []configurazione{
	{engine.RetrievalAll, engine.ScoringLegacy},
	{engine.RetrievalAny, engine.ScoringLegacy},
	{engine.RetrievalAny, engine.ScoringBM25},
}

// rapporti collects -run, which can be given more than once.
type rapporti []string

func (r *rapporti) String() string { return strings.Join(*r, ",") }

func (r *rapporti) Set(v string) error {
	*r = append(*r, v)

	return nil
}

func main() {
	corpus := flag.String("corpus", "", "corpus.jsonl in formato BEIR")
	queries := flag.String("queries", "", "queries.jsonl in formato BEIR")
	profondita := flag.Int("depth", 10, "quanti risultati per configurazione entrano nel pool")
	uscita := flag.String("out", "", "foglio di annotazione da scrivere")
	foglio := flag.String("sheet", "", "foglio compilato da riconvertire")
	qrels := flag.String("qrels", "", "file di giudizi da scrivere dal foglio")
	var esterni rapporti
	flag.Var(&esterni, "run", "rapporto di app:eval-run-queries da aggiungere al pool (ripetibile)")
	flag.Parse()

	var err error
	switch {
	case *foglio != "":
		err = converti(*foglio, *qrels)
	case *corpus != "" && *queries != "":
		err = prepara(*corpus, *queries, *uscita, *profondita, esterni)
	default:
		flag.Usage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "errore:", err)
		os.Exit(1)
	}
}

func prepara(pathCorpus, pathQueries, out string, profondita int, esterni []string) error {
	if out == "" {
		return fmt.Errorf("serve -out: dove scrivere il foglio")
	}

	docs, err := eval.LoadCorpus(pathCorpus)
	if err != nil {
		return err
	}
	domande, err := eval.LoadQueries(pathQueries)
	if err != nil {
		return err
	}

	perID := map[string]eval.Document{}
	nelCorpus := map[string]bool{}
	for _, d := range docs {
		perID[d.ID] = d
		nelCorpus[d.ID] = true
	}

	nomiEsterni := make([]string, len(esterni))
	rankingEsterni := make([]map[string][]string, len(esterni))
	for i, path := range esterni {
		run, err := eval.LoadExternalRun(path)
		if err != nil {
			return err
		}
		if rankingEsterni[i], err = run.Align(domande, nelCorpus); err != nil {
			return err
		}
		nomiEsterni[i] = run.Name
	}

	indici := make([]*engine.InvertedIndex, len(configurazioni))
	impostazioni := make([]engine.Settings, len(configurazioni))
	for i, c := range configurazioni {
		indici[i], impostazioni[i] = indicizza(docs, c)
	}

	ids := make([]string, 0, len(domande))
	for id := range domande {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var righe []eval.SheetRow
	senzaCandidati, soloEsterni := 0, 0
	for _, qid := range ids {
		testo := domande[qid]

		rankings := make([][]string, len(configurazioni))
		for i := range configurazioni {
			rankings[i], _ = indici[i].Search(testo, impostazioni[i], "0", nil)
		}

		// Quanti documenti porta solo chi sta fuori da Koskidex: se aggiungere
		// Elasticsearch non aggiunge mai niente, o va bene davvero o qualcosa
		// non funziona, e le due cose vanno distinte guardando.
		soloKoskidex := len(eval.PoolForQuery(rankings, profondita))
		for _, r := range rankingEsterni {
			rankings = append(rankings, r[qid])
		}
		pool := eval.PoolForQuery(rankings, profondita)
		soloEsterni += len(pool) - soloKoskidex
		if len(pool) == 0 {
			senzaCandidati++

			continue
		}
		for _, did := range pool {
			d := perID[did]
			righe = append(righe, eval.SheetRow{
				QueryID: qid,
				Query:   testo,
				DocID:   did,
				Title:   d.Title,
				Snippet: estratto(d.Text, 200),
			})
		}
	}

	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return err
	}
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := eval.WriteSheet(f, righe); err != nil {
		return err
	}

	fmt.Printf("%d documenti, %d query, profondita' %d\n", len(docs), len(domande), profondita)
	fmt.Printf("scritto %s: %d giudizi da dare\n", out, len(righe))
	if len(domande) > 0 {
		fmt.Printf("  in media %.1f documenti per query\n", float64(len(righe))/float64(len(domande)-senzaCandidati))
	}
	if len(esterni) > 0 {
		fmt.Printf("  %d documenti nel pool vengono solo da %s\n", soloEsterni, strings.Join(nomiEsterni, ", "))
	}
	if senzaCandidati > 0 {
		fmt.Printf("  %d query non pescano niente in nessuna configurazione e non sono nel foglio\n", senzaCandidati)
	}

	return nil
}

func converti(pathFoglio, pathQrels string) error {
	if pathQrels == "" {
		return fmt.Errorf("serve -qrels: dove scrivere i giudizi")
	}

	f, err := os.Open(pathFoglio)
	if err != nil {
		return err
	}
	defer f.Close()

	righe, err := eval.ReadSheet(f)
	if err != nil {
		return err
	}
	q, err := eval.SheetToQrels(righe)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(pathQrels), 0o755); err != nil {
		return err
	}
	out, err := os.Create(pathQrels)
	if err != nil {
		return err
	}
	defer out.Close()
	if err := eval.WriteQrels(out, q); err != nil {
		return err
	}

	var rilevanti, totale int
	for _, giudizi := range q {
		for _, g := range giudizi {
			totale++
			if g > 0 {
				rilevanti++
			}
		}
	}
	fmt.Printf("scritto %s: %d query, %d giudizi, %d rilevanti\n", pathQrels, len(q), totale, rilevanti)

	// Una query senza nemmeno un rilevante viene saltata in valutazione:
	// meglio saperlo adesso che vedere il conto delle query calare dopo.
	var vuote []string
	for qid, giudizi := range q {
		if eval.CountRelevant(giudizi) == 0 {
			vuote = append(vuote, qid)
		}
	}
	if len(vuote) > 0 {
		sort.Strings(vuote)
		fmt.Printf("  %d query senza nessun documento rilevante, verranno saltate: %s\n",
			len(vuote), strings.Join(vuote, " "))
	}

	return nil
}

func indicizza(docs []eval.Document, c configurazione) (*engine.InvertedIndex, engine.Settings) {
	s := engine.DefaultSettings()
	s.SearchableFields = []string{campo}
	s.FieldWeights = map[string]float64{campo: 1.0}
	s.TypoTolerance.Enabled = false
	s.RetrievalMode = c.recupero
	s.ScoringMode = c.punteggio

	idx := engine.NewInvertedIndex()
	for _, d := range docs {
		testo := strings.TrimSpace(d.Title + " " + d.Text)
		idx.AddDocument(d.ID, map[string]interface{}{campo: testo}, s)
	}

	return idx, s
}

func estratto(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) <= n {
		return s
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}

	return string(r[:n]) + "..."
}
