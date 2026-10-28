package engine

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"
)

// maxVarianti caps the rewritten queries one search can run.
// shortcut: past the cap the remaining combinations are dropped, raise it if
// queries with many synonym runs start missing documents.
const maxVarianti = 64

// Limits of Settings.SynonymGroups, checked by ValidaSinonimi: a search looks
// up the members that start like each query word, so their number is a cost
// every search can pay.
const (
	MaxSynonymGroups  = 500
	MaxSynonymMembers = 2000
)

// ValidaSinonimi refuses synonym settings past the limits, and a
// synonym_fields entry that is not searchable, which would switch synonyms
// off without saying so.
func ValidaSinonimi(s Settings) error {
	if len(s.SynonymGroups) > MaxSynonymGroups {
		return fmt.Errorf("synonym_groups: at most %d groups", MaxSynonymGroups)
	}
	membri := 0
	for _, g := range s.SynonymGroups {
		membri += len(g)
	}
	if membri > MaxSynonymMembers {
		return fmt.Errorf("synonym_groups: at most %d members in all", MaxSynonymMembers)
	}
	if len(s.SearchableFields) > 0 {
		for _, f := range s.SynonymFields {
			if !slices.Contains(s.SearchableFields, f) {
				return fmt.Errorf("synonym_fields: %q is not in searchable_fields", f)
			}
		}
	}
	return nil
}

type membro struct {
	gruppo int
	tokens []Token
	chiave string
}

// gruppiCompilati are the synonym groups tokenized once, with their members
// reachable from their first word.
type gruppiCompilati struct {
	gruppi       [][]membro
	perPrimo     map[string][]membro
	perLunghezza map[int][]string // distinct first words by length in runes
}

func compilaGruppi(s Settings) *gruppiCompilati {
	c := &gruppiCompilati{
		gruppi:       make([][]membro, len(s.SynonymGroups)),
		perPrimo:     make(map[string][]membro),
		perLunghezza: make(map[int][]string),
	}
	for i, g := range s.SynonymGroups {
		for _, testo := range g {
			t := Tokenize(testo, "", s)
			if len(t) == 0 {
				continue
			}
			m := membro{gruppo: i, tokens: t, chiave: chiave(t)}
			c.gruppi[i] = append(c.gruppi[i], m)
			primo := t[0].Term
			if _, ok := c.perPrimo[primo]; !ok {
				l := utf8.RuneCountInString(primo)
				c.perLunghezza[l] = append(c.perLunghezza[l], primo)
			}
			c.perPrimo[primo] = append(c.perPrimo[primo], m)
		}
	}
	return c
}

// candidati are the members whose first word the query word can match:
// the same word, or one whose length is within the typos it is allowed.
func (c *gruppiCompilati) candidati(parola string, maxTypos int) []membro {
	out := slices.Clone(c.perPrimo[parola])
	if maxTypos == 0 {
		return out
	}
	l := utf8.RuneCountInString(parola)
	for n := l - maxTypos; n <= l+maxTypos; n++ {
		for _, p := range c.perLunghezza[n] {
			if p != parola {
				out = append(out, c.perPrimo[p]...)
			}
		}
	}
	return out
}

// variantiSinonimi rewrites the query tokens with Settings.SynonymGroups:
// every run of tokens that matches a group member can be replaced by each of
// the other members of the groups it matches, and each combination is one
// more query to run. The query as typed is not among them. With no groups, or
// none matching, there is none and the search is the one of before.
func (s Settings) variantiSinonimi(tokens []Token, fuzziness string) [][]Token {
	if len(s.SynonymGroups) == 0 || len(tokens) == 0 {
		return nil
	}
	c := s.gruppiSinonimi()

	// Each piece is a run of the query with its options: the run itself
	// first, then its synonyms.
	var pezzi [][][]Token
	for i := 0; i < len(tokens); {
		// The longest member that matches here wins, as in Lucene's synonym
		// graph: "soft drink" is one run, not "soft" and then "drink".
		n := 0
		var gruppi []int
		for _, m := range c.candidati(tokens[i].Term, MaxTypos(tokens[i].Term, s.TypoTolerance, fuzziness)) {
			if len(m.tokens) < n || !s.corrisponde(tokens[i:], m.tokens, fuzziness) {
				continue
			}
			if len(m.tokens) > n {
				n, gruppi = len(m.tokens), nil
			}
			gruppi = append(gruppi, m.gruppo)
		}
		// In group order, so that the rewrites, and the cap, do not depend on
		// the order the members were found in.
		slices.Sort(gruppi)
		gruppi = slices.Compact(gruppi)

		run := tokens[i : i+max(n, 1)]
		opzioni := [][]Token{run}
		visti := map[string]bool{chiave(run): true}
		for _, g := range gruppi {
			for _, m := range c.gruppi[g] {
				// A member the run itself matches adds nothing: the run
				// finds it, typos included. A shorter one ("gas" for the
				// run "gas naturale") is a real alternative.
				if visti[m.chiave] || (len(m.tokens) == n && s.corrisponde(run, m.tokens, fuzziness)) {
					continue
				}
				visti[m.chiave] = true
				alt := make([]Token, len(m.tokens))
				for k, t := range m.tokens {
					alt[k] = Token{Term: t.Term, Sinonimo: true}
				}
				opzioni = append(opzioni, alt)
			}
		}
		pezzi = append(pezzi, opzioni)
		i += len(run)
	}

	varianti := [][]Token{nil}
	for _, opzioni := range pezzi {
		var nuove [][]Token
		for _, v := range varianti {
			for _, o := range opzioni {
				if len(nuove) < maxVarianti {
					nuove = append(nuove, append(slices.Clip(v), o...))
				}
			}
		}
		varianti = nuove
	}
	return varianti[1:]
}

// corrisponde says whether the query starts with member m, word by word,
// each word equal or within the typos the index would allow it: no more than
// MaxTypos, and the first PrefixLength characters exact.
func (s Settings) corrisponde(query, m []Token, fuzziness string) bool {
	if len(m) == 0 || len(m) > len(query) {
		return false
	}
	for k, t := range m {
		q := query[k].Term
		if q == t.Term {
			continue
		}
		esatto := []rune(q)
		if s.PrefixLength < len(esatto) {
			esatto = esatto[:s.PrefixLength]
		}
		if !strings.HasPrefix(t.Term, string(esatto)) || DamerauLevenshtein(q, t.Term) > MaxTypos(q, s.TypoTolerance, fuzziness) {
			return false
		}
	}
	return true
}

func chiave(tokens []Token) string {
	var k string
	for _, t := range tokens {
		k += t.Term + " "
	}
	return k
}
