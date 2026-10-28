package engine

import "slices"

// maxVarianti caps the rewritten queries one search can run.
// shortcut: past the cap the remaining combinations are dropped, raise it if
// queries with many synonym runs start missing documents.
const maxVarianti = 64

// variantiSinonimi rewrites the query tokens with Settings.SynonymGroups:
// every run of tokens that matches a group member can be replaced by each of
// the other members of the groups it matches, and each combination is one
// more query to run. The query as typed is not among them. With no groups, or
// none matching, it is nil and the search is the one of before.
func (s Settings) variantiSinonimi(tokens []Token, fuzziness string) [][]Token {
	if len(s.SynonymGroups) == 0 || len(tokens) == 0 {
		return nil
	}
	gruppi := make([][][]Token, len(s.SynonymGroups))
	for i, g := range s.SynonymGroups {
		for _, m := range g {
			gruppi[i] = append(gruppi[i], Tokenize(m, "", s))
		}
	}

	// Each piece is a run of the query with its options: the run itself
	// first, then its synonyms.
	var pezzi [][][]Token
	for i := 0; i < len(tokens); {
		// The longest member that matches here wins, as in Lucene's synonym
		// graph: "soft drink" is one run, not "soft" and then "drink".
		n := 0
		for _, g := range gruppi {
			for _, m := range g {
				if len(m) > n && s.corrisponde(tokens[i:], m, fuzziness) {
					n = len(m)
				}
			}
		}
		opzioni := [][]Token{tokens[i : i+max(n, 1)]}
		visti := map[string]bool{chiave(opzioni[0]): true}
		for _, g := range gruppi {
			if !slices.ContainsFunc(g, func(m []Token) bool { return len(m) == n && s.corrisponde(tokens[i:], m, fuzziness) }) {
				continue
			}
			for _, m := range g {
				// A member the run already matches adds nothing: the run
				// finds it, typos included.
				if len(m) == 0 || visti[chiave(m)] || s.corrisponde(tokens[i:], m, fuzziness) {
					continue
				}
				visti[chiave(m)] = true
				alt := make([]Token, len(m))
				for k, t := range m {
					alt[k] = Token{Term: t.Term, Sinonimo: true}
				}
				opzioni = append(opzioni, alt)
			}
		}
		pezzi = append(pezzi, opzioni)
		i += max(n, 1)
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
// each word equal or within the typo tolerance the index would allow it.
func (s Settings) corrisponde(query, m []Token, fuzziness string) bool {
	if len(m) == 0 || len(m) > len(query) {
		return false
	}
	for k, t := range m {
		if query[k].Term != t.Term && DamerauLevenshtein(query[k].Term, t.Term) > MaxTypos(query[k].Term, s.TypoTolerance, fuzziness) {
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
