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

// Korsia: "Bevande e vini" is the zone of every drink, Lambrusco included.
func TestSynonymFieldsKeepSynonymsOutOfTheOtherFields(t *testing.T) {
	docs := []map[string]interface{}{
		{"id": "1", "name": "Lambrusco", "category_label": "Vini", "zone_names": "Bevande e vini"},
		{"id": "2", "name": "Coca-Cola", "category_label": "Bevande", "zone_names": "Bevande e vini"},
	}
	for _, forma := range []string{"synonym_groups", "synonyms"} {
		s := DefaultSettings()
		s.SearchableFields = []string{"name", "category_label", "zone_names"}
		if forma == "synonyms" {
			s.Synonyms = map[string][]string{"bevande": {"bibite"}}
		} else {
			s.SynonymGroups = [][]string{{"bibite", "bevande"}}
		}
		indicizza := func(s Settings) *InvertedIndex {
			idx := NewInvertedIndex()
			for _, d := range docs {
				idx.AddDocument(d["id"].(string), d, s)
			}
			return idx
		}

		if got := conSinonimi(indicizza(s), "bibite", s); !slices.Equal(got, []string{"1", "2"}) {
			t.Errorf("%s, every field: got %v, want [1 2]", forma, got)
		}
		s.SynonymFields = []string{"name", "category_label"}
		idx := indicizza(s)
		if got := conSinonimi(idx, "bibite", s); !slices.Equal(got, []string{"2"}) {
			t.Errorf("%s, name and category_label: got %v, want [2]", forma, got)
		}
		// The word as typed still matches in every field.
		if got := conSinonimi(idx, "bevande", s); !slices.Equal(got, []string{"1", "2"}) {
			t.Errorf("%s, bevande: got %v, want [1 2]", forma, got)
		}
	}
}

// Every member reaches every other, also when one is the start of another.
func TestEveryMemberOfAGroupReachesEveryOther(t *testing.T) {
	idx := indiceSinonimi(t, map[string]string{"1": "Gas", "2": "Gas naturale", "3": "Metano"})
	s := conGruppi([]string{"gas naturale", "gas", "metano"})
	for _, q := range []string{"gas naturale", "gas", "metano"} {
		if got := conSinonimi(idx, q, s); !slices.Equal(got, []string{"1", "2", "3"}) {
			t.Errorf("%q: got %v, want [1 2 3]", q, got)
		}
	}
	// The same once the groups are compiled, as the manager stores them.
	s.Prepara()
	if got := conSinonimi(idx, "gas naturale", s); !slices.Equal(got, []string{"1", "2", "3"}) {
		t.Errorf("prepared, gas naturale: got %v, want [1 2 3]", got)
	}
}

// A group fires only on a typo the index would accept.
func TestSynonymGroupsKeepThePrefixLength(t *testing.T) {
	idx := indiceSinonimi(t, map[string]string{"1": "Pescheria", "2": "Pesce spada"})
	s := conGruppi([]string{"pesce", "pescheria"})
	s.PrefixLength = 1
	if got := conSinonimi(idx, "lesce", s); len(got) != 0 {
		t.Errorf("lesce with prefix_length 1: got %v, want nothing", got)
	}
	if got := conSinonimi(idx, "pesse", s); !slices.Equal(got, []string{"1", "2"}) {
		t.Errorf("pesse with prefix_length 1: got %v, want [1 2]", got)
	}
}

func TestSynonymSettingsPastTheLimitsAreRefused(t *testing.T) {
	s := DefaultSettings()
	s.SynonymGroups = make([][]string, MaxSynonymGroups+1)
	if ValidaSinonimi(s) == nil {
		t.Error("too many groups must be refused")
	}
	s.SynonymGroups = [][]string{make([]string, MaxSynonymMembers+1)}
	if ValidaSinonimi(s) == nil {
		t.Error("too many members must be refused")
	}
	s.SynonymGroups = [][]string{{"bibite", "bevande"}}
	s.SearchableFields = []string{"name"}
	s.SynonymFields = []string{"nmae"}
	if ValidaSinonimi(s) == nil {
		t.Error("a synonym field that is not searchable must be refused")
	}
	s.SynonymFields = []string{"name"}
	if err := ValidaSinonimi(s); err != nil {
		t.Error(err)
	}
}
