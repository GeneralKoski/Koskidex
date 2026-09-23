package engine

// Analyzer normalises a single term, after tokenization and before it reaches
// the index.
//
// It is an interface and not a language flag on purpose. English is what the
// published references use, so it is what validates the implementation;
// Italian is what the documents this engine is aimed at are written in, and
// has no published reference to validate against. Treating the language as a
// parameter of the index, rather than a property of the engine, is what makes
// the second one addable without touching anything here.
type Analyzer interface {
	Normalize(term string) string
}

// Nomi degli analizzatori, come finiscono in Settings.Stemmer.
const (
	// StemmerNone e' il valore vuoto: nessuna normalizzazione, cioe' il
	// comportamento di sempre. Le settings sono persistite e un indice
	// salvato prima che questo campo esistesse lo rilegge cosi'.
	StemmerNone = ""
	// StemmerPorter e' l'algoritmo di Porter del 1980, quello che usa
	// EnglishAnalyzer di Lucene.
	StemmerPorter = "porter"
)

type porterAnalyzer struct{}

func (porterAnalyzer) Normalize(term string) string { return stemPorter(term) }

// analizzatore resolves the name in the settings.
//
// A name nobody recognises returns nil, which means no normalization, rather
// than an error: an index saved by a future version with an analyzer this one
// does not have must still open. It will rank differently, and that is
// visible; refusing to start would not be.
func analizzatore(nome string) Analyzer {
	if nome == StemmerPorter {
		return porterAnalyzer{}
	}

	return nil
}

// EnglishStopWords is Lucene's stop set, the same 33 words EnglishAnalyzer
// removes by default, and therefore the same ones the published BM25
// references never see.
//
// It is returned as a fresh map every time because it goes into Settings,
// which the caller is free to add to.
func EnglishStopWords() map[string]bool {
	parole := []string{
		"a", "an", "and", "are", "as", "at", "be", "but", "by",
		"for", "if", "in", "into", "is", "it",
		"no", "not", "of", "on", "or", "such",
		"that", "the", "their", "then", "there", "these",
		"they", "this", "to", "was", "will", "with",
	}

	out := make(map[string]bool, len(parole))
	for _, p := range parole {
		out[p] = true
	}

	return out
}
