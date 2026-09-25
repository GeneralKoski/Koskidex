package engine

import (
	"math"
	"testing"
)

func punteggiIbridi(t *testing.T, s Settings, query string, q []float64) map[string]float64 {
	t.Helper()
	ms, _ := indiceIbrido(s).SearchScored(query, s, "0", q)
	out := map[string]float64{}
	for _, m := range ms {
		out[m.DocID] = m.Score
	}
	return out
}

func circa(t *testing.T, got, want float64, cosa string) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("%s: atteso %v, ottenuto %v", cosa, want, got)
	}
}

func f(x float64) *float64 { return &x }

// La somma di oggi con la costante esplicita: zero toglie il vettore, il
// doppio raddoppia il suo contributo, e senza costante resta 20.
func TestVectorWeightScalesTheSum(t *testing.T) {
	q := []float64{0.0, 1.0}
	base := impostazioniIbride(HybridUnion, 3)
	lessicale := punteggiIbridi(t, impostazioniIbride("", 3), "contratto", nil)["lessicale"]

	zero := base
	zero.VectorWeight = f(0)
	p := punteggiIbridi(t, zero, "contratto", q)
	circa(t, p["lessicale"], lessicale, "con peso zero il lessicale non cambia")
	circa(t, p["lontano"], 0, "con peso zero un documento del solo vettore vale zero")

	doppio := base
	doppio.VectorWeight = f(40)
	circa(t, punteggiIbridi(t, doppio, "contratto", q)["lontano"], 40, "peso 40, similarità 1")
	circa(t, punteggiIbridi(t, base, "contratto", q)["lontano"], 20, "senza peso resta 20")
}

// "contratto": lessicale trova solo "lessicale". Vettore della query (0.6, 0.8):
// similarità lontano 0.8, semantico 0.748..., lessicale 0.6.
func TestRRFFusesTheTwoRanks(t *testing.T) {
	s := impostazioniIbride("", 3)
	s.FusionMode = FusionRRF
	p := punteggiIbridi(t, s, "contratto", []float64{0.6, 0.8})
	if len(p) != 3 {
		t.Fatalf("RRF lavora sull'unione dei candidati, anche senza HybridMode: %v", p)
	}
	circa(t, p["lessicale"], 1.0/61+1.0/63, "lessicale: primo nel lessicale, terzo nel vettore")
	circa(t, p["lontano"], 1.0/61, "lontano: solo primo nel vettore")
	circa(t, p["semantico"], 1.0/62, "semantico: solo secondo nel vettore")
}

func TestConvexNormalisesEachList(t *testing.T) {
	s := impostazioniIbride("", 3)
	s.FusionMode = FusionConvex
	q := []float64{0.6, 0.8}
	sims := map[string]float64{"lontano": 0.8, "semantico": cosineSimilarity(q, []float64{0.98, 0.2}), "lessicale": 0.6}
	norm := func(id string) float64 { return (sims[id] - 0.6) / (0.8 - 0.6) }

	p := punteggiIbridi(t, s, "contratto", q)
	// Un solo candidato lessicale: il massimo e il minimo coincidono, e vale 1.
	circa(t, p["lessicale"], 0.5*1+0.5*norm("lessicale"), "lessicale, α 0,5")
	circa(t, p["lontano"], 0.5*norm("lontano"), "lontano, α 0,5")
	circa(t, p["semantico"], 0.5*norm("semantico"), "semantico, α 0,5")

	s.FusionAlpha = f(1)
	p = punteggiIbridi(t, s, "contratto", q)
	circa(t, p["lessicale"], 1, "α 1: conta solo il lessicale")
	circa(t, p["lontano"], 0, "α 1: il solo vettore vale zero")

	s.FusionAlpha = f(0)
	p = punteggiIbridi(t, s, "contratto", q)
	circa(t, p["lontano"], 1, "α 0: conta solo il vettore")
	circa(t, p["lessicale"], 0, "α 0: il lessicale, ultimo nel vettore, vale zero")
}

// Senza vettore nella query le fusioni non toccano niente.
func TestFusionWithoutAQueryVectorIsLexical(t *testing.T) {
	for _, modo := range []string{FusionRRF, FusionConvex} {
		s := impostazioniIbride("", 3)
		s.FusionMode = modo
		base := punteggiIbridi(t, impostazioniIbride("", 3), "contratto", nil)
		p := punteggiIbridi(t, s, "contratto", nil)
		if len(p) != 1 || p["lessicale"] != base["lessicale"] {
			t.Fatalf("%s senza vettore: %v invece di %v", modo, p, base)
		}
	}
}
