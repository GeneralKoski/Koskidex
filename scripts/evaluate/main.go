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
	"strconv"
	"strings"
	"time"

	"github.com/GeneralKoski/Koskidex/internal/embedder"
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
	analisi := flag.String("analyzer", "none", "analisi lessicale: none, stopwords, stemmer, english (stopword + stemmer), italian-stopwords, italian-stemmer, italian (stopword + stemmer)")
	rankings := flag.String("rankings", "", "rapporto di app:eval-run-queries da valutare al posto di Koskidex")
	archivio := flag.String("archivio", "koskidex-beir", "sottocartella dell'archivio dei risultati")
	k1 := flag.Float64("bm25-k1", engine.DefaultBM25K1, "k1 di BM25: saturazione della frequenza del termine")
	b := flag.Float64("bm25-b", engine.DefaultBM25B, "b di BM25: peso della normalizzazione della lunghezza")
	top := flag.Int("top", 0, "quanti id della testa di ogni ranking salvare nel file (0 = nessuno)")
	espansioni := flag.String("bm25-espansioni", "", "frequenza con cui BM25 pesa le espansioni: vuoto (il termine trovato) o blended")
	senzaPrefisso := flag.Bool("senza-prefisso", false, "spegne la ricerca per prefisso (Settings.DisablePrefixSearch)")
	minimo := flag.String("minimum-should-match", "", "termini richiesti in recupero any, sintassi di Elasticsearch (Settings.MinimumShouldMatch)")
	coordinazione := flag.Bool("coordinazione", false, "punteggio per quota di termini trovati, coord di Lucene (Settings.Coordination)")
	tokenizer := flag.String("tokenizer", "", "tokenizer: vuoto (spezza su tutto cio' che non e' lettera o cifra) o standard (Settings.Tokenizer)")
	elisione := flag.Bool("elisione", false, "toglie gli articoli elisi italiani, come l'analizzatore italian di Elasticsearch (Settings.ElisionArticles); vuole -tokenizer standard")
	modello := flag.String("embedder", "", "modello Ollama per i vettori di documenti e query (per esempio bge-m3); vuoto = senza vettori")
	ollama := flag.String("ollama", "", "URL di Ollama; vuoto = localhost:11434")
	ibrido := flag.String("ibrido", "", "chi portano i vettori fra i candidati: vuoto (riordinano i lessicali), union o vector (Settings.HybridMode); vuole -embedder")
	vettoriK := flag.Int("vettori-k", 0, "quanti documenti porta il vettore con -ibrido (Settings.VectorTopK); 0 = 100")
	fusione := flag.String("fusione", "", "come si fondono lessicale e vettore: vuoto (somma, lessicale + sim * peso), rrf o convex (Settings.FusionMode); vuole -embedder")
	pesoVettore := flag.String("peso-vettore", "", "la costante della somma (Settings.VectorWeight); vuoto = 20")
	alfa := flag.String("alfa", "", "il peso del lessicale nella fusione convex (Settings.FusionAlpha); vuoto = 0,5")
	cacheVettori := flag.String("vettori-cache", "eval/cache/embeddings.jsonl", "cache dei vettori, per modello e testo: si calcolano una volta sola")
	split := flag.String("split", "test", "giudizi da usare, qrels/<split>.tsv: test, oppure train per chi impara dalle query")
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
		fmt.Fprintf(os.Stderr, "analisi %q sconosciuta, usa none, stopwords, stemmer, english, italian-stopwords, italian-stemmer o italian\n", *analisi)
		os.Exit(1)
	}

	if *espansioni != "" && *espansioni != engine.BM25ExpansionBlended {
		fmt.Fprintf(os.Stderr, "espansioni %q sconosciute, usa %q o lascia vuoto\n", *espansioni, engine.BM25ExpansionBlended)
		os.Exit(1)
	}
	if *tokenizer != "" && *tokenizer != engine.TokenizerStandard {
		fmt.Fprintf(os.Stderr, "tokenizer %q sconosciuto, usa %q o lascia vuoto\n", *tokenizer, engine.TokenizerStandard)
		os.Exit(1)
	}
	// Col tokenizer predefinito l'apostrofo spezza il termine prima che
	// l'elisione lo veda: la run sarebbe identica a quella senza, e con
	// un'etichetta che dice il contrario.
	if *elisione && *tokenizer != engine.TokenizerStandard {
		fmt.Fprintln(os.Stderr, "-elisione vuole -tokenizer standard")
		os.Exit(1)
	}
	if *ibrido != "" && *ibrido != engine.HybridUnion && *ibrido != engine.HybridVector {
		fmt.Fprintf(os.Stderr, "ibrido %q sconosciuto, usa %q, %q o lascia vuoto\n", *ibrido, engine.HybridUnion, engine.HybridVector)
		os.Exit(1)
	}
	// Senza vettori la modalita' ibrida non ha niente da portare: la run
	// sarebbe quella lessicale, con un'etichetta che dice il contrario.
	if (*ibrido != "" || *vettoriK != 0 || *fusione != "" || *pesoVettore != "" || *alfa != "") && *modello == "" {
		fmt.Fprintln(os.Stderr, "-ibrido, -vettori-k, -fusione, -peso-vettore e -alfa vogliono -embedder")
		os.Exit(1)
	}
	if *fusione != "" && *fusione != engine.FusionRRF && *fusione != engine.FusionConvex {
		fmt.Fprintf(os.Stderr, "fusione %q sconosciuta, usa %q, %q o lascia vuoto\n", *fusione, engine.FusionRRF, engine.FusionConvex)
		os.Exit(1)
	}
	peso, err := numeroFacoltativo("-peso-vettore", *pesoVettore)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	pesoLessicale, err := numeroFacoltativo("-alfa", *alfa)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if pesoLessicale != nil && (*fusione != engine.FusionConvex || *pesoLessicale < 0 || *pesoLessicale > 1) {
		fmt.Fprintln(os.Stderr, "-alfa vale fra 0 e 1, e solo con -fusione convex")
		os.Exit(1)
	}
	if peso != nil && *fusione != "" {
		fmt.Fprintln(os.Stderr, "-peso-vettore vale solo per la somma, senza -fusione")
		os.Exit(1)
	}
	if _, err := engine.RequiredTerms(*minimo, 1); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	opzioni := opzioni{k1: *k1, b: *b, top: *top, senzaPrefisso: *senzaPrefisso, espansioni: *espansioni,
		minimo: *minimo, coordinazione: *coordinazione, split: *split, tokenizer: *tokenizer, elisione: *elisione,
		embedder: engine.EmbedderSettings{Model: *modello, URL: *ollama}, cacheVettori: *cacheVettori,
		ibrido: *ibrido, vettoriK: *vettoriK, fusione: *fusione, pesoVettore: peso, alfa: pesoLessicale}
	if *modello != "" {
		opzioni.embedder.Source = embedder.SourceOllama
	}
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
	"italian-stopwords": func(s *engine.Settings) { s.StopWords = engine.ItalianStopWords() },
	"italian-stemmer":   func(s *engine.Settings) { s.Stemmer = engine.StemmerItalianLight },
	"italian": func(s *engine.Settings) {
		s.StopWords = engine.ItalianStopWords()
		s.Stemmer = engine.StemmerItalianLight
	},
}

// opzioni are the settings added after the first runs; their defaults leave a
// run exactly as it was before them.
type opzioni struct {
	k1, b         float64
	top           int
	senzaPrefisso bool
	espansioni    string
	minimo        string
	coordinazione bool
	split         string
	tokenizer     string
	elisione      bool
	embedder      engine.EmbedderSettings
	cacheVettori  string
	ibrido        string
	vettoriK      int
	fusione       string
	pesoVettore   *float64
	alfa          *float64
}

// numeroFacoltativo reads a number that can be left empty: empty is nil, the
// engine's default, which zero could not stand for.
func numeroFacoltativo(nome, v string) (*float64, error) {
	if v == "" {
		return nil, nil
	}
	x, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return nil, fmt.Errorf("%s: %q non e' un numero", nome, v)
	}
	return &x, nil
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
	qrels, err := eval.LoadQrels(filepath.Join(dir, "qrels", o.split+".tsv"))
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

		var vettori eval.Vettori
		infoVettori := map[string]string{}
		if o.embedder.Source != "" {
			vettori, infoVettori, err = eval.CalcolaVettori(o.embedder, o.cacheVettori, docs, daValutare)
			if err != nil {
				return err
			}
		}

		fmt.Print("indicizzo... ")
		t0 := time.Now()
		searcher := eval.NewKoskidexSearcherConVettori(docs, func(st *engine.Settings) {
			st.RetrievalMode = modo
			st.ScoringMode = punteggio
			st.BM25K1 = o.k1
			st.BM25B = o.b
			st.DisablePrefixSearch = o.senzaPrefisso
			st.BM25Expansion = o.espansioni
			st.MinimumShouldMatch = o.minimo
			st.Coordination = o.coordinazione
			st.Tokenizer = o.tokenizer
			st.HybridMode = o.ibrido
			st.VectorTopK = o.vettoriK
			st.FusionMode = o.fusione
			st.VectorWeight = o.pesoVettore
			st.FusionAlpha = o.alfa
			if o.elisione {
				st.ElisionArticles = engine.ItalianElisionArticles()
			}
			analisiLessicali[analisi](st)
		}, vettori)
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
		if o.minimo != "" {
			res.Config["minimum_should_match"] = o.minimo
		}
		if o.coordinazione {
			res.Config["coordinazione"] = "accesa"
		}
		if o.tokenizer != "" {
			res.Config["tokenizer"] = o.tokenizer
		}
		if o.elisione {
			res.Config["elisione"] = "articoli italiani"
		}
		for k, v := range infoVettori {
			res.Config[k] = v
		}
		if o.embedder.Source != "" {
			res.Config["ibrido"] = "rerank"
			if o.ibrido != "" {
				res.Config["ibrido"] = o.ibrido
				k := o.vettoriK
				if k <= 0 {
					k = 100
				}
				res.Config["vettori_k"] = fmt.Sprint(k)
			}
			res.Config["fusione"] = "somma"
			if o.fusione != "" {
				res.Config["fusione"] = o.fusione
			}
			switch {
			case o.fusione == "" && o.pesoVettore != nil:
				res.Config["peso_vettore"] = strconv.FormatFloat(*o.pesoVettore, 'f', -1, 64)
			case o.fusione == "":
				res.Config["peso_vettore"] = "20"
			case o.fusione == engine.FusionConvex && o.alfa != nil:
				res.Config["alfa"] = strconv.FormatFloat(*o.alfa, 'f', -1, 64)
			case o.fusione == engine.FusionConvex:
				res.Config["alfa"] = "0.5"
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
	if o.split != "test" {
		res.Config["split"] = o.split
	}
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

// calcolaVettori embeds every document and every query to evaluate through
// the cache, so that a second run with the same model calls nothing. What it
// records says which model, which weights, and how many vectors were computed
// now rather than read from the cache.

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
