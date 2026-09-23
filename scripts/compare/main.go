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
	"strings"

	"github.com/GeneralKoski/Koskidex/internal/engine"
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
	etichetta string
	recupero  string
	punteggio string
}

func main() {
	corpus := flag.String("corpus", "", "file JSONL esportato da Documentale")
	queries := flag.String("queries", "", "file con una query per riga")
	quante := flag.Int("top", 5, "quanti risultati mostrare per query")
	uscita := flag.String("json", "", "se valorizzato, scrive i ranking anche in questo file")
	// Il default e' "auto" perche' Documentale cerca con fuzziness AUTO: con la
	// tolleranza spenta i due motori recuperano insiemi diversi e la tabella
	// sembra dire che Koskidex si comporta diversamente, mentre sta solo
	// girando con una configurazione che nessuno usa.
	refusi := flag.String("fuzziness", "auto", "tolleranza ai refusi: auto come Elasticsearch in produzione, 0 per spegnerla")
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
		{"oggi (and + euristico)", engine.RetrievalAll, engine.ScoringLegacy},
		{"or + euristico", engine.RetrievalAny, engine.ScoringLegacy},
		{"or + BM25", engine.RetrievalAny, engine.ScoringBM25},
	}

	fmt.Printf("%d documenti, %d query, tolleranza refusi %q\n", len(docs), len(domande), *refusi)

	vuote := make([]int, len(configurazioni))
	raccolta := map[string]map[string][]risultato{}

	for _, q := range domande {
		raccolta[q] = map[string][]risultato{}
		fmt.Printf("\n\n== %s  (%d parole)\n", q, len(strings.Fields(q)))
		for i, c := range configurazioni {
			idx, s := indicizza(docs, c, *refusi)
			trovati, _ := idx.SearchScored(q, s, *refusi, nil)
			if len(trovati) == 0 {
				vuote[i]++
			}

			for _, m := range trovati {
				raccolta[q][c.etichetta] = append(raccolta[q][c.etichetta], risultato{
					DocID:      m.DocID,
					Punteggio:  m.Score,
					Intestaz10: etichetta(docs, m.DocID),
				})
			}

			fmt.Printf("\n  %-24s %d risultati\n", c.etichetta, len(trovati))
			for n, m := range trovati {
				if n >= *quante {
					fmt.Printf("      ... e altri %d\n", len(trovati)-*quante)
					break
				}
				fmt.Printf("      %d. %-6.3f %s\n", n+1, m.Score, etichetta(docs, m.DocID))
			}
		}
	}

	fmt.Printf("\n\n== query a vuoto su %d\n", len(domande))
	for i, c := range configurazioni {
		fmt.Printf("  %-24s %d\n", c.etichetta, vuote[i])
	}

	if *uscita != "" {
		f, err := os.Create(*uscita)
		if err != nil {
			fmt.Fprintln(os.Stderr, "errore:", err)
			os.Exit(1)
		}
		defer f.Close()
		enc := json.NewEncoder(f)
		enc.SetIndent("", "  ")
		if err := enc.Encode(raccolta); err != nil {
			fmt.Fprintln(os.Stderr, "errore:", err)
			os.Exit(1)
		}
	}
}

// risultato e' una riga di ranking come finisce nel file JSON.
type risultato struct {
	DocID      string  `json:"doc_id"`
	Punteggio  float64 `json:"punteggio"`
	Intestaz10 string  `json:"titolo"`
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
