package engine

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Token represents a single parsed term from a document
type Token struct {
	Term     string
	Position int
	Field    string
}

// removeAccents strips diacritical marks from letters (e.g., è -> e)
func removeAccents(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, _ := transform.String(t, s)
	return result
}

// Tokenize processes a string, removes stops, lowercases, and splits by non-letter/number characters
//
// It takes the whole Settings rather than the stop words alone so that a new
// stage - today the stemmer - cannot reach some call sites and miss others.
// There are seven of them, across indexing, query parsing and search, and if
// the index were built with a stage the query does not apply, the index would
// hold terms the query can no longer produce. The symptom would be "it finds
// nothing", with no error anywhere, which is the hardest kind to trace back.
// With Settings in the signature, a call site left behind does not compile.
func Tokenize(text string, field string, settings Settings) []Token {
	stopwords := settings.StopWords
	// L'ordine e' quello di EnglishAnalyzer di Lucene: prima si tolgono le
	// stopword, poi si stemma. Invertirlo cambierebbe i risultati, perche' la
	// lista di stopword e' fatta di parole intere e non di radici.
	norm := analizzatore(settings.Stemmer)

	// 1. Single pass normalization (lowercase + remove accents)
	normalized := normalizzaNumeri(removeAccents(strings.ToLower(text)), settings)

	var tokens []Token
	var currentTerm strings.Builder
	position := 0

	addToken := func() {
		if currentTerm.Len() > 0 {
			term := currentTerm.String()
			if len(settings.ElisionArticles) > 0 {
				term = togliElisione(term, settings.ElisionArticles)
			}
			if stopwords == nil || !stopwords[term] {
				if norm != nil {
					term = norm.Normalize(term)
				}
				tokens = append(tokens, Token{
					Term:     term,
					Position: position,
					Field:    field,
				})
			}
			currentTerm.Reset()
			position++
		}
	}

	standard := settings.Tokenizer == TokenizerStandard
	rs := []rune(normalized)
	for i, r := range rs {
		if unicode.IsLetter(r) || unicode.IsNumber(r) || (standard && dentroLaParola(rs, i)) {
			currentTerm.WriteRune(r)
		} else {
			addToken()
		}
	}
	addToken() // flush remaining

	return tokens
}

// dentroLaParola says whether the punctuation at rs[i] stays inside the word
// under TokenizerStandard, following the UAX#29 rules Elasticsearch applies:
// WB6-7 between letters, WB11-12 between digits, WB13a-b for the underscore.
func dentroLaParola(rs []rune, i int) bool {
	parola := func(j int) bool {
		return j >= 0 && j < len(rs) && (unicode.IsLetter(rs[j]) || unicode.IsNumber(rs[j]) || rs[j] == '_')
	}
	r := rs[i]
	if r == '_' {
		return parola(i-1) || parola(i+1)
	}
	if i == 0 || i == len(rs)-1 {
		return false
	}
	prima, dopo := rs[i-1], rs[i+1]
	switch r {
	case '\'', '\u2018', '\u2019', '.':
		return (unicode.IsLetter(prima) && unicode.IsLetter(dopo)) || (unicode.IsNumber(prima) && unicode.IsNumber(dopo))
	case ':', '\u00b7':
		return unicode.IsLetter(prima) && unicode.IsLetter(dopo)
	case ',', ';':
		return unicode.IsNumber(prima) && unicode.IsNumber(dopo)
	}
	return false
}

var mesi = []string{"gennaio", "febbraio", "marzo", "aprile", "maggio", "giugno", "luglio", "agosto",
	"settembre", "ottobre", "novembre", "dicembre"}

// A date or an amount is recognised only as a whole: the character before it
// and the one after it must not continue it (Go's regexp has no lookbehind, so
// the boundaries are checked by hand, in confine).
var (
	dataBarra    = regexp.MustCompile(`(\d{1,2})/(\d{1,2})/(\d{4}|\d{2})`)
	dataPunto    = regexp.MustCompile(`(\d{1,2})\.(\d{1,2})\.(\d{4})`)
	dataTrattino = regexp.MustCompile(`(\d{1,2})-(\d{1,2})-(\d{4})`)
	dataMese     = regexp.MustCompile(`(?i)(\d{1,2})\s+(` + strings.Join(mesi, "|") + `)\s+(\d{4})`)
	importo      = regexp.MustCompile(`\d{1,3}(?:\.\d{3})+(?:,\d+)?`)
)

// normalizzaNumeri rewrites dates and amounts into one form each, for
// Settings.NormalizeDates and Settings.NormalizeAmounts, before the
// tokenizer: the same text for documents and queries, so that a date written
// 14/01/2026, 14.01.2026, 14-01-2026 or "14 gennaio 2026" is the one term
// 20260114, and 1.234,56 is 1234,56. A date that cannot be one (month 13, day
// 32, year outside 1900-2099) is left as it is.
func normalizzaNumeri(s string, settings Settings) string {
	// Plain string checks first: most texts cannot hold a given format, and a
	// regexp pass over every field of every document costs about 40% of the
	// indexing time.
	if !(settings.NormalizeDates || settings.NormalizeAmounts) || !strings.ContainsAny(s, "0123456789") {
		return s
	}
	if settings.NormalizeDates {
		if strings.Contains(s, "/") {
			s = sostituisci(s, dataBarra, "./,-", "/", "", comeData)
		}
		if strings.Contains(s, ".") {
			s = sostituisci(s, dataPunto, "./,-", "", ".,/", comeData)
		}
		if strings.Contains(s, "-") {
			s = sostituisci(s, dataTrattino, "./,-", "-", "", comeData)
		}
		if contieneMese(s) {
			s = sostituisci(s, dataMese, "", "", "", comeData)
		}
	}
	if settings.NormalizeAmounts && strings.Contains(s, ".") {
		s = sostituisci(s, importo, ".,", "", ".,", func(g []string) (string, bool) {
			return strings.ReplaceAll(g[0], ".", ""), true
		})
	}
	return s
}

// contieneMese says whether s holds an Italian month name, in any case.
func contieneMese(s string) bool {
	basso := strings.ToLower(s)
	for _, m := range mesi {
		if strings.Contains(basso, m) {
			return true
		}
	}
	return false
}

func comeData(g []string) (string, bool) {
	giorno, _ := strconv.Atoi(g[1])
	mese, err := strconv.Atoi(g[2])
	if err != nil {
		mese = 0
		for i, m := range mesi {
			if strings.EqualFold(m, g[2]) {
				mese = i + 1
			}
		}
	}
	anno, _ := strconv.Atoi(g[3])
	if len(g[3]) == 2 {
		anno += 2000
	}
	if giorno < 1 || giorno > 31 || mese < 1 || mese > 12 || anno < 1900 || anno > 2099 {
		return "", false
	}
	return fmt.Sprintf("%04d%02d%02d", anno, mese, giorno), true
}

// sostituisci replaces each match of re that stands alone: not preceded by a
// digit or by one of prima, not followed by a digit or by one of dopo, nor by
// one of dopoCifra followed by a digit.
func sostituisci(s string, re *regexp.Regexp, prima, dopo, dopoCifra string, nuovo func([]string) (string, bool)) string {
	trovati := re.FindAllStringSubmatchIndex(s, -1)
	if trovati == nil {
		return s
	}
	var out strings.Builder
	ultimo := 0
	for _, m := range trovati {
		if !confine(s, m[0], m[1], prima, dopo, dopoCifra) {
			continue
		}
		g := make([]string, len(m)/2)
		for i := range g {
			if m[2*i] >= 0 {
				g[i] = s[m[2*i]:m[2*i+1]]
			}
		}
		v, ok := nuovo(g)
		if !ok {
			continue
		}
		out.WriteString(s[ultimo:m[0]])
		out.WriteString(v)
		ultimo = m[1]
	}
	out.WriteString(s[ultimo:])
	return out.String()
}

func confine(s string, inizio, fine int, prima, dopo, dopoCifra string) bool {
	if inizio > 0 {
		r, _ := utf8.DecodeLastRuneInString(s[:inizio])
		if unicode.IsDigit(r) || strings.ContainsRune(prima, r) {
			return false
		}
	}
	if fine < len(s) {
		r, n := utf8.DecodeRuneInString(s[fine:])
		if unicode.IsDigit(r) || strings.ContainsRune(dopo, r) {
			return false
		}
		if strings.ContainsRune(dopoCifra, r) && fine+n < len(s) {
			if r2, _ := utf8.DecodeRuneInString(s[fine+n:]); unicode.IsDigit(r2) {
				return false
			}
		}
	}
	return true
}
