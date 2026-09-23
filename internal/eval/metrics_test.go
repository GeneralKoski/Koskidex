package eval

import (
	"math"
	"testing"
)

// Expected values computed independently in Python before the Go code existed,
// so a mistake in the implementation cannot be inherited by its own test:
//
//	dcg(rels) = sum(r / log2(i+2))
//	giudizi   = {d1:3, d2:2, d3:1}, d4 unjudged
var giudizi = relevance{"d1": 3, "d2": 2, "d3": 1}

func vicino(t *testing.T, got, atteso float64) {
	t.Helper()
	if math.Abs(got-atteso) > 1e-9 {
		t.Fatalf("atteso %.15f, ottenuto %.15f", atteso, got)
	}
}

func TestNDCG(t *testing.T) {
	casi := []struct {
		nome   string
		ranked []string
		k      int
		atteso float64
	}{
		// DCG = 2/log2(2) + 0/log2(3) + 3/log2(4) = 3.5
		// IDCG = 3/log2(2) + 2/log2(3) + 1/log2(4) = 4.7618595071429155
		{"ranking misto con un non giudicato", []string{"d2", "d4", "d1"}, 3, 0.7350069851388743},

		{"ranking perfetto", []string{"d1", "d2", "d3"}, 3, 1.0},

		// Scambiare il grado 2 col grado 1 costa poco ma non zero: e' la
		// differenza che il gain lineare e quello esponenziale pesano diverso.
		{"gradi 1 e 2 invertiti", []string{"d1", "d3", "d2"}, 3, 0.9725044904464192},

		// A k=2 anche l'ideale si tronca a 2, quindi IDCG cambia.
		{"troncato a k=2", []string{"d2", "d4"}, 2, 0.46927872602275644},

		{"nessun documento rilevante trovato", []string{"d4", "d5"}, 3, 0.0},
		{"ranking vuoto", nil, 3, 0.0},
		{"k non valido", []string{"d1"}, 0, 0.0},
	}

	for _, c := range casi {
		t.Run(c.nome, func(t *testing.T) {
			vicino(t, NDCG(c.ranked, giudizi, c.k), c.atteso)
		})
	}
}

// A query with judgments but none of them relevant has nothing a ranking could
// have achieved: the score is 0, not NaN from a division by zero.
func TestNDCGWithNoRelevantDocuments(t *testing.T) {
	got := NDCG([]string{"d1"}, relevance{"d1": 0, "d2": 0}, 10)
	if got != 0 {
		t.Fatalf("atteso 0, ottenuto %v", got)
	}
}

// Fewer relevant documents than k still allows a perfect score: the ideal is
// built from what exists, not from k.
func TestNDCGReachesOneWithFewerRelevantThanK(t *testing.T) {
	vicino(t, NDCG([]string{"d1", "d9", "d8"}, relevance{"d1": 1}, 10), 1.0)
}

// Unjudged documents cost position: the same relevant document lower down has
// to score less.
func TestNDCGPenalisesPaddingWithUnjudgedDocuments(t *testing.T) {
	alto := NDCG([]string{"d1", "x", "y"}, relevance{"d1": 1}, 10)
	basso := NDCG([]string{"x", "y", "d1"}, relevance{"d1": 1}, 10)
	if !(alto > basso) {
		t.Fatalf("il documento piu' in alto deve valere di piu': %v contro %v", alto, basso)
	}
	vicino(t, alto, 1.0)
	vicino(t, basso, 1/math.Log2(4))
}

func TestRecall(t *testing.T) {
	rel := relevance{"d1": 1, "d2": 2, "d3": 1, "d0": 0}

	casi := []struct {
		nome   string
		ranked []string
		k      int
		atteso float64
	}{
		{"due su tre", []string{"d1", "x", "d2"}, 3, 2.0 / 3.0},
		{"tutti", []string{"d1", "d2", "d3"}, 3, 1.0},
		{"il taglio esclude il terzo", []string{"d1", "d2", "d3"}, 2, 2.0 / 3.0},
		{"nessuno", []string{"x", "y"}, 2, 0.0},
		// d0 e' giudicato 0: non conta ne' al numeratore ne' al denominatore.
		{"il giudizio 0 non e' un rilevante", []string{"d0"}, 3, 0.0},
	}

	for _, c := range casi {
		t.Run(c.nome, func(t *testing.T) {
			vicino(t, Recall(c.ranked, rel, c.k), c.atteso)
		})
	}
}

func TestMRR(t *testing.T) {
	rel := relevance{"d1": 1, "d2": 1}

	casi := []struct {
		nome   string
		ranked []string
		k      int
		atteso float64
	}{
		{"primo posto", []string{"d1", "x"}, 10, 1.0},
		{"secondo posto", []string{"x", "d1"}, 10, 0.5},
		{"terzo posto", []string{"x", "y", "d2"}, 10, 1.0 / 3.0},
		{"oltre il taglio", []string{"x", "y", "d2"}, 2, 0.0},
		{"nessun rilevante", []string{"x", "y"}, 10, 0.0},
	}

	for _, c := range casi {
		t.Run(c.nome, func(t *testing.T) {
			vicino(t, MRR(c.ranked, rel, c.k), c.atteso)
		})
	}
}

func TestCountRelevant(t *testing.T) {
	if n := CountRelevant(relevance{"a": 2, "b": 1, "c": 0}); n != 2 {
		t.Fatalf("attesi 2 rilevanti, ottenuti %d", n)
	}
	if n := CountRelevant(relevance{}); n != 0 {
		t.Fatalf("attesi 0 rilevanti, ottenuti %d", n)
	}
}

// topGrades replaces a sort with bucket counting: it has to give exactly what
// a descending sort truncated to k would give.
func TestTopGradesMatchesASortedTruncation(t *testing.T) {
	rel := relevance{"a": 1, "b": 3, "c": 2, "d": 3, "e": 0, "f": 1}

	got := topGrades(rel, 4)
	atteso := []int{3, 3, 2, 1}
	if len(got) != len(atteso) {
		t.Fatalf("attesi %v, ottenuti %v", atteso, got)
	}
	for i := range got {
		if got[i] != atteso[i] {
			t.Fatalf("attesi %v, ottenuti %v", atteso, got)
		}
	}

	if g := topGrades(rel, 100); len(g) != 5 {
		t.Fatalf("con k oltre il numero di rilevanti attesi 5 gradi, ottenuti %v", g)
	}
	if g := topGrades(relevance{"a": 0}, 5); g != nil {
		t.Fatalf("senza rilevanti atteso nil, ottenuto %v", g)
	}
}
