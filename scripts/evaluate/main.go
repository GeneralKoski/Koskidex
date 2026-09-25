// Command evaluate scores Koskidex over a BEIR collection and writes a result
// file. Every number that ends up in the thesis comes from one of these files,
// never from a run nobody recorded.
//
//	go run ./scripts/evaluate -collection scifact -run legacy
//
// The collections are not in git: fetch them with eval/corpora/fetch.sh.
//
// With -rankings it scores another engine instead of Koskidex: the report of
// Documentale's app:eval-run-queries, run on the same queries, goes through
// the same metric code, so the two engines are counted the same way.
//
//	go run ./scripts/evaluate -corpora eval/corpora/c3-albo -collection known-item-auto \
//	    -run elasticsearch -rankings rapporto.json -archivio valutazioni-albo
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/GeneralKoski/Koskidex/internal/engine"
	"github.com/GeneralKoski/Koskidex/internal/eval"
)

func main() {
	collezione := flag.String("collection", "scifact", "nome della collezione sotto eval/corpora/c1-public")
	nomeRun := flag.String("run", "legacy", "etichetta della configurazione misurata, finisce nel file dei risultati")
	radice := flag.String("corpora", "eval/corpora/c1-public", "cartella delle collezioni")
	uscita := flag.String("out", "eval/results", "cartella dove scrivere il file dei risultati")
	modo := flag.String("mode", engine.RetrievalAll, "modalita' di recupero: all (congiuntivo) oppure any (disgiuntivo)")
	punteggio := flag.String("scoring", engine.ScoringLegacy, "modalita' di punteggio: legacy oppure bm25")
	analisi := flag.String("analyzer", "none", "analisi lessicale: none, stopwords, stemmer, english (stopword + stemmer)")
	rankings := flag.String("rankings", "", "rapporto di app:eval-run-queries da valutare al posto di Koskidex")
	archivio := flag.String("archivio", "koskidex-beir", "sottocartella dell'archivio dei risultati")
	k1 := flag.Float64("bm25-k1", engine.DefaultBM25K1, "k1 di BM25: saturazione della frequenza del termine")
	b := flag.Float64("bm25-b", engine.DefaultBM25B, "b di BM25: peso della normalizzazione della lunghezza")
	top := flag.Int("top", 0, "quanti id della testa di ogni ranking salvare nel file (0 = nessuno)")
	espansioni := flag.String("bm25-espansioni", "", "frequenza con cui BM25 pesa le espansioni: vuoto (il termine trovato) o blended")
	senzaPrefisso := flag.Bool("senza-prefisso", false, "spegne la ricerca per prefisso (Settings.DisablePrefixSearch)")
	flag.Parse()

	if *modo != engine.RetrievalAll && *modo != engine.RetrievalAny {
		fmt.Fprintf(os.Stderr, "modalita' %q sconosciuta, usa %q o %q\n", *modo, engine.RetrievalAll, engine.RetrievalAny)
		os.Exit(1)
	}

	if *punteggio != engine.ScoringLegacy && *punteggio != engine.ScoringBM25 {
		fmt.Fprintf(os.Stderr, "punteggio %q sconosciuto, usa %q o %q\n", *punteggio, engine.ScoringLegacy, engine.ScoringBM25)
		os.Exit(1)
	}

	if _, noto := analisiLessicali[*analisi]; !noto {
		fmt.Fprintf(os.Stderr, "analisi %q sconosciuta, usa none, stopwords, stemmer o english\n", *analisi)
		os.Exit(1)
	}

	if *espansioni != "" && *espansioni != engine.BM25ExpansionBlended {
		fmt.Fprintf(os.Stderr, "espansioni %q sconosciute, usa %q o lascia vuoto\n", *espansioni, engine.BM25ExpansionBlended)
		os.Exit(1)
	}
	opzioni := opzioni{k1: *k1, b: *b, top: *top, senzaPrefisso: *senzaPrefisso, espansioni: *espansioni}
	if err := esegui(*radice, *collezione, *nomeRun, *uscita, *modo, *punteggio, *analisi, *rankings, *archivio, opzioni); err != nil {
		fmt.Fprintln(os.Stderr, "errore:", err)
		os.Exit(1)
	}
}

// analisiLessicali sono le quattro combinazioni da misurare separate: senza
// sapere quale dei due stadi porta cosa, il totale non dice niente su
// nessuno dei due.
var analisiLessicali = map[string]func(*engine.Settings){
	"none":      func(*engine.Settings) {},
	"stopwords": func(s *engine.Settings) { s.StopWords = engine.EnglishStopWords() },
	"stemmer":   func(s *engine.Settings) { s.Stemmer = engine.StemmerPorter },
	"english": func(s *engine.Settings) {
		s.StopWords = engine.EnglishStopWords()
		s.Stemmer = engine.StemmerPorter
	},
}

// opzioni are the settings added after the first runs; their defaults leave a
// run exactly as it was before them.
type opzioni struct {
	k1, b         float64
	top           int
	senzaPrefisso bool
	espansioni    string
}

func esegui(radice, collezione, nomeRun, uscita, modo, punteggio, analisi, rankings, archivio string, o opzioni) error {
	dir := filepath.Join(radice, collezione)
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("collezione %q non trovata in %s, lancia eval/corpora/fetch.sh", collezione, radice)
	}

	docs, err := eval.LoadCorpus(filepath.Join(dir, "corpus.jsonl"))
	if err != nil {
		return err
	}
	queries, err := eval.LoadQueries(filepath.Join(dir, "queries.jsonl"))
	if err != nil {
		return err
	}
	qrels, err := eval.LoadQrels(filepath.Join(dir, "qrels", "test.tsv"))
	if err != nil {
		return err
	}
	daValutare, err := eval.QueriesToEvaluate(queries, qrels)
	if err != nil {
		return err
	}

	var res eval.Results
	if rankings != "" {
		res, err = valutaEsterno(rankings, docs, daValutare, qrels, nomeRun, collezione, o.top)
		if err != nil {
			return err
		}
	} else {
		fmt.Printf("%s: %d documenti, %d query da valutare (su %d nel file), recupero %q, punteggio %q, analisi %q\n",
			collezione, len(docs), len(daValutare), len(queries), modo, punteggio, analisi)

		fmt.Print("indicizzo... ")
		t0 := time.Now()
		searcher := eval.NewKoskidexSearcher(docs, func(st *engine.Settings) {
			st.RetrievalMode = modo
			st.ScoringMode = punteggio
			st.BM25K1 = o.k1
			st.BM25B = o.b
			st.DisablePrefixSearch = o.senzaPrefisso
			st.BM25Expansion = o.espansioni
			analisiLessicali[analisi](st)
		})
		indicizzazione := time.Since(t0)
		fmt.Println(indicizzazione.Round(time.Millisecond))

		fmt.Print("valuto... ")
		res = eval.RunTop(searcher, nomeRun, collezione, daValutare, qrels, o.top)
		fmt.Println(res.Elapsed)

		res.Timings.IndexMs = float64(indicizzazione.Nanoseconds()) / 1e6
		res.Config = eval.Provenienza(".")
		res.Config["recupero"] = modo
		res.Config["punteggio"] = punteggio
		res.Config["analisi"] = analisi
		res.Config["refusi"] = "0"
		if punteggio == engine.ScoringBM25 {
			res.Config["bm25_k1"] = fmt.Sprint(o.k1)
			res.Config["bm25_b"] = fmt.Sprint(o.b)
			if o.espansioni != "" {
				res.Config["bm25_espansioni"] = o.espansioni
			}
		}
	}
	if o.top > 0 {
		res.Config["top"] = fmt.Sprint(o.top)
	}
	if o.senzaPrefisso {
		res.Config["ricerca_per_prefisso"] = "spenta"
	}
	res.Config["collezione"] = collezione
	res.Config["documenti"] = fmt.Sprint(len(docs))
	if impronta, err := eval.ImprontaFile(filepath.Join(dir, "corpus.jsonl")); err == nil {
		res.Config["corpus_sha256"] = impronta
	}

	if err := os.MkdirAll(uscita, 0o755); err != nil {
		return err
	}
	path := filepath.Join(uscita, fmt.Sprintf("%s-%s.json", collezione, nomeRun))
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	var buf bytes.Buffer
	if err := res.WriteJSON(&buf); err != nil {
		return err
	}
	if _, err := f.Write(buf.Bytes()); err != nil {
		return err
	}
	archiviato, err := eval.Archivia(archivio, fmt.Sprintf("%s-%s", collezione, nomeRun), buf.Bytes())
	if err != nil {
		return fmt.Errorf("archiviazione fallita: %w", err)
	}

	fmt.Printf("\n  nDCG@10    %.4f\n  Recall@100 %.4f\n  MRR@10     %.4f\n",
		res.MeanNDCG10, res.MeanRecall100, res.MeanMRR10)
	if res.ZeroResults > 0 {
		fmt.Printf("  QUERY A VUOTO %d su %d (%.0f%%)\n",
			res.ZeroResults, res.Queries, 100*float64(res.ZeroResults)/float64(res.Queries))
	}
	fmt.Printf("  su %d query", res.Queries)
	if res.SkippedNoRel > 0 {
		fmt.Printf(", %d saltate perche' senza documenti rilevanti", res.SkippedNoRel)
	}
	fmt.Printf("\n\nscritto in %s\n", path)
	if archiviato != "" {
		fmt.Printf("archiviato in %s\n", archiviato)
	} else {
		fmt.Printf("\n!!! RISULTATO NON ARCHIVIATO: %s non e' impostata.\n", eval.VarArchivio)
		fmt.Println("!!! Il file qui sopra verra' sovrascritto alla prossima esecuzione.")
	}
	return nil
}

// valutaEsterno scores a recorded run. The per-query times are the ones the
// app measured: timing a lookup in a map would say nothing about the engine.
func valutaEsterno(path string, docs []eval.Document, domande map[string]string, qrels eval.Qrels, nomeRun, collezione string, top int) (eval.Results, error) {
	run, err := eval.LoadExternalRun(path)
	if err != nil {
		return eval.Results{}, err
	}
	nelCorpus := make(map[string]bool, len(docs))
	for _, d := range docs {
		nelCorpus[d.ID] = true
	}
	if _, err := run.Align(domande, nelCorpus); err != nil {
		return eval.Results{}, err
	}

	fmt.Printf("%s: %d documenti, %d query da valutare, ranking di %s da %s\n",
		collezione, len(docs), len(domande), run.Name, filepath.Base(path))
	res := eval.RunTop(run, nomeRun, collezione, domande, qrels, top)
	for qid := range res.Timings.PerQueryMs {
		res.Timings.PerQueryMs[qid] = run.Ms[strings.TrimSpace(domande[qid])]
	}

	res.Config = eval.Provenienza(".")
	res.Config["motore"] = run.Name
	res.Config["rapporto"] = filepath.Base(path)
	if impronta, err := eval.ImprontaFile(path); err == nil {
		res.Config["rapporto_sha256"] = impronta
	}

	return res, nil
}
