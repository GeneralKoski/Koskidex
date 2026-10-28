package tests

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"testing"

	"github.com/GeneralKoski/Koskidex/internal/engine"
	"github.com/GeneralKoski/Koskidex/internal/manager"
	"github.com/GeneralKoski/Koskidex/internal/storage"
)

// L'indice dei prodotti e delle zone di Korsia, configurato via HTTP come lo
// configurerà l'app.
func TestSynonymGroupsAreASettingOverHTTP(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	richiesta(t, srv, "POST", "/indexes", `{"name":"korsia"}`)
	richiesta(t, srv, "POST", "/indexes/korsia/documents", `[
		{"id": "1", "name": "Coca-Cola", "category_label": "Bevande"},
		{"id": "2", "name": "Soft drink al limone", "category_label": "Bibite"}]`)

	w := richiesta(t, srv, "PUT", "/indexes/korsia/settings",
		`{"searchable_fields": ["name", "category_label"], "synonym_groups": [["bibite", "bevande", "soft drink"]]}`)
	if w.Code != http.StatusOK {
		t.Fatalf("aggiornamento impostazioni fallito: %d %s", w.Code, w.Body.String())
	}
	for _, q := range []string{"bibite", "bevande", "soft%20drink"} {
		if n := len(searchV2(t, srv, "/indexes/korsia/search?q="+q)["hits"].([]interface{})); n != 2 {
			t.Errorf("%s: %d risultati, ne servono 2", q, n)
		}
	}
}

func TestSynonymFieldsAreASettingOverHTTP(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	richiesta(t, srv, "POST", "/indexes", `{"name":"korsia"}`)
	richiesta(t, srv, "POST", "/indexes/korsia/documents", `[
		{"id": "1", "name": "Lambrusco", "category_label": "Vini", "zone_names": "Bevande e vini"},
		{"id": "2", "name": "Coca-Cola", "category_label": "Bevande", "zone_names": "Bevande e vini"}]`)
	w := richiesta(t, srv, "PUT", "/indexes/korsia/settings", `{
		"searchable_fields": ["name", "category_label", "zone_names"],
		"synonym_groups": [["bibite", "bevande"]],
		"synonym_fields": ["name", "category_label"]}`)
	if w.Code != http.StatusOK {
		t.Fatalf("aggiornamento impostazioni fallito: %d %s", w.Code, w.Body.String())
	}
	hits := searchV2(t, srv, "/indexes/korsia/search?q=bibite")["hits"].([]interface{})
	if len(hits) != 1 || hits[0].(map[string]interface{})["id"] != "2" {
		t.Fatalf("\"bibite\" deve trovare solo la Coca-Cola, non il Lambrusco: %v", hits)
	}
}

func TestBoostsAreASettingOverHTTP(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	richiesta(t, srv, "POST", "/indexes", `{"name":"korsia"}`)
	richiesta(t, srv, "POST", "/indexes/korsia/documents", `[
		{"id": "1", "kind": "product", "name": "Pepe nero"},
		{"id": "2", "kind": "zone", "name": "Pepe"}]`)
	for _, b := range []string{`{"filter": "kind=zone"}`, `{"filter": "kind=zone", "factor": 1e308}`, `{"filter": "garbage, kind=zone", "factor": 2}`} {
		if w := richiesta(t, srv, "PUT", "/indexes/korsia/settings", `{"boosts": [`+b+`]}`); w.Code != http.StatusBadRequest {
			t.Fatalf("il boost %s va rifiutato: %d", b, w.Code)
		}
	}
	if w := richiesta(t, srv, "PUT", "/indexes/korsia/settings", `{"searchable_fields": ["name"], "boosts": [{"filter": "kind=product", "factor": 2}]}`); w.Code != http.StatusOK {
		t.Fatalf("aggiornamento impostazioni fallito: %d %s", w.Code, w.Body.String())
	}
	hits := searchV2(t, srv, "/indexes/korsia/search?q=pepe")["hits"].([]interface{})
	if hits[0].(map[string]interface{})["id"] != "1" {
		t.Fatalf("il prodotto, raddoppiato, deve venire prima: %v", hits)
	}
}

func TestSynonymLimitsAreCheckedOverHTTP(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	richiesta(t, srv, "POST", "/indexes", `{"name":"korsia"}`)
	if w := richiesta(t, srv, "PUT", "/indexes/korsia/settings", `{"searchable_fields": ["name"], "synonym_fields": ["nmae"]}`); w.Code != http.StatusBadRequest {
		t.Fatalf("un campo dei sinonimi che non si cerca va rifiutato: %d", w.Code)
	}
}

// Una PUT che cambia solo gruppi e boost non reindicizza e arriva in pochi
// millisecondi, mentre le ricerche girano: ognuna deve leggere impostazioni
// intere (go test -race) e nessun risultato vecchio deve restare in cache.
func TestSettingsChangeWhileSearching(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	richiesta(t, srv, "POST", "/indexes", `{"name":"korsia"}`)
	richiesta(t, srv, "POST", "/indexes/korsia/documents", `[
		{"id": "1", "kind": "product", "name": "Bibite gassate"},
		{"id": "2", "kind": "zone", "name": "Bevande"}]`)

	var wg sync.WaitGroup
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				richiesta(t, srv, "GET", "/indexes/korsia/search?q=bibite", "")
			}
		}()
	}
	for i := 0; i < 50; i++ {
		corpo := fmt.Sprintf(`{"searchable_fields": ["name"], "synonym_groups": [["bibite", "bevande"]], "boosts": [{"filter": "kind=zone", "factor": %d}]}`, 1+i%3)
		if i%2 == 1 {
			corpo = `{"searchable_fields": ["name"]}`
		}
		richiesta(t, srv, "PUT", "/indexes/korsia/settings", corpo)
	}
	wg.Wait()

	// The last PUT has no groups: "bibite" finds only the product.
	if n := len(searchV2(t, srv, "/indexes/korsia/search?q=bibite")["hits"].([]interface{})); n != 1 {
		t.Fatalf("dopo l'ultima PUT \"bibite\" trova %d documenti, ne serve 1", n)
	}
}

func TestEverySettingsUpdateIsANewGeneration(t *testing.T) {
	mgr, err := manager.NewManager(storage.Options{DataDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Close()
	_ = mgr.CreateIndex("k")
	_, prima, _ := mgr.Impostazioni("k")
	s := engine.DefaultSettings()
	s.SynonymGroups = [][]string{{"bibite", "bevande"}}
	_ = mgr.UpdateSettings("k", s)
	if _, dopo, _ := mgr.Impostazioni("k"); dopo == prima {
		t.Fatalf("la generazione non cambia: %d", dopo)
	}
}

// Settings replayed from the WAL apply to the documents of the snapshot too.
func TestReplayedSettingsReindexTheSnapshot(t *testing.T) {
	dir := t.TempDir()
	opts := storage.Options{DataDir: dir}
	mgr, err := manager.NewManager(opts)
	if err != nil {
		t.Fatal(err)
	}
	_ = mgr.CreateIndex("k")
	_, _, _ = mgr.AddDocuments("k", []map[string]interface{}{{"id": "1", "name": "Matrix", "code": "xr7"}})
	mgr.Close()

	s := engine.DefaultSettings()
	s.SearchableFields = []string{"name"}
	p := storage.NewPersistence(opts)
	if err := p.AppendWAL(storage.WALOperation{Op: "UPDATE_SETTINGS", Index: "k", Settings: &s}); err != nil {
		t.Fatal(err)
	}
	p.Wait()

	mgr, err = manager.NewManager(opts)
	if err != nil {
		t.Fatal(err)
	}
	defer mgr.Close()
	idx, _ := mgr.GetIndex("k")
	impostazioni, _, _ := mgr.Impostazioni("k")
	if ids, _ := idx.Engine.Search("xr7", impostazioni, "0", nil); len(ids) != 0 {
		t.Fatalf("code non si cerca più, ma \"xr7\" trova %v", ids)
	}
	_ = os.RemoveAll(dir)
}
