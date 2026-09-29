package engine

import (
	"math/rand"
	"strings"
	"sync"
	"testing"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// removeAccentsComePrima is removeAccents before the pool and the ASCII
// shortcut: a new chain for every call.
func removeAccentsComePrima(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, _ := transform.String(t, s)
	return result
}

func casiAccenti() []string {
	casi := []string{
		"", "a", "determina 1223 crispiano", "Perché è così", "città più già",
		"ÀÈÌÒÙ àèìòù áéíóú âêîôû äëïöü ãõñ ç", "é à ỗ",
		"ǅ ǆ ﬁ Å Å ﬀ", "ṩ ḍ̇ q̣̇", "Ελληνικά ά έ", "Русский й ё", "日本語のテキスト",
		"emoji 😀 e ☃", "\xff\xfe invalido \xc3", "\xc3\xa8\xe0", strings.Repeat("àèìòù ", 2000),
		"Delibera n. 12/2026 dell'1.1.2026: € 1.234,56", "tab\tnuova\nriga\r",
	}
	rng := rand.New(rand.NewSource(20260929))
	alfabeto := []rune("aeiouAEIOU àèéìòùÀÈÉÌÒÙçñäöüß̀́̂̈ 0123456789.,/'-ΑαЖж日😀�")
	for i := 0; i < 5000; i++ {
		var b strings.Builder
		for j := rng.Intn(60); j > 0; j-- {
			b.WriteRune(alfabeto[rng.Intn(len(alfabeto))])
		}
		s := b.String()
		if i%7 == 0 && len(s) > 2 {
			k := rng.Intn(len(s))
			s = s[:k] + string([]byte{byte(0x80 + rng.Intn(0x80))}) + s[k:]
		}
		casi = append(casi, s)
	}
	return casi
}

func TestRemoveAccentsComePrima(t *testing.T) {
	for _, s := range casiAccenti() {
		if got, want := removeAccents(s), removeAccentsComePrima(s); got != want {
			t.Fatalf("removeAccents(%q) = %q, prima %q", s, got, want)
		}
	}
}

// The pool hands each goroutine a chain of its own: under -race, a shared
// one would be reported, and without -race it would mix up outputs.
func TestRemoveAccentsDaPiuGoroutine(t *testing.T) {
	casi := casiAccenti()
	attesi := make([]string, len(casi))
	for i, s := range casi {
		attesi[i] = removeAccentsComePrima(s)
	}
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := g; i < len(casi); i += 3 {
				if got := removeAccents(casi[i]); got != attesi[i] {
					t.Errorf("goroutine %d: removeAccents(%q) = %q, attesi %q", g, casi[i], got, attesi[i])
					return
				}
			}
		}(g)
	}
	wg.Wait()
}

func TestRemoveAccentsNonAllocaSuASCII(t *testing.T) {
	s := "determina 1223 crispiano"
	if n := testing.AllocsPerRun(100, func() { removeAccents(s) }); n != 0 {
		t.Fatalf("su un testo ASCII removeAccents alloca %.0f volte, attese 0", n)
	}
}
