package engine

import "testing"

// I candidati fuzzy sono i termini che condividono un bigramma con la parola
// cercata. Due parole a un refuso di distanza possono non condividerne
// nessuno: una modifica rompe fino a due bigrammi, una trasposizione tre.
// Questi termini erano entro la distanza ammessa e non venivano mai valutati.
func TestTypoMatchesThatShareNoBigramAreFound(t *testing.T) {
	casi := []struct {
		doc, q string
		soglia int
		refusi string
		perche string
	}{
		{"arte", "atre", 4, "AUTO", "trasposizione in mezzo a quattro lettere, con le soglie predefinite"},
		{"ordinanza 17", "187", 3, "AUTO", "una cancellazione in mezzo a tre caratteri, con le soglie di Elasticsearch"},
		{"cat", "cut", 4, "1", "una sostituzione in mezzo a tre lettere, con fuzziness 1 esplicita"},
	}
	for _, c := range casi {
		idx := NewInvertedIndex()
		s := DefaultSettings()
		s.TypoTolerance.MinWordLengthOneTypo = c.soglia
		idx.AddDocument("1", map[string]interface{}{"t": c.doc}, s)
		if got, _ := idx.Search(c.q, s, c.refusi, nil); len(got) != 1 {
			t.Errorf("%q deve trovare %q (%s)", c.q, c.doc, c.perche)
		}
	}
}

// La correzione non deve allargare niente oltre la distanza ammessa.
func TestTypoMatchesStayWithinTheAllowedDistance(t *testing.T) {
	idx := NewInvertedIndex()
	s := DefaultSettings()
	s.TypoTolerance.MinWordLengthOneTypo = 3
	idx.AddDocument("1", map[string]interface{}{"t": "tra"}, s)
	idx.AddDocument("2", map[string]interface{}{"t": "abcdef"}, s)
	for _, q := range []string{"rat", "xyz"} {
		if got, _ := idx.Search(q, s, "AUTO", nil); len(got) != 0 {
			t.Errorf("%q dista due o più da ogni termine e non deve trovare niente: %v", q, got)
		}
	}
}
