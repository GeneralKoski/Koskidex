package engine

import (
	"reflect"
	"testing"
)

// Un bigramma si registra una volta per termine, anche se il termine lo
// ripete ("abab"), sparisce dall'indice e poi torna.
func TestPrefissiSenzaDoppioni(t *testing.T) {
	idx := NewInvertedIndex()
	settings := DefaultSettings()
	settings.SearchableFields = []string{"title"}

	idx.AddDocument("1", map[string]interface{}{"title": "abab samsung"}, settings)
	idx.AddDocument("2", map[string]interface{}{"title": "abab"}, settings)
	idx.DeleteDocument("1")
	idx.DeleteDocument("2")
	idx.AddDocument("3", map[string]interface{}{"title": "samsung abab"}, settings)

	for bigramma, termini := range idx.prefixMap {
		visti := map[string]bool{}
		for _, termine := range termini {
			if visti[termine] {
				t.Fatalf("%q compare due volte sotto %q: %v", termine, bigramma, termini)
			}
			visti[termine] = true
		}
	}
	if got := idx.prefixMap["ab"]; !reflect.DeepEqual(got, []string{"abab"}) {
		t.Fatalf("prefixMap[ab] = %v, voglio [abab]", got)
	}
	if res, _ := idx.Search("samsong", settings, "AUTO", nil); len(res) != 1 || res[0] != "3" {
		t.Fatalf("la ricerca con un refuso deve trovare il documento 3, trovato %v", res)
	}
}

// Un blocco con documenti già presenti, e un id ripetuto nel blocco, deve
// lasciare l'indice com'è dopo le stesse aggiunte fatte una per una.
func TestAddDocumentsComeUnoAUno(t *testing.T) {
	settings := Settings{SearchableFields: []string{"title"}}
	iniziali := []string{"red phone", "blue phone", "red watch", "dark camera"}
	ids := []string{"2", "9", "1", "2", "4"}
	titoli := []string{"neon phone", "solar phone", "red phone red", "quantum mouse", "dark camera"}

	uno, blocco := NewInvertedIndex(), NewInvertedIndex()
	for i, titolo := range iniziali {
		id := string(rune('1' + i))
		uno.AddDocument(id, map[string]interface{}{"title": titolo}, settings)
		blocco.AddDocument(id, map[string]interface{}{"title": titolo}, settings)
	}

	docs := make([]map[string]interface{}, len(ids))
	for i := range ids {
		uno.AddDocument(ids[i], map[string]interface{}{"title": titoli[i]}, settings)
		docs[i] = map[string]interface{}{"title": titoli[i]}
	}
	blocco.AddDocuments(ids, docs, settings)

	confronta(t, uno, blocco)
}

func TestDeleteDocumentsComeUnoAUno(t *testing.T) {
	settings := Settings{SearchableFields: []string{"title"}}
	uno, blocco := NewInvertedIndex(), NewInvertedIndex()
	for i, titolo := range []string{"red phone", "blue phone", "red watch", "dark camera"} {
		id := string(rune('1' + i))
		uno.AddDocument(id, map[string]interface{}{"title": titolo}, settings)
		blocco.AddDocument(id, map[string]interface{}{"title": titolo}, settings)
	}

	via := []string{"3", "1", "3", "7"}
	for _, id := range via {
		uno.DeleteDocument(id)
	}
	blocco.DeleteDocuments(via)

	confronta(t, uno, blocco)
}

func confronta(t *testing.T, a, b *InvertedIndex) {
	t.Helper()
	if !reflect.DeepEqual(a.index, b.index) {
		t.Errorf("posting diverse:\n%v\n%v", a.index, b.index)
	}
	if !reflect.DeepEqual(a.docs, b.docs) {
		t.Errorf("documenti diversi:\n%v\n%v", a.docs, b.docs)
	}
	if !reflect.DeepEqual(a.docFreq, b.docFreq) {
		t.Errorf("docFreq diverse:\n%v\n%v", a.docFreq, b.docFreq)
	}
	if !reflect.DeepEqual(a.docLengths, b.docLengths) || a.totalLen != b.totalLen {
		t.Errorf("lunghezze diverse: %v %d, %v %d", a.docLengths, a.totalLen, b.docLengths, b.totalLen)
	}
}
