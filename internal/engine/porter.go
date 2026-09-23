package engine

// Porter's stemming algorithm, 1980.
//
// It reduces inflected forms to a common stem so that "manutenzione" and
// "manutenzioni" - or, in the collections the references are published on,
// "retrieving" and "retrieval" - meet in the index instead of being two
// unrelated terms.
//
// The stem is not a word and is not meant to be shown to anybody: "relational"
// and "relate" both become "relat". That is the point. What matters is only
// that the same rule runs over the documents and over the query.
//
// This is the original algorithm, not Porter2/Snowball, because the published
// BM25 references this project measures itself against run Lucene's
// EnglishAnalyzer, which uses it. Correctness is checked against Porter's own
// test vocabulary, 23531 words, in porter_test.go.
type porter struct {
	b []byte
	k int // indice dell'ultimo carattere
	j int // offset di lavoro, lo posiziona ends()
}

// cons says whether the letter at i is a consonant. 'y' is the awkward one: it
// is a consonant at the start of a word and after a vowel, a vowel otherwise.
func (p *porter) cons(i int) bool {
	switch p.b[i] {
	case 'a', 'e', 'i', 'o', 'u':
		return false
	case 'y':
		if i == 0 {
			return true
		}

		return !p.cons(i - 1)
	}

	return true
}

// m counts the consonant-vowel sequences between the start of the word and j.
// It is Porter's measure of "how long the stem already is", and it is what
// most rules are conditioned on: stripping a suffix off a short word usually
// destroys it.
func (p *porter) m() int {
	n, i := 0, 0
	for {
		if i > p.j {
			return n
		}
		if !p.cons(i) {
			break
		}
		i++
	}
	i++
	for {
		for {
			if i > p.j {
				return n
			}
			if p.cons(i) {
				break
			}
			i++
		}
		i++
		n++
		for {
			if i > p.j {
				return n
			}
			if !p.cons(i) {
				break
			}
			i++
		}
		i++
	}
}

func (p *porter) vowelInStem() bool {
	for i := 0; i <= p.j; i++ {
		if !p.cons(i) {
			return true
		}
	}

	return false
}

// doublec: the letter at i is a consonant doubled, as in "hopp" or "fall".
func (p *porter) doublec(i int) bool {
	if i < 1 || p.b[i] != p.b[i-1] {
		return false
	}

	return p.cons(i)
}

// cvc: consonant-vowel-consonant ending where the last consonant is not w, x
// or y. It marks short words like "hop" or "fil" where a final 'e' has to be
// restored after stripping, so "hoping" does not become "hop".
func (p *porter) cvc(i int) bool {
	if i < 2 || !p.cons(i) || p.cons(i-1) || !p.cons(i-2) {
		return false
	}
	switch p.b[i] {
	case 'w', 'x', 'y':
		return false
	}

	return true
}

// ends reports whether the word ends in s, and on success leaves j just before
// the suffix so the rules can measure the stem.
func (p *porter) ends(s string) bool {
	l := len(s)
	o := p.k - l + 1
	if o < 0 || string(p.b[o:o+l]) != s {
		return false
	}
	p.j = p.k - l

	return true
}

// setto replaces everything after j with s. The buffer can have to grow: "at"
// becomes "ate", "bl" becomes "ble".
func (p *porter) setto(s string) {
	for len(p.b) < p.j+1+len(s) {
		p.b = append(p.b, 0)
	}
	copy(p.b[p.j+1:], s)
	p.k = p.j + len(s)
}

// sostituisci applies setto only if what is left is long enough to survive it.
func (p *porter) sostituisci(s string) {
	if p.m() > 0 {
		p.setto(s)
	}
}

// step1ab removes plurals and -ed/-ing.
func (p *porter) step1ab() {
	if p.b[p.k] == 's' {
		switch {
		case p.ends("sses"):
			p.k -= 2
		case p.ends("ies"):
			p.setto("i")
		case p.b[p.k-1] != 's':
			p.k--
		}
	}

	switch {
	case p.ends("eed"):
		if p.m() > 0 {
			p.k--
		}
	case (p.ends("ed") || p.ends("ing")) && p.vowelInStem():
		p.k = p.j
		switch {
		case p.ends("at"):
			p.setto("ate")
		case p.ends("bl"):
			p.setto("ble")
		case p.ends("iz"):
			p.setto("ize")
		case p.doublec(p.k):
			p.k--
			switch p.b[p.k] {
			case 'l', 's', 'z':
				p.k++
			}
		case p.m() == 1 && p.cvc(p.k):
			p.setto("e")
		}
	}
}

// step1c turns a terminal y into i, so "happy" and "happiness" meet.
func (p *porter) step1c() {
	if p.ends("y") && p.vowelInStem() {
		p.b[p.k] = 'i'
	}
}

// step2 maps double suffixes to single ones: -ization becomes -ize.
func (p *porter) step2() {
	if p.k == 0 {
		return
	}
	switch p.b[p.k-1] {
	case 'a':
		if p.ends("ational") {
			p.sostituisci("ate")
		} else if p.ends("tional") {
			p.sostituisci("tion")
		}
	case 'c':
		if p.ends("enci") {
			p.sostituisci("ence")
		} else if p.ends("anci") {
			p.sostituisci("ance")
		}
	case 'e':
		if p.ends("izer") {
			p.sostituisci("ize")
		}
	case 'g':
		// "logi" -> "log": e' una delle due deviazioni dall'algoritmo
		// pubblicato che Porter ha poi messo nel proprio codice di
		// riferimento, ed e' quella che il vocabolario di prova si aspetta
		// ("apology" deve dare "apolog", non "apologi").
		if p.ends("logi") {
			p.sostituisci("log")
		}
	case 'l':
		// Seconda deviazione: l'articolo pubblicato dice "abli" -> "able",
		// il codice di riferimento dice "bli" -> "ble", ed e' questa che
		// riproduce l'output atteso ("assembly" -> "assembl").
		if p.ends("bli") {
			p.sostituisci("ble")
		} else if p.ends("alli") {
			p.sostituisci("al")
		} else if p.ends("entli") {
			p.sostituisci("ent")
		} else if p.ends("eli") {
			p.sostituisci("e")
		} else if p.ends("ousli") {
			p.sostituisci("ous")
		}
	case 'o':
		if p.ends("ization") {
			p.sostituisci("ize")
		} else if p.ends("ation") {
			p.sostituisci("ate")
		} else if p.ends("ator") {
			p.sostituisci("ate")
		}
	case 's':
		if p.ends("alism") {
			p.sostituisci("al")
		} else if p.ends("iveness") {
			p.sostituisci("ive")
		} else if p.ends("fulness") {
			p.sostituisci("ful")
		} else if p.ends("ousness") {
			p.sostituisci("ous")
		}
	case 't':
		if p.ends("aliti") {
			p.sostituisci("al")
		} else if p.ends("iviti") {
			p.sostituisci("ive")
		} else if p.ends("biliti") {
			p.sostituisci("ble")
		}
	}
}

// step3 deals with -ic-, -full, -ness.
func (p *porter) step3() {
	switch p.b[p.k] {
	case 'e':
		if p.ends("icate") {
			p.sostituisci("ic")
		} else if p.ends("ative") {
			p.sostituisci("")
		} else if p.ends("alize") {
			p.sostituisci("al")
		}
	case 'i':
		if p.ends("iciti") {
			p.sostituisci("ic")
		}
	case 'l':
		if p.ends("ical") {
			p.sostituisci("ic")
		} else if p.ends("ful") {
			p.sostituisci("")
		}
	case 's':
		if p.ends("ness") {
			p.sostituisci("")
		}
	}
}

// step4 strips -ant, -ence and the rest, but only off a stem long enough to
// take it.
func (p *porter) step4() {
	if p.k == 0 {
		return
	}

	var trovato bool
	switch p.b[p.k-1] {
	case 'a':
		trovato = p.ends("al")
	case 'c':
		trovato = p.ends("ance") || p.ends("ence")
	case 'e':
		trovato = p.ends("er")
	case 'i':
		trovato = p.ends("ic")
	case 'l':
		trovato = p.ends("able") || p.ends("ible")
	case 'n':
		trovato = p.ends("ant") || p.ends("ement") || p.ends("ment") || p.ends("ent")
	case 'o':
		if p.ends("ion") && p.j >= 0 && (p.b[p.j] == 's' || p.b[p.j] == 't') {
			trovato = true
		} else {
			trovato = p.ends("ou")
		}
	case 's':
		trovato = p.ends("ism")
	case 't':
		trovato = p.ends("ate") || p.ends("iti")
	case 'u':
		trovato = p.ends("ous")
	case 'v':
		trovato = p.ends("ive")
	case 'z':
		trovato = p.ends("ize")
	}

	if trovato && p.m() > 1 {
		p.k = p.j
	}
}

// step5 removes a final e, and undoubles a final ll.
func (p *porter) step5() {
	p.j = p.k
	if p.b[p.k] == 'e' {
		a := p.m()
		if a > 1 || (a == 1 && !p.cvc(p.k-1)) {
			p.k--
		}
	}
	if p.b[p.k] == 'l' && p.doublec(p.k) && p.m() > 1 {
		p.k--
	}
}

// stemPorter reduces a lowercase word to its stem.
//
// Words of one or two letters come back untouched: there is nothing to strip
// that would not destroy them.
func stemPorter(parola string) string {
	if len(parola) <= 2 {
		return parola
	}

	p := &porter{b: []byte(parola), k: len(parola) - 1}
	p.step1ab()
	if p.k > 0 {
		p.step1c()
		p.step2()
		p.step3()
		p.step4()
		p.step5()
	}

	return string(p.b[:p.k+1])
}
