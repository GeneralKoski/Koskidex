package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/GeneralKoski/Koskidex/internal/engine"
)

var campi = []string{"name", "tags", "summary", "subjects", "notes", "additional_data"}

func main() {
	f, _ := os.Open(os.Args[1])
	var docs []map[string]string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		var d map[string]string
		json.Unmarshal(sc.Bytes(), &d)
		docs = append(docs, d)
	}
	byID := map[string]map[string]string{}
	var rap struct{ Query []struct{ Query string; IDs []string } `json:"query"` }
	b, _ := os.ReadFile(os.Args[2])
	json.Unmarshal(b, &rap)
	s := engine.DefaultSettings()
	s.SearchableFields = campi
	s.RetrievalMode = engine.RetrievalAll
	s.AllTermsInOneField = true
	s.TypoTolerance.MinWordLengthOneTypo, s.TypoTolerance.MinWordLengthTwoTypos = 3, 6
	s.DisablePrefixSearch, s.PrefixLength = true, 1
	idx := engine.NewInvertedIndex()
	for _, d := range docs {
		m := map[string]interface{}{}
		for _, c := range campi {
			m[c] = d[c]
		}
		idx.AddDocument(d["_id"], m, s)
		byID[d["_id"]] = d
	}
	for _, q := range rap.Query {
		if !strings.Contains(os.Args[3], q.Query) {
			continue
		}
		es := map[string]bool{}
		for _, id := range q.IDs {
			es[id] = true
		}
		r, _ := idx.SearchScored(q.Query, s, "auto", nil)
		fmt.Printf("\n== %s\n", q.Query)
		for _, m := range r {
			if es[m.DocID] {
				continue
			}
			fmt.Printf("  %s\n", m.DocID)
			for _, tok := range engine.Tokenize(q.Query, "", s) {
				for _, c := range campi {
					for _, dt := range engine.Tokenize(byID[m.DocID][c], c, s) {
						if d := engine.DamerauLevenshtein(tok.Term, dt.Term); d <= engine.MaxTypos(tok.Term, s.TypoTolerance, "auto") && dt.Term[0] == tok.Term[0] {
							fmt.Printf("     %-14s -> %-16s d=%d  in %s\n", tok.Term, dt.Term, d, c)
						}
					}
				}
			}
		}
	}
}
