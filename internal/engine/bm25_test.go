package engine

import (
	"math"
	"testing"
)

// Same corpus as stats_test.go: N=3, avgdl=2.
//
//	a = "gatto gatto nero"  |a|=3
//	b = "gatto bianco"      |b|=2
//	c = "cane"              |c|=1
//	df(gatto)=2, df(nero)=df(cane)=1
func indiceBM25() (*InvertedIndex, Settings) {
	idx, s := indiceStatistiche()
	s.ScoringMode = ScoringBM25
	s.RetrievalMode = RetrievalAny
	s.TypoTolerance.Enabled = false
	return idx, s
}

func quasi(t *testing.T, got, atteso float64) {
	t.Helper()
	if math.Abs(got-atteso) > 1e-9 {
		t.Fatalf("atteso %.15f, ottenuto %.15f", atteso, got)
	}
}

// Values computed in Python before this code existed:
//
//	idf  = ln(1 + (N - df + 0.5)/(df + 0.5))
//	w    = idf * tf*(k1+1) / (tf + k1*(1 - b + b*|D|/avgdl))
func TestBM25MatchesTheHandComputedWeights(t *testing.T) {
	idx, s := indiceBM25()

	casi := []struct {
		query  string
		doc    string
		atteso float64
	}{
		{"gatto", "a", 0.5665797174469143},
		{"gatto", "b", 0.47000362924573563},
		{"cane", "c", 1.233042489500456},
		{"nero", "a", 0.8142733421229427},
	}

	for _, c := range casi {
		t.Run(c.query+" in "+c.doc, func(t *testing.T) {
			scored, _ := idx.SearchScored(c.query, s, "0", nil)
			for _, m := range scored {
				if m.DocID == c.doc {
					quasi(t, m.Score, c.atteso)
					return
				}
			}
			t.Fatalf("documento %q non trovato fra %v", c.doc, scored)
		})
	}
}

// The whole point of IDF. Under the legacy scorer these two weigh the same.
func TestBM25GivesMoreWeightToRarerTerms(t *testing.T) {
	idx, s := indiceBM25()

	comune, _ := idx.SearchScored("gatto", s, "0", nil) // df=2
	raro, _ := idx.SearchScored("nero", s, "0", nil)    // df=1

	var scoreComune, scoreRaro float64
	for _, m := range comune {
		if m.DocID == "a" {
			scoreComune = m.Score
		}
	}
	for _, m := range raro {
		if m.DocID == "a" {
			scoreRaro = m.Score
		}
	}

	if !(scoreRaro > scoreComune) {
		t.Fatalf("nello stesso documento il termine raro deve pesare di piu': raro %v, comune %v", scoreRaro, scoreComune)
	}

	// Sotto il punteggio legacy pesano identico: e' il difetto.
	s.ScoringMode = ScoringLegacy
	lc, _ := idx.SearchScored("gatto", s, "0", nil)
	lr, _ := idx.SearchScored("nero", s, "0", nil)
	if lc[0].Score != lr[0].Score {
		t.Fatalf("il legacy dovrebbe pesarli uguale, invece %v e %v", lc[0].Score, lr[0].Score)
	}
}

// Length normalisation: the same term, once, in a shorter document.
func TestBM25PrefersTheShorterDocumentAtEqualFrequency(t *testing.T) {
	idx := NewInvertedIndex()
	s := DefaultSettings()
	s.SearchableFields = []string{"testo"}
	s.ScoringMode = ScoringBM25
	s.RetrievalMode = RetrievalAny
	s.TypoTolerance.Enabled = false

	idx.AddDocument("corto", map[string]interface{}{"testo": "gatto"}, s)
	idx.AddDocument("lungo", map[string]interface{}{"testo": "gatto parola parola parola parola"}, s)

	scored, _ := idx.SearchScored("gatto", s, "0", nil)
	if len(scored) != 2 {
		t.Fatalf("attesi 2 risultati, ottenuti %v", scored)
	}
	if scored[0].DocID != "corto" {
		t.Fatalf("a parita' di frequenza deve vincere il documento corto, ha vinto %q", scored[0].DocID)
	}
}

// Frequency saturates: four occurrences are worth more than one, but nowhere
// near four times as much. It is what stops keyword stuffing.
func TestBM25SaturatesTermFrequency(t *testing.T) {
	punteggio := func(occorrenze int) float64 {
		idx := NewInvertedIndex()
		s := DefaultSettings()
		s.SearchableFields = []string{"testo"}
		s.ScoringMode = ScoringBM25
		s.RetrievalMode = RetrievalAny
		s.TypoTolerance.Enabled = false

		testo := "gatto"
		for i := 1; i < occorrenze; i++ {
			testo += " gatto"
		}
		// Un secondo documento tiene avgdl e df stabili fra le chiamate.
		idx.AddDocument("x", map[string]interface{}{"testo": testo}, s)

		scored, _ := idx.SearchScored("gatto", s, "0", nil)
		return scored[0].Score
	}

	uno, quattro := punteggio(1), punteggio(4)
	if !(quattro > uno) {
		t.Fatalf("piu' occorrenze devono valere di piu': %v contro %v", quattro, uno)
	}
	if quattro > 2.5*uno {
		t.Fatalf("la frequenza deve saturare: da %v a %v e' troppo per un fattore 4", uno, quattro)
	}
}

// Settings are persisted. An index saved before these fields existed reads
// them back as zero values, and must keep scoring exactly as it did.
func TestEmptyScoringModeBehavesAsLegacy(t *testing.T) {
	idx, s := indiceStatistiche()
	s.RetrievalMode = RetrievalAny
	s.TypoTolerance.Enabled = false

	s.ScoringMode = ScoringLegacy
	atteso, _ := idx.SearchScored("gatto", s, "0", nil)

	s.ScoringMode = ""
	got, _ := idx.SearchScored("gatto", s, "0", nil)

	if len(got) != len(atteso) {
		t.Fatalf("attesi %d risultati, ottenuti %d", len(atteso), len(got))
	}
	for i := range got {
		if got[i].DocID != atteso[i].DocID || got[i].Score != atteso[i].Score {
			t.Fatalf("una modalita' vuota deve comportarsi come legacy: %+v contro %+v", got[i], atteso[i])
		}
	}
}

// With k1 and b at zero BM25 collapses to pure IDF, which is a cheap way to
// check the parameters are actually read from the settings instead of being
// hardcoded.
func TestBM25ReadsItsParametersFromSettings(t *testing.T) {
	idx, s := indiceBM25()
	s.BM25K1 = 0
	s.BM25B = 0

	scored, _ := idx.SearchScored("gatto", s, "0", nil)
	if len(scored) == 0 {
		t.Fatal("nessun risultato")
	}
	// idf(df=2, N=3) = ln(1 + 1.5/2.5)
	quasi(t, scored[0].Score, math.Log(1+1.5/2.5))
}

// The tie frozen on 2026-09-23 closes here.
//
// On the identifier query the legacy scorer gives d1 and d3 exactly 48 each and
// cannot tell them apart, so the order came down to a document-ID tiebreaker.
// BM25 separates them, and it picks d3, the shorter document whose title is
// essentially the identifier, over the longer invoice. That is length
// normalisation doing precisely what it exists for.
func TestBM25BreaksTheIdentifierTieTheLegacyScorerCouldNot(t *testing.T) {
	idx, s := baselineIndex()

	s.ScoringMode = ScoringLegacy
	legacy, _ := idx.SearchScored("2026/0173", s, "auto", nil)
	if legacy[0].Score != legacy[1].Score {
		t.Fatalf("il legacy dovrebbe restare in parita': %v e %v", legacy[0].Score, legacy[1].Score)
	}

	s.ScoringMode = ScoringBM25
	bm25, _ := idx.SearchScored("2026/0173", s, "auto", nil)
	if len(bm25) != 2 {
		t.Fatalf("attesi 2 risultati, ottenuti %d", len(bm25))
	}
	if bm25[0].Score == bm25[1].Score {
		t.Fatalf("BM25 deve separarli, invece fanno entrambi %v", bm25[0].Score)
	}
	if bm25[0].DocID != "d3" {
		t.Fatalf("il documento piu' corto deve vincere, ha vinto %q con %v contro %v",
			bm25[0].DocID, bm25[0].Score, bm25[1].Score)
	}
}
