// Command esmirror misura quante divergenze fra Koskidex ed Elasticsearch
// spiega ciascuna delle cause trovate: le prime quattro il 23/09/2026, il
// tokenizer il 25/09/2026.
//
// Da' a Koskidex esattamente cio' che Elasticsearch ha indicizzato e accende
// un'impostazione di compatibilita' alla volta, contando a ogni passo le query
// con lo stesso insieme di risultati e i documenti trovati da un motore solo.
//
//	go run ./scripts/esmirror <es-source.jsonl> <rapporto elasticsearch archiviato> <file di query>
//
// es-source.jsonl si produce con copia-indice-es.py (nella cartella
// dell'esperimento, nell'archivio della tesi) e contiene gli oggetti degli
// atti: va tenuto fuori da ogni repository. Il 23/09 lo stesso esperimento
// girava con una patch al motore; da quando le impostazioni esistono (Task F7)
// gira da codice committato.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/GeneralKoski/Koskidex/internal/engine"
	"github.com/GeneralKoski/Koskidex/internal/eval"
)

// I campi e i pesi di ElasticsearchService::fuzzySearch.
var campi = []string{"name", "tags", "summary", "subjects", "notes", "additional_data"}
var pesi = map[string]float64{"name": 5, "tags": 4, "summary": 3, "subjects": 3, "notes": 2, "additional_data": 1}

type passo struct {
	Nome        string `json:"nome"`
	PerCampo    bool   `json:"and_dentro_un_campo"`
	SoglieES    bool   `json:"soglie_refusi_es"`
	NoPrefisso  bool   `json:"niente_prefisso"`
	PrimaEsatta bool   `json:"prima_lettera_esatta"`
	Standard    bool   `json:"tokenizer_standard"`

	Identiche int      `json:"query_identiche"`
	Query     int      `json:"query"`
	SoloES    int      `json:"documenti_solo_es"`
	SoloKX    int      `json:"documenti_solo_koskidex"`
	Diverse   []string `json:"query_ancora_diverse"`
	IndexMs   float64  `json:"index_ms"`
}

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "uso: esmirror <es-source.jsonl> <rapporto elasticsearch> <file di query>")
		os.Exit(1)
	}
	docs := leggiSorgente(os.Args[1])

	var rap struct {
		Query []struct {
			Query string   `json:"query"`
			IDs   []string `json:"ids"`
		} `json:"query"`
	}
	b, err := os.ReadFile(os.Args[2])
	fallisci(err)
	fallisci(json.Unmarshal(b, &rap))
	es := map[string][]string{}
	for _, q := range rap.Query {
		es[q.Query] = q.IDs
	}
	query := leggiRighe(os.Args[3])
	for _, q := range query {
		if _, ok := es[q]; !ok {
			fallisci(fmt.Errorf("la query %q non e' nel rapporto di Elasticsearch", q))
		}
	}

	passi := []passo{
		{Nome: "0. Koskidex com'e', stessi campi di ES"},
		{Nome: "1. + AND dentro un campo solo (best_fields)", PerCampo: true},
		{Nome: "2. + soglie refusi di ES (3 -> 1, 6 -> 2)", PerCampo: true, SoglieES: true},
		{Nome: "3. + niente ricerca per prefisso", PerCampo: true, SoglieES: true, NoPrefisso: true},
		{Nome: "4. + prima lettera esatta (prefix_length 1)", PerCampo: true, SoglieES: true, NoPrefisso: true, PrimaEsatta: true},
		{Nome: "5. + tokenizer standard (dell'illuminazione)", PerCampo: true, SoglieES: true, NoPrefisso: true, PrimaEsatta: true, Standard: true},
	}

	for i := range passi {
		p := &passi[i]

		s := engine.DefaultSettings()
		s.SearchableFields = campi
		s.RetrievalMode = engine.RetrievalAll
		s.ScoringMode = engine.ScoringLegacy
		s.FieldWeights = pesi
		s.AllTermsInOneField = p.PerCampo
		if p.SoglieES {
			s.TypoTolerance.MinWordLengthOneTypo = 3
			s.TypoTolerance.MinWordLengthTwoTypos = 6
		}
		s.DisablePrefixSearch = p.NoPrefisso
		if p.PrimaEsatta {
			s.PrefixLength = 1
		}
		if p.Standard {
			s.Tokenizer = engine.TokenizerStandard
		}

		t0 := time.Now()
		idx := engine.NewInvertedIndex()
		for _, d := range docs {
			m := map[string]interface{}{}
			for _, c := range campi {
				m[c] = d[c]
			}
			idx.AddDocument(d["_id"], m, s)
		}
		p.IndexMs = float64(time.Since(t0).Nanoseconds()) / 1e6

		p.Query = len(query)
		for _, q := range query {
			kx := map[string]bool{}
			r, _ := idx.SearchScored(q, s, "auto", nil)
			for _, m := range r {
				kx[m.DocID] = true
			}
			e := map[string]bool{}
			for _, id := range es[q] {
				e[id] = true
			}
			soloES, soloKX := 0, 0
			for id := range e {
				if !kx[id] {
					soloES++
				}
			}
			for id := range kx {
				if !e[id] {
					soloKX++
				}
			}
			p.SoloES += soloES
			p.SoloKX += soloKX
			if soloES == 0 && soloKX == 0 {
				p.Identiche++
			} else {
				p.Diverse = append(p.Diverse, fmt.Sprintf("%s (-%d/+%d)", q, soloES, soloKX))
			}
		}
		sort.Strings(p.Diverse)
		fmt.Printf("%-48s identiche %2d/%d   documenti solo ES %4d   solo Koskidex %4d\n",
			p.Nome, p.Identiche, p.Query, p.SoloES, p.SoloKX)
	}

	esito := map[string]interface{}{
		"ran_at": time.Now().UTC().Format(time.RFC3339),
		"config": func() map[string]string {
			c := eval.Provenienza(".")
			c["rapporto_elasticsearch"] = os.Args[2]
			c["query_file"] = os.Args[3]
			c["documenti"] = fmt.Sprint(len(docs))
			c["metodo"] = "impostazioni di compatibilita' del motore (Task F7), un indice solo"
			return c
		}(),
		"passi": passi,
	}
	dati, err := json.MarshalIndent(esito, "", "  ")
	fallisci(err)
	path, err := eval.Archivia("esperimenti/2026-09-23_cause-divergenza", "esito", dati)
	fallisci(err)
	if path == "" {
		fmt.Printf("\n!!! RISULTATO NON ARCHIVIATO: %s non e' impostata.\n", eval.VarArchivio)
	} else {
		fmt.Println("\narchiviato in", path)
	}
}

func leggiSorgente(path string) []map[string]string {
	f, err := os.Open(path)
	fallisci(err)
	defer f.Close()
	var docs []map[string]string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		var d map[string]string
		fallisci(json.Unmarshal(sc.Bytes(), &d))
		docs = append(docs, d)
	}
	fallisci(sc.Err())
	return docs
}

func leggiRighe(path string) []string {
	f, err := os.Open(path)
	fallisci(err)
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if r := sc.Text(); r != "" {
			out = append(out, r)
		}
	}
	return out
}

func fallisci(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "errore:", err)
		os.Exit(1)
	}
}
