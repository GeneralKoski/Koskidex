// Command compare runs the same queries through several Koskidex
// configurations over a corpus exported from Documentale, and prints the
// rankings side by side.
//
// It exists to answer, on one screen, the question the thesis asks: what does
// each defect actually cost on the documents of a real archive, as opposed to
// on a public collection.
//
//	go run ./scripts/compare -corpus corpus.jsonl -queries query.txt
//
// The corpus comes from Documentale's app:export-eval-corpus. To put the
// numbers next to Elasticsearch's own, the settings have to match what
// Documentale actually runs, which is why fuzziness defaults to "auto" here and
// to off in scripts/evaluate: there the references are non-fuzzy, here the
// comparison is against a live configuration.
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/GeneralKoski/Koskidex/internal/engine"
	"github.com/GeneralKoski/Koskidex/internal/eval"
)

// documento is one line of the JSONL produced by Documentale's
// app:export-eval-corpus, which writes BEIR keys: the id is "_id", and the
// name of the document travels in "title", separate from "text".
type documento struct {
	ID    string `json:"_id"`
	Title string `json:"title"`
	Text  string `json:"text"`
}

type configurazione struct {
	etichetta    string
	recupero     string
	punteggio    string
	sottostringa bool
}

func main() {
	corpus := flag.String("corpus", "", "file JSONL esportato da Documentale")
	queries := flag.String("queries", "", "file con una query per riga")
	quante := flag.Int("top", 5, "quanti risultati mostrare per query")
	conserva := flag.Int("conserva", 1000, "quanti risultati per query e configurazione salvare nel rapporto; il totale trovato si salva sempre")
	uscita := flag.String("json", "", "se valorizzato, scrive il rapporto anche in questo file (l'archivio della tesi lo riceve comunque)")
	// Il default e' "auto" perche' Documentale cerca con fuzziness AUTO: con la
	// tolleranza spenta i due motori recuperano insiemi diversi e la tabella
	// sembra dire che Koskidex si comporta diversamente, mentre sta solo
	// girando con una configurazione che nessuno usa.
	refusi := flag.String("fuzziness", "auto", "tolleranza ai refusi: auto come Elasticsearch in produzione, 0 per spegnerla")
	// La ricerca delle cartelle di Documentale unisce al match un wildcard
	// *termine*. Accanto alla configurazione or + euristico se ne misura una
	// uguale con la sottostringa accesa: stesse query, stesso corpus, cosi' la
	// differenza di latenza e' il costo della scansione del vocabolario.
	sottostringa := flag.Bool("sottostringa", false, "aggiunge la configurazione or + euristico con la ricerca per sottostringa")
	flag.Parse()

	if *corpus == "" || *queries == "" {
		fmt.Fprintln(os.Stderr, "servono -corpus e -queries")
		os.Exit(1)
	}

	docs, err := leggiCorpus(*corpus)
	if err != nil {
		fmt.Fprintln(os.Stderr, "errore:", err)
		os.Exit(1)
	}
	if err := controllaIdentificatori(docs); err != nil {
		fmt.Fprintln(os.Stderr, "errore:", err)
		os.Exit(1)
	}

	domande, err := leggiRighe(*queries)
	if err != nil {
		fmt.Fprintln(os.Stderr, "errore:", err)
		os.Exit(1)
	}

	configurazioni := []configurazione{
		{"oggi (and + euristico)", engine.RetrievalAll, engine.ScoringLegacy, false},
		{"or + euristico", engine.RetrievalAny, engine.ScoringLegacy, false},
		{"or + BM25", engine.RetrievalAny, engine.ScoringBM25, false},
	}
	if *sottostringa {
		configurazioni = append(configurazioni, configurazione{"or + euristico + sottostringa", engine.RetrievalAny, engine.ScoringLegacy, true})
	}

	fmt.Printf("%d documenti, %d query, tolleranza refusi %q\n", len(docs), len(domande), *refusi)

	vuote := make([]int, len(configurazioni))

	// Un indice per configurazione, costruito una volta sola e cronometrato.
	// Prima si ricostruiva a ogni query: era lento, e soprattutto rendeva
	// impossibile dire quanto costa indicizzare e quanto costa cercare.
	indici := make([]*engine.InvertedIndex, len(configurazioni))
	impostazioni := make([]engine.Settings, len(configurazioni))
	rap := rapporto{
		RanAt:  time.Now().UTC().Format(time.RFC3339),
		Config: eval.Provenienza("."),
	}
	rap.Config["corpus"] = *corpus
	rap.Config["documenti"] = fmt.Sprint(len(docs))
	rap.Config["refusi"] = *refusi
	rap.Config["risultati_conservati_per_query"] = fmt.Sprint(*conserva)
	if impronta, err := eval.ImprontaFile(*corpus); err == nil {
		rap.Config["corpus_sha256"] = impronta
	}
	for i, c := range configurazioni {
		t0 := time.Now()
		indici[i], impostazioni[i] = indicizza(docs, c, *refusi)
		ms := ms(time.Since(t0))
		rap.Configurazioni = append(rap.Configurazioni, infoConfigurazione{c.etichetta, c.recupero, c.punteggio, c.sottostringa, indici[i].VocabularySize(), ms})
		fmt.Printf("indice %-30s %8.0f ms  %d termini\n", c.etichetta, ms, indici[i].VocabularySize())
	}

	for _, q := range domande {
		riga := esitoQuery{Query: q, Esiti: map[string]esito{}}
		fmt.Printf("\n\n== %s  (%d parole)\n", q, len(strings.Fields(q)))
		for i, c := range configurazioni {
			t0 := time.Now()
			trovati, _ := indici[i].SearchScored(q, impostazioni[i], *refusi, nil)
			durata := ms(time.Since(t0))
			if len(trovati) == 0 {
				vuote[i]++
			}

			// Solo id e punteggi: i titoli degli atti possono contenere nomi di
			// persone, e questi file finiscono nel repository della tesi.
			// Si conserva la testa del ranking, non tutto: con il recupero
			// disgiuntivo sono migliaia di documenti per query, 12 MB a
			// esecuzione, e nessuna metrica guarda oltre le prime 100
			// posizioni. Il totale resta, perche' e' l'unico modo di dire se
			// l'insieme recuperato e' cambiato.
			e := esito{IDs: []string{}, Punteggi: []float64{}, Trovati: len(trovati), Ms: durata}
			for n, m := range trovati {
				if n >= *conserva {
					break
				}
				e.IDs = append(e.IDs, m.DocID)
				e.Punteggi = append(e.Punteggi, m.Score)
			}
			riga.Esiti[c.etichetta] = e

			fmt.Printf("\n  %-24s %d risultati  (%.2f ms)\n", c.etichetta, len(trovati), durata)
			for n, m := range trovati {
				if n >= *quante {
					fmt.Printf("      ... e altri %d\n", len(trovati)-*quante)
					break
				}
				fmt.Printf("      %d. %-6.3f %s\n", n+1, m.Score, etichetta(docs, m.DocID))
			}
		}
		rap.Query = append(rap.Query, riga)
	}

	fmt.Printf("\n\n== query a vuoto su %d\n", len(domande))
	for i, c := range configurazioni {
		fmt.Printf("  %-24s %d\n", c.etichetta, vuote[i])
	}

	// Compatto: indentato, un rapporto con migliaia di id va a capo a ogni id
	// e raddoppia di peso senza diventare piu' leggibile.
	dati, err := json.Marshal(rap)
	if err != nil {
		fmt.Fprintln(os.Stderr, "errore:", err)
		os.Exit(1)
	}
	if *uscita != "" {
		if err := os.WriteFile(*uscita, dati, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "errore:", err)
			os.Exit(1)
		}
	}
	nome := filepath.Base(filepath.Dir(*corpus)) + "-koskidex"
	archiviato, err := eval.Archivia("confronto", nome, dati)
	if err != nil {
		fmt.Fprintln(os.Stderr, "errore di archiviazione:", err)
		os.Exit(1)
	}
	if archiviato != "" {
		fmt.Printf("\narchiviato in %s\n", archiviato)
	} else {
		fmt.Printf("\n!!! RISULTATO NON ARCHIVIATO: %s non e' impostata.\n", eval.VarArchivio)
	}
}

// rapporto e' il file che finisce nell'archivio della tesi: configurazione,
// provenienza del codice, tempi di indicizzazione e, per ogni query, ranking e
// latenza di ciascuna configurazione.
type rapporto struct {
	RanAt          string               `json:"ran_at"`
	Config         map[string]string    `json:"config"`
	Configurazioni []infoConfigurazione `json:"configurazioni"`
	Query          []esitoQuery         `json:"query"`
}

type infoConfigurazione struct {
	Etichetta    string  `json:"etichetta"`
	Recupero     string  `json:"recupero"`
	Punteggio    string  `json:"punteggio"`
	Sottostringa bool    `json:"sottostringa"`
	Termini      int     `json:"termini"`
	IndexMs      float64 `json:"index_ms"`
}

type esitoQuery struct {
	Query string           `json:"query"`
	Esiti map[string]esito `json:"esiti"`
}

type esito struct {
	IDs      []string  `json:"ids"`
	Punteggi []float64 `json:"punteggi"`
	Trovati  int       `json:"trovati"`
	Ms       float64   `json:"ms"`
}

func ms(d time.Duration) float64 {
	return float64(d.Nanoseconds()) / 1e6
}

// indicizza costruisce un indice per configurazione. Le statistiche BM25
// dipendono dalle settings usate in indicizzazione, quindi l'indice non si
// riusa fra configurazioni diverse.
func indicizza(docs []documento, c configurazione, refusi string) (*engine.InvertedIndex, engine.Settings) {
	s := engine.DefaultSettings()
	// Due campi con i pesi di Documentale: Elasticsearch cerca su name^5 e sui
	// campi dei metadati con peso minore. Impastare titolo e testo in un campo
	// solo darebbe al titolo il peso del corpo, e il confronto misurerebbe una
	// configurazione che nessuno usa.
	s.SearchableFields = []string{"title", "text"}
	s.FieldWeights = map[string]float64{"title": 5.0, "text": 1.0}
	s.TypoTolerance.Enabled = refusi != "0"
	s.RetrievalMode = c.recupero
	s.ScoringMode = c.punteggio
	s.SubstringMatch = c.sottostringa

	idx := engine.NewInvertedIndex()
	for _, d := range docs {
		idx.AddDocument(d.ID, map[string]interface{}{"title": d.Title, "text": d.Text}, s)
	}

	return idx, s
}

func etichetta(docs []documento, id string) string {
	for _, d := range docs {
		if d.ID == id {
			if d.Title != "" {
				return d.Title
			}
			prima, _, _ := strings.Cut(d.Text, "\n")
			return prima
		}
	}
	return id
}

// Un id vuoto o ripetuto non fa rumore: AddDocument sovrascrive, e l'indice si
// riduce in silenzio a un documento solo. E' successo leggendo un corpus BEIR
// con la chiave sbagliata - 10.018 documenti diventati 1, e il confronto
// continuava a stampare tabelle come se niente fosse.
func controllaIdentificatori(docs []documento) error {
	visti := make(map[string]bool, len(docs))
	vuoti, ripetuti := 0, 0
	for _, d := range docs {
		if d.ID == "" {
			vuoti++
			continue
		}
		if visti[d.ID] {
			ripetuti++
		}
		visti[d.ID] = true
	}
	if vuoti > 0 || ripetuti > 0 {
		return fmt.Errorf("corpus inutilizzabile: %d documenti senza id, %d con id ripetuto, %d distinti su %d righe",
			vuoti, ripetuti, len(visti), len(docs))
	}
	return nil
}

func leggiCorpus(path string) ([]documento, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []documento
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		riga := strings.TrimSpace(sc.Text())
		if riga == "" {
			continue
		}
		var d documento
		if err := json.Unmarshal([]byte(riga), &d); err != nil {
			return nil, err
		}
		out = append(out, d)
	}

	return out, sc.Err()
}

func leggiRighe(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if riga := strings.TrimSpace(sc.Text()); riga != "" {
			out = append(out, riga)
		}
	}

	return out, sc.Err()
}
