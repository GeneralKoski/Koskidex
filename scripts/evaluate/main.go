// Command evaluate scores Koskidex over a BEIR collection and writes a result
// file. Every number that ends up in the thesis comes from one of these files,
// never from a run nobody recorded.
//
//	go run ./scripts/evaluate -collection scifact -run legacy
//
// The collections are not in git: fetch them with eval/corpora/fetch.sh.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/GeneralKoski/Koskidex/internal/engine"
	"github.com/GeneralKoski/Koskidex/internal/eval"
)

func main() {
	collezione := flag.String("collection", "scifact", "nome della collezione sotto eval/corpora/c1-public")
	nomeRun := flag.String("run", "legacy", "etichetta della configurazione misurata, finisce nel file dei risultati")
	radice := flag.String("corpora", "eval/corpora/c1-public", "cartella delle collezioni")
	uscita := flag.String("out", "eval/results", "cartella dove scrivere il file dei risultati")
	modo := flag.String("mode", engine.RetrievalAll, "modalita' di recupero: all (congiuntivo) oppure any (disgiuntivo)")
	flag.Parse()

	if *modo != engine.RetrievalAll && *modo != engine.RetrievalAny {
		fmt.Fprintf(os.Stderr, "modalita' %q sconosciuta, usa %q o %q\n", *modo, engine.RetrievalAll, engine.RetrievalAny)
		os.Exit(1)
	}

	if err := esegui(*radice, *collezione, *nomeRun, *uscita, *modo); err != nil {
		fmt.Fprintln(os.Stderr, "errore:", err)
		os.Exit(1)
	}
}

func esegui(radice, collezione, nomeRun, uscita, modo string) error {
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

	fmt.Printf("%s: %d documenti, %d query da valutare (su %d nel file), recupero %q\n",
		collezione, len(docs), len(daValutare), len(queries), modo)

	fmt.Print("indicizzo... ")
	searcher := eval.NewKoskidexSearcher(docs, func(st *engine.Settings) {
		st.RetrievalMode = modo
	})
	fmt.Println("fatto")

	fmt.Print("valuto... ")
	res := eval.Run(searcher, nomeRun, collezione, daValutare, qrels)
	fmt.Println(res.Elapsed)

	if err := os.MkdirAll(uscita, 0o755); err != nil {
		return err
	}
	path := filepath.Join(uscita, fmt.Sprintf("%s-%s.json", collezione, nomeRun))
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := res.WriteJSON(f); err != nil {
		return err
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
	return nil
}
