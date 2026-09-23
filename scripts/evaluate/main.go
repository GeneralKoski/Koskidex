// Command evaluate scores Koskidex over a BEIR collection and writes a result
// file. Every number that ends up in the thesis comes from one of these files,
// never from a run nobody recorded.
//
//	go run ./scripts/evaluate -collection scifact -run legacy
//
// The collections are not in git: fetch them with eval/corpora/fetch.sh.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
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

	if err := esegui(*radice, *collezione, *nomeRun, *uscita, *modo, *punteggio, *analisi); err != nil {
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

func esegui(radice, collezione, nomeRun, uscita, modo, punteggio, analisi string) error {
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

	fmt.Printf("%s: %d documenti, %d query da valutare (su %d nel file), recupero %q, punteggio %q, analisi %q\n",
		collezione, len(docs), len(daValutare), len(queries), modo, punteggio, analisi)

	fmt.Print("indicizzo... ")
	t0 := time.Now()
	searcher := eval.NewKoskidexSearcher(docs, func(st *engine.Settings) {
		st.RetrievalMode = modo
		st.ScoringMode = punteggio
		analisiLessicali[analisi](st)
	})
	indicizzazione := time.Since(t0)
	fmt.Println(indicizzazione.Round(time.Millisecond))

	fmt.Print("valuto... ")
	res := eval.Run(searcher, nomeRun, collezione, daValutare, qrels)
	fmt.Println(res.Elapsed)

	res.Timings.IndexMs = float64(indicizzazione.Nanoseconds()) / 1e6
	res.Config = eval.Provenienza(".")
	res.Config["collezione"] = collezione
	res.Config["documenti"] = fmt.Sprint(len(docs))
	res.Config["recupero"] = modo
	res.Config["punteggio"] = punteggio
	res.Config["analisi"] = analisi
	res.Config["refusi"] = "0"
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
	archiviato, err := eval.Archivia("koskidex-beir", fmt.Sprintf("%s-%s", collezione, nomeRun), buf.Bytes())
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
