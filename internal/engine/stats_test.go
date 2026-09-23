package engine

import "testing"

func indiceStatistiche() (*InvertedIndex, Settings) {
	idx := NewInvertedIndex()
	s := DefaultSettings()
	s.SearchableFields = []string{"testo"}

	// "gatto" sta in due documenti su tre, "cane" in uno solo.
	idx.AddDocument("a", map[string]interface{}{"testo": "gatto gatto nero"}, s)
	idx.AddDocument("b", map[string]interface{}{"testo": "gatto bianco"}, s)
	idx.AddDocument("c", map[string]interface{}{"testo": "cane"}, s)

	return idx, s
}

func TestDocFrequencyCountsDocumentsNotOccurrences(t *testing.T) {
	idx, _ := indiceStatistiche()

	// "gatto" compare due volte in "a", ma df conta i documenti.
	if df := idx.DocFrequency("gatto"); df != 2 {
		t.Fatalf("df(gatto): attesi 2 documenti, ottenuti %d", df)
	}
	if df := idx.DocFrequency("cane"); df != 1 {
		t.Fatalf("df(cane): atteso 1, ottenuto %d", df)
	}
	if df := idx.DocFrequency("inesistente"); df != 0 {
		t.Fatalf("df di un termine assente deve essere 0, e' %d", df)
	}
}

func TestDocLengthCountsOccurrences(t *testing.T) {
	idx, _ := indiceStatistiche()

	if l := idx.DocLength("a"); l != 3 {
		t.Fatalf("|a| = 'gatto gatto nero' fa 3 token, ottenuto %d", l)
	}
	if l := idx.DocLength("c"); l != 1 {
		t.Fatalf("|c| = 'cane' fa 1 token, ottenuto %d", l)
	}
}

func TestAverageDocLength(t *testing.T) {
	idx, _ := indiceStatistiche()

	// (3 + 2 + 1) / 3
	if avg := idx.AverageDocLength(); avg != 2.0 {
		t.Fatalf("lunghezza media attesa 2.0, ottenuta %v", avg)
	}
	if avg := NewInvertedIndex().AverageDocLength(); avg != 0 {
		t.Fatalf("su un indice vuoto attesa 0, ottenuta %v", avg)
	}
}

// Hand-maintained counters rot on deletion. This is where they rot.
func TestStatisticsSurviveDeletion(t *testing.T) {
	idx, _ := indiceStatistiche()

	idx.DeleteDocument("a")

	if df := idx.DocFrequency("gatto"); df != 1 {
		t.Fatalf("df(gatto) dopo la cancellazione di 'a': atteso 1, ottenuto %d", df)
	}
	if df := idx.DocFrequency("nero"); df != 0 {
		t.Fatalf("'nero' stava solo in 'a', df atteso 0, ottenuto %d", df)
	}
	if l := idx.DocLength("a"); l != 0 {
		t.Fatalf("la lunghezza di un documento cancellato deve essere 0, e' %d", l)
	}
	// (2 + 1) / 2
	if avg := idx.AverageDocLength(); avg != 1.5 {
		t.Fatalf("media dopo la cancellazione attesa 1.5, ottenuta %v", avg)
	}
}

// Re-adding the same id is an update: addDocumentLocked deletes first, so
// nothing may be counted twice.
func TestStatisticsDoNotDoubleCountOnUpdate(t *testing.T) {
	idx, s := indiceStatistiche()

	idx.AddDocument("a", map[string]interface{}{"testo": "gatto gatto nero"}, s)

	if df := idx.DocFrequency("gatto"); df != 2 {
		t.Fatalf("df(gatto) dopo un reinserimento identico: atteso 2, ottenuto %d", df)
	}
	if l := idx.DocLength("a"); l != 3 {
		t.Fatalf("|a| dopo il reinserimento: atteso 3, ottenuto %d", l)
	}
	if avg := idx.AverageDocLength(); avg != 2.0 {
		t.Fatalf("media dopo il reinserimento: attesa 2.0, ottenuta %v", avg)
	}
}

// An update that shortens a document has to shorten the statistics too.
func TestStatisticsFollowAShrinkingUpdate(t *testing.T) {
	idx, s := indiceStatistiche()

	idx.AddDocument("a", map[string]interface{}{"testo": "gatto"}, s)

	if l := idx.DocLength("a"); l != 1 {
		t.Fatalf("|a| accorciato: atteso 1, ottenuto %d", l)
	}
	if df := idx.DocFrequency("nero"); df != 0 {
		t.Fatalf("'nero' non c'e' piu' in nessun documento, df atteso 0, ottenuto %d", df)
	}
	// (1 + 2 + 1) / 3
	if avg := idx.AverageDocLength(); avg != 4.0/3.0 {
		t.Fatalf("media attesa %v, ottenuta %v", 4.0/3.0, idx.AverageDocLength())
	}
}

// Reindex rebuilds every map from scratch: the statistics must be rebuilt with
// them, not doubled or left behind.
func TestStatisticsAreRebuiltByReindex(t *testing.T) {
	idx, s := indiceStatistiche()

	primaDF := idx.DocFrequency("gatto")
	primaAvg := idx.AverageDocLength()

	idx.Reindex(s)

	if df := idx.DocFrequency("gatto"); df != primaDF {
		t.Fatalf("df(gatto) dopo Reindex: atteso %d, ottenuto %d", primaDF, df)
	}
	if avg := idx.AverageDocLength(); avg != primaAvg {
		t.Fatalf("media dopo Reindex: attesa %v, ottenuta %v", primaAvg, avg)
	}
}
