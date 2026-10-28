package engine

import (
	"slices"
	"testing"
)

func indiceSinonimi(t *testing.T, docs map[string]string) *InvertedIndex {
	t.Helper()
	idx := NewInvertedIndex()
	// Indexed without any synonym: the groups must work at query time alone.
	s := DefaultSettings()
	s.SearchableFields = []string{"name"}
	for id, name := range docs {
		idx.AddDocument(id, map[string]interface{}{"id": id, "name": name}, s)
	}
	return idx
}

func conGruppi(gruppi ...[]string) Settings {
	s := DefaultSettings()
	s.SearchableFields = []string{"name"}
	s.SynonymGroups = gruppi
	return s
}

func conSinonimi(idx *InvertedIndex, q string, s Settings) []string {
	ids, _ := idx.Search(q, s, "", nil)
	slices.Sort(ids)
	return ids
}

func TestSynonymGroupsWorkBothWaysAtQueryTime(t *testing.T) {
	idx := indiceSinonimi(t, map[string]string{"1": "Bibite gassate", "2": "Bevande calde", "3": "Pasta"})
	s := conGruppi([]string{"bibite", "bevande"})
	for _, q := range []string{"bibite", "bevande"} {
		if got := conSinonimi(idx, q, s); !slices.Equal(got, []string{"1", "2"}) {
			t.Errorf("%q: got %v, want [1 2]", q, got)
		}
	}
}

func TestSynonymGroupsTakeMultiWordMembers(t *testing.T) {
	idx := indiceSinonimi(t, map[string]string{"1": "Bibite gassate", "2": "Soft drink al limone", "3": "Drink analcolico"})
	s := conGruppi([]string{"bibite", "soft drink"})
	if got := conSinonimi(idx, "soft drink", s); !slices.Equal(got, []string{"1", "2"}) {
		t.Errorf("soft drink: got %v, want [1 2]", got)
	}
	// "soft drink" as a synonym needs both words: "Drink analcolico" stays out.
	if got := conSinonimi(idx, "bibite", s); !slices.Equal(got, []string{"1", "2"}) {
		t.Errorf("bibite: got %v, want [1 2]", got)
	}
	// The other query words still apply to every rewrite.
	if got := conSinonimi(idx, "soft drink limone", s); !slices.Equal(got, []string{"2"}) {
		t.Errorf("soft drink limone: got %v, want [2]", got)
	}
}

func TestSynonymGroupsFireOnATypoButMatchSynonymsAsWritten(t *testing.T) {
	idx := indiceSinonimi(t, map[string]string{"1": "Pescheria", "2": "Pesca gialla", "3": "Pesce spada"})
	s := conGruppi([]string{"pesce", "pescheria"})
	// "pese" is one typo from "pesce": the group fires and finds the zone.
	if got := conSinonimi(idx, "pese", s); !slices.Contains(got, "1") {
		t.Errorf("pese: got %v, want the Pescheria among them", got)
	}
	// "pesce" as a synonym of "pescheria" must not reach "pesca" by a typo.
	if got := conSinonimi(idx, "pescheria", s); !slices.Equal(got, []string{"1", "3"}) {
		t.Errorf("pescheria: got %v, want [1 3]", got)
	}
}

func TestSynonymGroupsLiveBesideTheIndexTimeMap(t *testing.T) {
	idx := NewInvertedIndex()
	s := DefaultSettings()
	s.SearchableFields = []string{"name"}
	s.Synonyms = map[string][]string{"tv": {"televisore"}}
	idx.AddDocument("1", map[string]interface{}{"id": "1", "name": "tv"}, s)
	idx.AddDocument("2", map[string]interface{}{"id": "2", "name": "schermo"}, s)
	s.SynonymGroups = [][]string{{"televisore", "schermo"}}
	if got := conSinonimi(idx, "televisore", s); !slices.Equal(got, []string{"1", "2"}) {
		t.Errorf("televisore: got %v, want [1 2]", got)
	}
}

func TestWithoutSynonymGroupsTheQueryIsNotRewritten(t *testing.T) {
	s := DefaultSettings()
	tokens := Tokenize("soft drink", "", s)
	if v := s.variantiSinonimi(tokens, ""); len(v) != 0 {
		t.Fatalf("no groups: got %v", v)
	}
	s.SynonymGroups = [][]string{{"pane", "panetteria"}}
	if v := s.variantiSinonimi(tokens, ""); len(v) != 0 {
		t.Fatalf("no group matches: got %v", v)
	}
}

func TestOnlyQueryTimeSettingsKeepTheSameIndex(t *testing.T) {
	a := DefaultSettings()
	b := DefaultSettings()
	b.SynonymGroups = [][]string{{"bibite", "bevande"}}
	if !a.StessoIndice(b) {
		t.Error("synonym_groups alone must not need a reindex")
	}
	b.Synonyms = map[string][]string{"bibite": {"bevande"}}
	if a.StessoIndice(b) {
		t.Error("the index-time synonyms map needs a reindex")
	}
}
