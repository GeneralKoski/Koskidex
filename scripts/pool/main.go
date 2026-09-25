// Command pool prepares the relevance judgments for a collection, and turns
// them back into qrels once they are written.
//
// Two steps, because between them there is a person.
//
//	go run ./scripts/pool -corpus c.jsonl -queries q.jsonl -out giudizi.tsv \
//	    [-embedder bge-m3] [-composizione pool.json] [-run rapporto-elasticsearch.json ...]
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
//
// -embedder adds the configurations with vectors, which bring documents with
// none of the query's words. -composizione writes, ids only, which
// configuration brought which document: the sheet does not say it, on purpose,
// and the thesis needs it.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GeneralKoski/Koskidex/internal/embedder"
	"github.com/GeneralKoski/Koskidex/internal/engine"
	"github.com/GeneralKoski/Koskidex/internal/eval"
)

type configurazione struct {
	nome     string
	vettori  bool
	modifica func(*engine.Settings)
}

func bm25(s *engine.Settings) {
	s.ScoringMode = engine.ScoringBM25
	s.BM25Expansion = engine.BM25ExpansionBlended
}

// Le configurazioni in gioco: quella di oggi, le correzioni del lessicale e,
// con -embedder, le due ibride calibrate (2026-09-25_scelta-ibrida) e il solo
// vettore. Il pool le unisce tutte, cosi' nessuna viene giudicata su documenti
// scelti da un'altra.
var configurazioni = []configurazione{
	{"all-euristico", false, func(s *engine.Settings) { s.RetrievalMode = engine.RetrievalAll; s.ScoringMode = engine.ScoringLegacy }},
	{"any-euristico", false, func(s *engine.Settings) { s.RetrievalMode = engine.RetrievalAny; s.ScoringMode = engine.ScoringLegacy }},
	{"any-bm25", false, func(s *engine.Settings) { s.RetrievalMode = engine.RetrievalAny; s.ScoringMode = engine.ScoringBM25 }},
	{"any-bm25-mescolata", false, func(s *engine.Settings) { s.RetrievalMode = engine.RetrievalAny; bm25(s) }},
	{"all-bm25-mescolata", false, func(s *engine.Settings) { s.RetrievalMode = engine.RetrievalAll; bm25(s) }},
	{"any-bm25-mescolata-italiano", false, func(s *engine.Settings) {
		s.RetrievalMode = engine.RetrievalAny
		bm25(s)
		s.Tokenizer = engine.TokenizerStandard
		s.ElisionArticles = engine.ItalianElisionArticles()
		s.StopWords = engine.ItalianStopWords()
		s.Stemmer = engine.StemmerItalianLight
	}},
	{"ibrida-A-any-160", true, func(s *engine.Settings) {
		s.RetrievalMode = engine.RetrievalAny
		bm25(s)
		s.HybridMode = engine.HybridUnion
		peso := 160.0
		s.VectorWeight = &peso
	}},
	{"ibrida-B-all-10", true, func(s *engine.Settings) {
		s.RetrievalMode = engine.RetrievalAll
		bm25(s)
		s.HybridMode = engine.HybridUnion
		peso := 10.0
		s.VectorWeight = &peso
	}},
	{"solo-vettore", true, func(s *engine.Settings) {
		s.RetrievalMode = engine.RetrievalAny
		bm25(s)
		s.HybridMode = engine.HybridVector
	}},
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
	modello := flag.String("embedder", "", "modello Ollama (per esempio bge-m3): aggiunge le configurazioni con i vettori")
	ollama := flag.String("ollama", "", "URL di Ollama; vuoto = localhost:11434")
	cacheVettori := flag.String("vettori-cache", "eval/cache/embeddings.jsonl", "cache dei vettori, la stessa di scripts/evaluate")
	composizione := flag.String("composizione", "", "dove scrivere quale configurazione ha portato quale documento (solo id)")
	var esterni rapporti
	flag.Var(&esterni, "run", "rapporto di app:eval-run-queries da aggiungere al pool (ripetibile)")
	flag.Parse()
	var vettori *engine.EmbedderSettings
	if *modello != "" {
		vettori = &engine.EmbedderSettings{Source: embedder.SourceOllama, Model: *modello, URL: *ollama}
	}

	var err error
	switch {
	case *foglio != "":
		err = converti(*foglio, *qrels)
	case *corpus != "" && *queries != "":
		err = prepara(*corpus, *queries, *uscita, *profondita, esterni, vettori, *cacheVettori, *composizione)
	default:
		flag.Usage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "errore:", err)
		os.Exit(1)
	}
}

func prepara(pathCorpus, pathQueries, out string, profondita int, esterni []string,
	impostazioniVettori *engine.EmbedderSettings, cacheVettori, pathComposizione string) error {
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
		// Il nome del motore non basta: dall'app Koskidex si interroga in piu'
		// configurazioni. Si usa il nome del file senza l'ora.
		nome := strings.TrimSuffix(filepath.Base(path), ".json")
		if _, dopo, ok := strings.Cut(nome, "_"); ok {
			nome = dopo
		}
		nomiEsterni[i] = nome
	}

	var vettori eval.Vettori
	info := map[string]string{}
	if impostazioniVettori != nil {
		if vettori, info, err = eval.CalcolaVettori(*impostazioniVettori, cacheVettori, docs, domande); err != nil {
			return err
		}
	}
	var usate []configurazione
	var cercatori []*eval.KoskidexSearcher
	for _, c := range configurazioni {
		if c.vettori && impostazioniVettori == nil {
			continue
		}
		v := eval.Vettori{}
		if c.vettori {
			v = vettori
		}
		usate = append(usate, c)
		cercatori = append(cercatori, eval.NewKoskidexSearcherConVettori(docs, c.modifica, v))
	}
	nomi := make([]string, 0, len(usate)+len(esterni))
	for _, c := range usate {
		nomi = append(nomi, c.nome)
	}
	nomi = append(nomi, nomiEsterni...)
	portati := map[string]map[string][]string{}

	ids := make([]string, 0, len(domande))
	for id := range domande {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var righe []eval.SheetRow
	senzaCandidati, soloEsterni := 0, 0
	for _, qid := range ids {
		testo := domande[qid]

		rankings := make([][]string, len(cercatori))
		for i, c := range cercatori {
			rankings[i], _ = c.Search(testo, profondita)
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
		portati[qid] = map[string][]string{}
		for i, r := range rankings {
			portati[qid][nomi[i]] = r[:min(profondita, len(r))]
		}
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

	if pathComposizione != "" {
		dati, err := json.MarshalIndent(map[string]interface{}{
			"profondita": profondita, "configurazioni": nomi, "vettori": info, "query": portati,
		}, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(pathComposizione, append(dati, '\n'), 0o644); err != nil {
			return err
		}
	}

	fmt.Printf("%d documenti, %d query, profondita' %d, configurazioni: %s\n", len(docs), len(domande), profondita, strings.Join(nomi, ", "))
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
