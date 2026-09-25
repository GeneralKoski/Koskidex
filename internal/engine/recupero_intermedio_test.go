package engine

import (
	"sort"
	"testing"
)

func TestIntermediateRetrievalIsOffByDefault(t *testing.T) {
	s := DefaultSettings()
	if s.MinimumShouldMatch != "" || s.Coordination {
		t.Fatal("minimum_should_match e coordinazione devono essere spenti per default: il baseline deve restare misurabile")
	}
}

// I casi della documentazione di minimum_should_match di Elasticsearch, e il
// calcolo di Queries.calculateMinShouldMatch che la implementa.
func TestRequiredTermsFollowsElasticsearch(t *testing.T) {
	casi := []struct {
		spec   string
		n      int
		attesi int
	}{
		{"3", 5, 3},
		{"3", 2, 2}, // mai più dei termini
		{"-2", 5, 3},
		{"-7", 5, 1}, // mai meno di uno
		{"75%", 4, 3},
		{"75%", 3, 2}, // 2,25 arrotondato per difetto
		{"75%", 1, 1},
		{"-25%", 4, 3},
		{"-25%", 3, 3}, // 0,75 per difetto è 0: non ne toglie nessuno
		{"-25%", 5, 4},
		{"3<90%", 3, 3},
		{"3<90%", 5, 4},
		{"2<-25% 9<-3", 1, 1},
		{"2<-25% 9<-3", 2, 2},
		{"2<-25% 9<-3", 3, 3},
		{"2<-25% 9<-3", 4, 3},
		{"2<-25% 9<-3", 9, 7},
		{"2<-25% 9<-3", 10, 7},
		{"2<-25% 9<-3", 12, 9},
		{" 2 < -25%  9 < -3 ", 12, 9},
	}
	for _, c := range casi {
		got, err := RequiredTerms(c.spec, c.n)
		if err != nil {
			t.Errorf("%q con %d termini: errore %v", c.spec, c.n, err)
			continue
		}
		if got != c.attesi {
			t.Errorf("%q con %d termini: %d, attesi %d", c.spec, c.n, got, c.attesi)
		}
	}
}

func TestRequiredTermsRefusesAMalformedSpec(t *testing.T) {
	for _, spec := range []string{"tre", "75.5%", "%", "2<", "<75%", "2<75%<3", "2<x"} {
		if _, err := RequiredTerms(spec, 5); err == nil {
			t.Errorf("%q deve essere rifiutata", spec)
		}
	}
	if n, err := RequiredTerms("", 5); err != nil || n != 1 {
		t.Errorf("vuota vuol dire un termine basta: %d, %v", n, err)
	}
}

func indiceIntermedio(modifica func(*Settings)) (*InvertedIndex, Settings) {
	idx := NewInvertedIndex()
	s := DefaultSettings()
	s.SearchableFields = []string{"t"}
	s.RetrievalMode = RetrievalAny
	s.ScoringMode = ScoringBM25
	s.TypoTolerance.Enabled = false
	s.DisablePrefixSearch = true
	if modifica != nil {
		modifica(&s)
	}
	for id, testo := range map[string]string{
		"giusto":   "determina 1209 crispiano",
		"ripete":   "crispiano crispiano crispiano",
		"altro":    "avviso 77 lecce",
		"numero":   "delibera 1209 lecce",
		"quattro1": "alfa beta gamma delta",
		"quattro2": "alfa beta gamma",
		"quattro3": "alfa beta",
	} {
		idx.AddDocument(id, map[string]interface{}{"t": testo}, s)
	}
	return idx, s
}

func risultati(idx *InvertedIndex, s Settings, q string) ([]string, map[string]float64) {
	r, _ := idx.SearchScored(q, s, "0", nil)
	ordine, punti := []string{}, map[string]float64{}
	for _, m := range r {
		ordine = append(ordine, m.DocID)
		punti[m.DocID] = m.Score
	}
	return ordine, punti
}

func TestMinimumShouldMatchDropsDocumentsWithTooFewTerms(t *testing.T) {
	idx, s := indiceIntermedio(func(s *Settings) { s.MinimumShouldMatch = "2<-25% 9<-3" })
	got, _ := risultati(idx, s, "1209 crispiano")
	if len(got) != 1 || got[0] != "giusto" {
		t.Fatalf("con due termini li vuole tutti e due: %v", got)
	}
	got, _ = risultati(idx, s, "alfa beta gamma delta")
	sort.Strings(got)
	if len(got) != 2 || got[0] != "quattro1" || got[1] != "quattro2" {
		t.Fatalf("con quattro termini ne vuole tre: %v", got)
	}
}

func TestMinimumShouldMatchOnlyNarrowsAnyRetrieval(t *testing.T) {
	idx, s := indiceIntermedio(func(s *Settings) {
		s.MinimumShouldMatch = "1"
		s.RetrievalMode = RetrievalAll
	})
	if got, _ := risultati(idx, s, "1209 crispiano"); len(got) != 1 {
		t.Fatalf("con all resta congiuntivo, qualunque sia la specifica: %v", got)
	}
}

func TestCoordinationScalesTheScoreByTheShareOfTermsMatched(t *testing.T) {
	oggi, s0 := indiceIntermedio(nil)
	coord, s1 := indiceIntermedio(func(s *Settings) { s.Coordination = true })
	ordine0, p0 := risultati(oggi, s0, "1209 crispiano")
	ordine1, p1 := risultati(coord, s1, "1209 crispiano")
	if len(ordine0) != len(ordine1) {
		t.Fatalf("la coordinazione non toglie documenti: %v contro %v", ordine0, ordine1)
	}
	quasi(t, p1["giusto"], p0["giusto"])
	quasi(t, p1["ripete"], p0["ripete"]/2)
	quasi(t, p1["numero"], p0["numero"]/2)
	if ordine1[0] != "giusto" {
		t.Fatalf("con tutti e due i termini deve essere primo: %v", ordine1)
	}
}
