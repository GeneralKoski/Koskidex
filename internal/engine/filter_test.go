package engine

import "testing"

func TestParseFilters(t *testing.T) {
	filters := ParseFilters("year>=2000, genre=Sci-Fi")
	if len(filters) != 2 {
		t.Fatalf("expected 2 filters, got %d", len(filters))
	}
	if filters[0].Field != "year" || filters[0].Operator != ">=" || filters[0].Value != "2000" {
		t.Fatalf("unexpected first filter: %+v", filters[0])
	}
	if filters[1].Field != "genre" || filters[1].Operator != "=" || filters[1].Value != "Sci-Fi" {
		t.Fatalf("unexpected second filter: %+v", filters[1])
	}

	if ParseFilters("") != nil {
		t.Fatal("expected nil for empty filter string")
	}
}

func TestApplyFiltersNumericAndString(t *testing.T) {
	doc := map[string]interface{}{
		"year":  float64(1999),
		"genre": "Sci-Fi",
	}

	if !ApplyFilters(doc, ParseFilters("year>=1999")) {
		t.Fatal("expected year>=1999 to match")
	}
	if ApplyFilters(doc, ParseFilters("year>2000")) {
		t.Fatal("expected year>2000 not to match")
	}
	// String equality is case-insensitive.
	if !ApplyFilters(doc, ParseFilters("genre=sci-fi")) {
		t.Fatal("expected case-insensitive genre match")
	}
	if !ApplyFilters(doc, ParseFilters("genre!=Horror")) {
		t.Fatal("expected genre!=Horror to match")
	}
	// Missing field never matches.
	if ApplyFilters(doc, ParseFilters("rating>4")) {
		t.Fatal("expected missing field to fail the filter")
	}
}
