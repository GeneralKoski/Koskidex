package tests

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GeneralKoski/Koskidex/internal/manager"
	"github.com/GeneralKoski/Koskidex/internal/server"
	"github.com/GeneralKoski/Koskidex/internal/storage"
)

func serverSuCartella(t *testing.T, dir string) (*server.Server, *manager.Manager) {
	t.Helper()
	mgr, err := manager.NewManager(storage.Options{DataDir: dir})
	if err != nil {
		t.Fatalf("avvio del manager fallito: %v", err)
	}
	return server.NewServer(mgr, "", 0, "*"), mgr
}

func seminaCinque(t *testing.T, srv *server.Server) {
	t.Helper()
	richiesta(t, srv, "POST", "/indexes", `{"name":"documents"}`)
	w := richiesta(t, srv, "POST", "/indexes/documents/documents", `[
		{"id": 1, "title": "delibera uno"}, {"id": 2, "title": "delibera due"},
		{"id": 3, "title": "delibera tre"}, {"id": 4, "title": "delibera quattro"},
		{"id": 5, "title": "delibera cinque"}]`)
	if w.Code != http.StatusAccepted {
		t.Fatalf("semina fallita: %d %s", w.Code, w.Body.String())
	}
}

func idTrovati(t *testing.T, srv *server.Server) map[string]bool {
	t.Helper()
	res := searchV2(t, srv, "/indexes/documents/search?q=delibera&ids_only=true")
	out := map[string]bool{}
	for _, h := range res["hits"].([]interface{}) {
		out[h.(map[string]interface{})["id"].(string)] = true
	}
	return out
}

// Documentale cancella con bulkDeleteElement, un deleteByQuery sugli id: gli
// id arrivano interi o stringhe, e quelli che non esistono non sono un errore.
func TestDeleteBatchRemovesTheGivenIDs(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	seminaCinque(t, srv)
	// Mette la ricerca in cache: dopo la cancellazione non deve tornare indietro.
	idTrovati(t, srv)

	w := richiesta(t, srv, "POST", "/indexes/documents/documents/delete-batch", `[1, "3", 99, 3]`)
	if w.Code != http.StatusOK {
		t.Fatalf("attesa 200, avuta %d: %s", w.Code, w.Body.String())
	}
	var r map[string]interface{}
	_ = json.NewDecoder(w.Body).Decode(&r)
	if r["deleted"] != float64(2) {
		t.Fatalf("cancellati davvero sono 2 (1 e 3; 99 non c'è, 3 è ripetuto), la risposta dice %v", r["deleted"])
	}

	trovati := idTrovati(t, srv)
	if len(trovati) != 3 || trovati["1"] || trovati["3"] || !trovati["2"] || !trovati["4"] || !trovati["5"] {
		t.Fatalf("dopo la cancellazione devono restare 2, 4 e 5: %v", trovati)
	}
	if w := richiesta(t, srv, "GET", "/indexes/documents/documents/1", ""); w.Code != http.StatusNotFound {
		t.Fatalf("il documento 1 non deve più esistere: %d", w.Code)
	}
	if n := documentiNellIndice(t, srv, "documents"); n != 3 {
		t.Fatalf("l'indice deve contare 3 documenti, ne conta %v", n)
	}
}

// Come per l'aggiunta: un id inutilizzabile fa rifiutare tutto, e nessun
// documento viene toccato.
func TestDeleteBatchWithAnInvalidIDDeletesNothing(t *testing.T) {
	srv, cleanup := setupTestServer(t)
	defer cleanup()
	seminaCinque(t, srv)

	for _, corpo := range []string{`[1, ""]`, `[2, 1.5]`, `[3, null]`, `{"ids": [4]}`, `[]`} {
		w := richiesta(t, srv, "POST", "/indexes/documents/documents/delete-batch", corpo)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("%s: atteso 400, avuto %d: %s", corpo, w.Code, w.Body.String())
		}
	}
	if n := len(idTrovati(t, srv)); n != 5 {
		t.Fatalf("una richiesta rifiutata non deve cancellare niente: restano %d documenti su 5", n)
	}

	if w := richiesta(t, srv, "POST", "/indexes/manca/documents/delete-batch", `[1]`); w.Code != http.StatusNotFound {
		t.Fatalf("indice inesistente: atteso 404, avuto %d", w.Code)
	}
}

// Un solo record nel WAL e una sola sincronizzazione per tutta la richiesta,
// invece di un record per id.
func TestDeleteBatchWritesOneWALRecord(t *testing.T) {
	dir := t.TempDir()
	srv, mgr := serverSuCartella(t, dir)
	defer mgr.Close()
	seminaCinque(t, srv)

	richiesta(t, srv, "POST", "/indexes/documents/documents/delete-batch", `[1, 2, 3]`)

	// L'istantanea che svuota il WAL parte un secondo dopo l'ultima scrittura:
	// qui il WAL ha ancora tutto.
	dati, err := os.ReadFile(filepath.Join(dir, "operations.log"))
	if err != nil {
		t.Fatal(err)
	}
	var cancellazioni []storage.WALOperation
	for _, riga := range strings.Split(strings.TrimSpace(string(dati)), "\n") {
		var op storage.WALOperation
		if err := json.Unmarshal([]byte(riga), &op); err != nil {
			t.Fatalf("riga del WAL illeggibile: %q", riga)
		}
		if strings.HasPrefix(op.Op, "DELETE") {
			cancellazioni = append(cancellazioni, op)
		}
	}
	if len(cancellazioni) != 1 || cancellazioni[0].Op != "DELETE_DOCS" || len(cancellazioni[0].DocIDs) != 3 {
		t.Fatalf("atteso un solo record DELETE_DOCS con 3 id, trovati: %+v", cancellazioni)
	}
}

// Se il processo muore prima dell'istantanea, al riavvio il WAL va rigiocato
// e i documenti cancellati devono restare cancellati.
func TestDeleteBatchSurvivesAWALReplay(t *testing.T) {
	dir := t.TempDir()
	p := storage.NewPersistence(storage.Options{DataDir: dir})
	for _, op := range []storage.WALOperation{
		{Op: "CREATE_INDEX", Index: "documents"},
		{Op: "ADD_DOC", Index: "documents", DocID: "1", DocData: map[string]interface{}{"id": "1", "title": "delibera uno"}},
		{Op: "ADD_DOC", Index: "documents", DocID: "2", DocData: map[string]interface{}{"id": "2", "title": "delibera due"}},
		{Op: "ADD_DOC", Index: "documents", DocID: "3", DocData: map[string]interface{}{"id": "3", "title": "delibera tre"}},
		{Op: "DELETE_DOCS", Index: "documents", DocIDs: []string{"1", "3"}},
	} {
		if err := p.AppendWAL(op); err != nil {
			t.Fatal(err)
		}
	}
	p.Wait()

	srv, mgr := serverSuCartella(t, dir)
	defer mgr.Close()
	if trovati := idTrovati(t, srv); len(trovati) != 1 || !trovati["2"] {
		t.Fatalf("dopo il replay deve restare solo il 2: %v", trovati)
	}
}
