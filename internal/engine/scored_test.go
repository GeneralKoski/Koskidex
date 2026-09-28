package engine

import (
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

func scoredTestIndex() (*InvertedIndex, Settings) {
	idx := NewInvertedIndex()
	settings := DefaultSettings()
	settings.SearchableFields = []string{"title"}

	idx.AddDocument("a", map[string]interface{}{"title": "gatto nero"}, settings)
	idx.AddDocument("b", map[string]interface{}{"title": "gatto"}, settings)
	idx.AddDocument("c", map[string]interface{}{"title": "cane"}, settings)

	return idx, settings
}

// SearchScored must return the same documents, in the same order, that Search
// returns: Search is only a wrapper that drops everything but the IDs.
func TestSearchScoredMatchesSearchOrder(t *testing.T) {
	idx, settings := scoredTestIndex()

	ids, _ := idx.Search("gatto nero", settings, "auto", nil)
	scored, _ := idx.SearchScored("gatto nero", settings, "auto", nil)

	if len(ids) != len(scored) {
		t.Fatalf("lunghezze diverse: Search %d, SearchScored %d", len(ids), len(scored))
	}
	for i := range ids {
		if ids[i] != scored[i].DocID {
			t.Fatalf("posizione %d: Search dice %q, SearchScored dice %q", i, ids[i], scored[i].DocID)
		}
	}
}

// The score is the object of the whole thesis: it has to actually come out.
func TestSearchScoredExposesTheScore(t *testing.T) {
	idx, settings := scoredTestIndex()

	scored, _ := idx.SearchScored("gatto nero", settings, "auto", nil)
	if len(scored) == 0 {
		t.Fatal("nessun risultato")
	}
	if scored[0].Score <= 0 {
		t.Fatalf("il primo risultato ha punteggio %v, atteso maggiore di zero", scored[0].Score)
	}
	for i := 1; i < len(scored); i++ {
		if scored[i-1].Score < scored[i].Score {
			t.Fatalf("punteggi non decrescenti: posizione %d ha %v, posizione %d ha %v",
				i-1, scored[i-1].Score, i, scored[i].Score)
		}
	}
}

// Both must deduplicate the highlights, not just Search.
func TestSearchScoredDeduplicatesHighlights(t *testing.T) {
	idx, settings := scoredTestIndex()

	_, fromSearch := idx.Search("gatto", settings, "auto", nil)
	_, fromScored := idx.SearchScored("gatto", settings, "auto", nil)

	for docID, terms := range fromScored {
		seen := map[string]bool{}
		for _, term := range terms {
			if seen[term] {
				t.Fatalf("documento %q: termine %q duplicato negli highlight", docID, term)
			}
			seen[term] = true
		}
	}
	if len(fromSearch) != len(fromScored) {
		t.Fatalf("highlight per %d documenti da Search, %d da SearchScored",
			len(fromSearch), len(fromScored))
	}
}

// The candidate step of a conjunctive search must change nothing: same
// documents, same order, same scores to the last bit, same highlights for
// every document returned. Random documents and queries, with typos,
// prefixes, repeated and missing words, under the settings that touch the
// must loop.
func TestCandidatiCongiuntiviNonCambianoNulla(t *testing.T) {
	confrontaCandidati(t, false)
}

// With StableTermOrder the vocabulary scan has a fixed order too, so every
// query is compared, none skipped.
func TestCandidatiCongiuntiviConOrdineFisso(t *testing.T) {
	confrontaCandidati(t, true)
}

func confrontaCandidati(t *testing.T, ordineFisso bool) {
	parole := []string{"gatto", "gatti", "gatta", "cane", "cani", "casa", "case", "cassa", "contratto",
		"contratti", "noleggio", "noleggi", "comune", "comunale", "delibera", "determina", "lavori",
		"lavoro", "strada", "strade", "manutenzione", "scuola", "scuole", "città", "perché", "2026", "123"}
	r := rand.New(rand.NewSource(7))
	frase := func(n int) string {
		out := make([]string, n)
		for i := range out {
			out[i] = parole[r.Intn(len(parole))]
		}
		return strings.Join(out, " ")
	}
	refuso := func(w string) string {
		rw := []rune(w)
		switch k := r.Intn(len(rw)); r.Intn(4) {
		case 0:
			return string(rw[:len(rw)-1])
		case 1:
			if k+1 < len(rw) {
				rw[k], rw[k+1] = rw[k+1], rw[k]
			}
		case 2:
			rw[k] = 'x'
		}
		return string(rw)
	}

	varianti := []func(*Settings){
		func(s *Settings) {},
		func(s *Settings) { s.ScoringMode = ScoringBM25 },
		func(s *Settings) { s.ScoringMode = ScoringBM25; s.BM25Expansion = BM25ExpansionBlended },
		func(s *Settings) { s.ScoringMode = ScoringBM25; s.AllTermsInOneField = true },
		func(s *Settings) { s.Coordination = true; s.DisablePrefixSearch = true; s.PrefixLength = 1 },
		func(s *Settings) {
			s.ScoringMode = ScoringBM25
			s.RetrievalMode = RetrievalAll
			s.FieldWeights = map[string]float64{"title": 5, "tags": 4}
		},
	}
	instabili, confrontate := 0, 0
	for v, varia := range varianti {
		settings := DefaultSettings()
		varia(&settings)
		settings.StableTermOrder = ordineFisso
		idx := NewInvertedIndex()
		for i := 0; i < 400; i++ {
			idx.AddDocument(fmt.Sprint(i), map[string]interface{}{
				"title": frase(1 + r.Intn(4)), "body": frase(r.Intn(30)), "tags": frase(r.Intn(3)),
			}, settings)
		}
		for q := 0; q < 700; q++ {
			termini := strings.Fields(frase(2 + r.Intn(4)))
			for i := range termini {
				switch r.Intn(6) {
				case 0:
					termini[i] = refuso(termini[i])
				case 1:
					termini[i] = string([]rune(termini[i])[:3])
				case 2:
					termini[i] = termini[0]
				}
			}
			if r.Intn(10) == 0 {
				termini = append(termini, "inesistente")
			}
			query := strings.Join(termini, " ")
			if r.Intn(8) == 0 {
				query += " -" + parole[r.Intn(len(parole))]
			}
			fuzziness := []string{"auto", "0", "1", "2"}[r.Intn(4)]

			// A short word with a typo allowed is matched by scanning the
			// vocabulary map, whose order changes from call to call: without
			// StableTermOrder the same search already differs from itself
			// there, candidates or not.
			instabile := false
			for _, tk := range Tokenize(query, "", settings) {
				d := MaxTypos(tk.Term, settings.TypoTolerance, fuzziness)
				instabile = instabile || (d > 0 && len([]rune(tk.Term))-1-3*d < 1)
			}
			if instabile && !ordineFisso {
				instabili++
				continue
			}
			veloce, hlVeloce := idx.SearchScored(query, settings, fuzziness, nil)
			senzaCandidati = true
			lento, hlLento := idx.SearchScored(query, settings, fuzziness, nil)
			senzaCandidati = false
			confrontate++

			if !reflect.DeepEqual(veloce, lento) {
				t.Fatalf("variante %d, %q (%s): risultati diversi\n  con i candidati %v\n  senza %v", v, query, fuzziness, veloce, lento)
			}
			for _, m := range veloce {
				if !reflect.DeepEqual(hlVeloce[m.DocID], hlLento[m.DocID]) {
					t.Fatalf("variante %d, %q: highlights di %s diversi: %v contro %v", v, query, m.DocID, hlVeloce[m.DocID], hlLento[m.DocID])
				}
			}
		}
	}
	t.Logf("%d ricerche confrontate, %d saltate perché instabili anche senza i candidati (scansione del vocabolario)", confrontate, instabili)
	minimo := 1500
	if ordineFisso {
		minimo = 4200
	}
	if confrontate < minimo {
		t.Fatalf("solo %d ricerche confrontate", confrontate)
	}
}

// A document holds two terms at the same distance from a short query word,
// with different frequencies: which one it is credited with decides its BM25
// score. With StableTermOrder the answer, and the highlights, never change,
// from one call to the next or with the documents added in another order,
// through the vocabulary scan and through substring_match alike.
func TestStableTermOrder(t *testing.T) {
	testi := map[string]string{
		"a": "cani cani cani cant",
		"b": "cant cant cani",
		"c": "casa cane",
		"d": "contratto cantiere canile",
	}
	riempi := func(ordine []string, settings Settings) *InvertedIndex {
		idx := NewInvertedIndex()
		for _, id := range ordine {
			idx.AddDocument(id, map[string]interface{}{"title": testi[id]}, settings)
		}
		return idx
	}
	for _, sottostringa := range []bool{false, true} {
		settings := DefaultSettings()
		settings.ScoringMode = ScoringBM25
		settings.StableTermOrder = true
		settings.SubstringMatch = sottostringa
		settings.RetrievalMode = RetrievalAny
		query := "canx"
		if sottostringa {
			query = "can"
		}
		primo, hlPrimo := riempi([]string{"a", "b", "c", "d"}, settings).SearchScored(query, settings, "1", nil)
		if len(primo) == 0 {
			t.Fatalf("%q: nessun risultato", query)
		}
		for i := 0; i < 200; i++ {
			ordine := []string{"d", "c", "b", "a"}
			if i%2 == 0 {
				ordine = []string{"a", "b", "c", "d"}
			}
			res, hl := riempi(ordine, settings).SearchScored(query, settings, "1", nil)
			if !reflect.DeepEqual(res, primo) || !reflect.DeepEqual(hl, hlPrimo) {
				t.Fatalf("%q, chiamata %d: %v %v, la prima dava %v %v", query, i, res, hl, primo, hlPrimo)
			}
		}
	}
}
