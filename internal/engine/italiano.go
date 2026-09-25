package engine

import (
	"strings"
	"unicode/utf8"
)

// StemmerItalianLight is Lucene's ItalianLightStemmer, Savoy's light stemmer
// from the CLEF 2001 report: the one Elasticsearch's italian analyzer uses. It
// only folds accents and strips the final vowel of plurals and genders
// (-e, -i, -a, -o, -ie, -he, -hi, -ii, -ia, -io), and leaves words shorter than
// six characters alone.
const StemmerItalianLight = "italian_light"

type italianLightAnalyzer struct{}

func (italianLightAnalyzer) Normalize(term string) string { return stemItalianLight(term) }

func stemItalianLight(term string) string {
	s := []rune(term)
	n := len(s)
	if n < 6 {
		return term
	}
	for i, r := range s {
		switch r {
		case 'à', 'á', 'â', 'ä':
			s[i] = 'a'
		case 'ò', 'ó', 'ô', 'ö':
			s[i] = 'o'
		case 'è', 'é', 'ê', 'ë':
			s[i] = 'e'
		case 'ù', 'ú', 'û', 'ü':
			s[i] = 'u'
		case 'ì', 'í', 'î', 'ï':
			s[i] = 'i'
		}
	}
	switch s[n-1] {
	case 'e':
		if s[n-2] == 'i' || s[n-2] == 'h' {
			return string(s[:n-2])
		}
		return string(s[:n-1])
	case 'i':
		if s[n-2] == 'h' || s[n-2] == 'i' {
			return string(s[:n-2])
		}
		return string(s[:n-1])
	case 'a', 'o':
		if s[n-2] == 'i' {
			return string(s[:n-2])
		}
		return string(s[:n-1])
	}
	return string(s)
}

// ItalianStopWords is Snowball's Italian stop list, the one Lucene's
// ItalianAnalyzer and Elasticsearch's italian analyzer remove. Koskidex folds
// accents before it looks a term up, so the words are stored folded: "è" and
// "e" become one, and "sarà" also removes the name "Sara", where Lucene, which
// checks the accented form, keeps it.
func ItalianStopWords() map[string]bool {
	parole := strings.Fields(`ad al allo ai agli all agl alla alle con col coi da dal dallo dai dagli dall dagl dalla dalle
		di del dello dei degli dell degl della delle in nel nello nei negli nell negl nella nelle su sul sullo sui sugli
		sull sugl sulla sulle per tra contro io tu lui lei noi voi loro mio mia miei mie tuo tua tuoi tue suo sua suoi sue
		nostro nostra nostri nostre vostro vostra vostri vostre mi ti ci vi lo la li le gli ne il un uno una ma ed se
		perché anche come dov dove che chi cui non più quale quanto quanti quanta quante quello quelli quella quelle
		questo questi questa queste si tutto tutti a c e i l o ho hai ha abbiamo avete hanno abbia abbiate abbiano avrò
		avrai avrà avremo avrete avranno avrei avresti avrebbe avremmo avreste avrebbero avevo avevi aveva avevamo
		avevate avevano ebbi avesti ebbe avemmo aveste ebbero avessi avesse avessimo avessero avendo avuto avuta avuti
		avute sono sei è siamo siete sia siate siano sarò sarai sarà saremo sarete saranno sarei saresti sarebbe saremmo
		sareste sarebbero ero eri era eravamo eravate erano fui fosti fu fummo foste furono fossi fosse fossimo fossero
		essendo faccio fai facciamo fanno faccia facciate facciano farò farai farà faremo farete faranno farei faresti
		farebbe faremmo fareste farebbero facevo facevi faceva facevamo facevate facevano feci facesti fece facemmo
		faceste fecero facessi facesse facessimo facessero facendo sto stai sta stiamo stanno stia stiate stiano starò
		starai starà staremo starete staranno starei staresti starebbe staremmo stareste starebbero stavo stavi stava
		stavamo stavate stavano stetti stesti stette stemmo steste stettero stessi stesse stessimo stessero stando`)
	out := make(map[string]bool, len(parole))
	for _, p := range parole {
		out[removeAccents(p)] = true
	}
	return out
}

// ItalianElisionArticles are the articles Elasticsearch's italian analyzer
// strips before an apostrophe, the DEFAULT_ARTICLES of Lucene's
// ItalianAnalyzer: "dell'illuminazione" is indexed as "illuminazione".
// Returned fresh every time, because it goes into Settings.
func ItalianElisionArticles() []string {
	return []string{"c", "l", "all", "dall", "dell", "nell", "sull", "coll", "pell", "gl", "agl",
		"dagl", "degl", "negl", "sugl", "un", "m", "t", "s", "v", "d"}
}

// togliElisione drops the part of a term up to its first apostrophe (' or ’,
// as Lucene's ElisionFilter) when that part is one of the articles. Only the
// standard tokenizer keeps an apostrophe inside a term; the default one has
// already split there.
func togliElisione(term string, articoli []string) string {
	i := strings.IndexAny(term, "'’")
	if i < 0 {
		return term
	}
	for _, a := range articoli {
		if strings.EqualFold(term[:i], a) {
			_, n := utf8.DecodeRuneInString(term[i:])
			return term[i+n:]
		}
	}
	return term
}
