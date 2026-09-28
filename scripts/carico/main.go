// Command carico misura quanto costano Elasticsearch e Koskidex sulla macchina,
// con le stesse ricerche che manda Documentale: una ricerca alla volta, sotto
// carico con piu' client in parallelo, a riposo. Esperimento 2026-09-28_carico
// nell'archivio della tesi.
//
//	go run ./scripts/carico -motore es -url http://localhost:9201 -indice search-documents-local \
//	    -query known-item-auto=q.jsonl -query confronto-24=c.jsonl -modo sequenziale -etichetta es
//
// Modi:
//   - verifica: per ogni query, i risultati con e senza spazi in coda devono
//     essere identici (e' il modo di aggirare la cache di Koskidex);
//   - sequenziale: ogni query -ripetizioni volte, una alla volta;
//   - carico: -concorrenza client senza pause, -durata per livello;
//   - riposo: memoria e CPU senza richieste, per -durata;
//   - risposte: per ogni query l'impronta SHA-256 della lista completa dei
//     risultati, id e punteggi come li scrive il motore, e quanti sono: due
//     versioni del motore danno gli stessi risultati se le impronte coincidono.
//
// Le richieste sono quelle dell'app: per Elasticsearch il multi_match di
// ElasticsearchService::cerca con size 10000 (-sorgente=false toglie il
// _source dalla risposta), per Koskidex la GET di KoskidexClient::search.
// Con -senza-cache ogni richiesta porta un numero di spazi in coda diverso:
// il tokenizer li ignora, la chiave della cache di Koskidex no.
package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/GeneralKoski/Koskidex/internal/eval"
)

type query struct {
	Famiglia string
	ID       string
	Testo    string
}

type fileQuery []string

func (f *fileQuery) String() string     { return strings.Join(*f, ",") }
func (f *fileQuery) Set(v string) error { *f = append(*f, v); return nil }

var (
	motore      = flag.String("motore", "", "es oppure koskidex")
	base        = flag.String("url", "", "indirizzo del motore")
	indice      = flag.String("indice", "search-documents-local", "indice da interrogare")
	modo        = flag.String("modo", "sequenziale", "verifica, sequenziale, carico, riposo, risposte")
	ripetizioni = flag.Int("ripetizioni", 5, "sequenziale: quante volte ogni query")
	livelli     = flag.String("concorrenza", "1,2,4,8,16,32", "carico: client in parallelo, per livello")
	durata      = flag.Duration("durata", 20*time.Second, "carico e riposo: durata di un livello")
	riscalda    = flag.Duration("riscaldamento", 5*time.Second, "carico: riscaldamento prima di ogni livello")
	senzaCache  = flag.Bool("senza-cache", true, "spazi in coda diversi a ogni richiesta")
	sorgente    = flag.Bool("sorgente", true, "es: _source nella risposta, come l'app")
	container   = flag.String("container", "", "container di cui campionare CPU e memoria")
	pid         = flag.Int("pid", 0, "processo di cui campionare CPU e memoria")
	etichetta   = flag.String("etichetta", "", "nome del file di esito")
	esperimento = flag.String("esperimento", "2026-09-28_carico", "cartella dell'esperimento nell'archivio")
	seme        = flag.Int64("seme", 20260928, "seme per l'ordine delle query")
	fileDiQuery fileQuery
)

var client *http.Client

func main() {
	flag.Var(&fileDiQuery, "query", "famiglia=percorso di un queries.jsonl (ripetibile)")
	flag.Parse()
	if (*motore != "es" && *motore != "koskidex") || *base == "" || *etichetta == "" {
		fmt.Fprintln(os.Stderr, "servono -motore es|koskidex, -url e -etichetta")
		os.Exit(2)
	}
	client = &http.Client{Timeout: 60 * time.Second, Transport: &http.Transport{MaxIdleConns: 256, MaxIdleConnsPerHost: 256}}

	queries, impronte := leggiQuery()
	esito := map[string]interface{}{
		"ran_at": time.Now().UTC().Format(time.RFC3339),
		"config": configurazione(len(queries), impronte),
	}
	switch *modo {
	case "verifica":
		esito["verifica"] = verifica(queries)
	case "sequenziale":
		esito["sequenziale"] = sequenziale(queries)
	case "carico":
		esito["carico"] = carico(queries)
	case "riposo":
		esito["riposo"] = campiona(*durata)
	case "risposte":
		esito["risposte"] = risposte(queries)
	default:
		fmt.Fprintln(os.Stderr, "modo sconosciuto:", *modo)
		os.Exit(2)
	}
	dati, _ := json.MarshalIndent(esito, "", "  ")
	path, err := eval.Archivia("esperimenti/"+*esperimento, *etichetta, dati)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if path == "" {
		fmt.Fprintln(os.Stderr, "!!! RISULTATO NON ARCHIVIATO: TESI_RISULTATI non è impostata.")
		os.Exit(1)
	}
	fmt.Println("archiviato in", path)
}

func leggiQuery() ([]query, map[string]interface{}) {
	var out []query
	impronte := map[string]interface{}{}
	for _, v := range fileDiQuery {
		famiglia, percorso, ok := strings.Cut(v, "=")
		if !ok {
			fmt.Fprintln(os.Stderr, "-query vuole famiglia=percorso:", v)
			os.Exit(2)
		}
		grezzo, err := os.ReadFile(percorso)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		h := sha256.Sum256(grezzo)
		n := 0
		sc := bufio.NewScanner(bytes.NewReader(grezzo))
		sc.Buffer(make([]byte, 1<<20), 1<<24)
		for sc.Scan() {
			var r struct {
				ID   string `json:"_id"`
				Text string `json:"text"`
			}
			if json.Unmarshal(sc.Bytes(), &r) == nil && strings.TrimSpace(r.Text) != "" {
				out = append(out, query{famiglia, r.ID, r.Text})
				n++
			}
		}
		impronte[famiglia] = map[string]interface{}{"query": n, "sha256": hex.EncodeToString(h[:])}
	}
	if len(out) == 0 {
		fmt.Fprintln(os.Stderr, "nessuna query")
		os.Exit(2)
	}
	return out, impronte
}

func configurazione(n int, impronte map[string]interface{}) map[string]interface{} {
	c := map[string]interface{}{}
	for k, v := range eval.Provenienza(".") {
		c[k] = v
	}
	c["motore"] = *motore
	c["url"] = *base
	c["indice"] = *indice
	c["modo"] = *modo
	c["query"] = n
	c["famiglie"] = impronte
	c["senza_cache"] = *senzaCache
	c["seme"] = *seme
	c["client_cpu"] = runtime.NumCPU()
	c["client_go"] = runtime.Version()
	if *motore == "es" {
		c["sorgente"] = *sorgente
		var info struct {
			Version struct {
				Number string `json:"number"`
			} `json:"version"`
		}
		if r, err := client.Get(*base); err == nil {
			_ = json.NewDecoder(r.Body).Decode(&info)
			r.Body.Close()
		}
		c["versione"] = info.Version.Number
	} else {
		var info map[string]interface{}
		if r, err := client.Get(*base + "/health"); err == nil {
			_ = json.NewDecoder(r.Body).Decode(&info)
			r.Body.Close()
		}
		c["versione"] = info["version"]
	}
	if *container != "" {
		c["container"] = *container
	}
	if *pid != 0 {
		c["pid"] = *pid
	}
	switch *modo {
	case "sequenziale":
		c["ripetizioni"] = *ripetizioni
	case "carico":
		c["concorrenza"] = *livelli
		c["durata_s"] = durata.Seconds()
		c["riscaldamento_s"] = riscalda.Seconds()
	case "riposo":
		c["durata_s"] = durata.Seconds()
	}
	if out, err := exec.Command("docker", "info", "--format", "{{.NCPU}} {{.MemTotal}}").Output(); err == nil {
		c["docker_vm"] = strings.TrimSpace(string(out))
	}
	return c
}

// richiesta costruisce la ricerca come la fa l'app.
func richiesta(testo string) (*http.Request, error) {
	if *motore == "es" {
		corpo := map[string]interface{}{
			"query": map[string]interface{}{"multi_match": map[string]interface{}{
				"query": testo, "type": "best_fields", "fuzziness": "AUTO", "prefix_length": 1,
				"operator": "and", "tie_breaker": 0.3,
				"fields": []string{"name^5", "tags^4", "summary^3", "subjects^3", "notes^2", "additional_data^1"},
			}},
			"size": 10000,
		}
		if !*sorgente {
			corpo["_source"] = false
		}
		b, _ := json.Marshal(corpo)
		r, err := http.NewRequest("POST", *base+"/"+*indice+"/_search", bytes.NewReader(b))
		if err == nil {
			r.Header.Set("Content-Type", "application/json")
		}
		return r, err
	}
	v := url.Values{"q": {testo}, "limit": {"10000"}, "ids_only": {"true"}, "fuzziness": {"AUTO"}}
	return http.NewRequest("GET", *base+"/indexes/"+*indice+"/search?"+v.Encode(), nil)
}

type risposta struct {
	ms    float64
	byte  int64
	ids   []string
	stato int
	err   error
}

// esegui manda una ricerca; con leggiIDs decodifica anche gli id dei risultati.
func esegui(testo string, leggiIDs bool) risposta {
	r, err := richiesta(testo)
	if err != nil {
		return risposta{err: err}
	}
	t0 := time.Now()
	resp, err := client.Do(r)
	if err != nil {
		return risposta{err: err}
	}
	var buf bytes.Buffer
	var n int64
	if leggiIDs {
		n, err = io.Copy(&buf, resp.Body)
	} else {
		n, err = io.Copy(io.Discard, resp.Body)
	}
	resp.Body.Close()
	out := risposta{ms: float64(time.Since(t0).Microseconds()) / 1000, byte: n, stato: resp.StatusCode, err: err}
	if err == nil && resp.StatusCode != 200 {
		out.err = fmt.Errorf("stato %d", resp.StatusCode)
	}
	if leggiIDs && out.err == nil {
		out.ids, out.err = ids(buf.Bytes())
	}
	return out
}

func ids(b []byte) ([]string, error) {
	if *motore == "es" {
		var r struct {
			Hits struct {
				Hits []struct {
					ID string `json:"_id"`
				} `json:"hits"`
			} `json:"hits"`
		}
		if err := json.Unmarshal(b, &r); err != nil {
			return nil, err
		}
		out := make([]string, len(r.Hits.Hits))
		for i, h := range r.Hits.Hits {
			out[i] = h.ID
		}
		return out, nil
	}
	var r struct {
		Hits []struct {
			ID string `json:"id"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, err
	}
	out := make([]string, len(r.Hits))
	for i, h := range r.Hits {
		out[i] = h.ID
	}
	return out, nil
}

// spazi dà a ogni richiesta della stessa query un numero di spazi diverso,
// fino a 50: una chiave si ripete solo dopo 50 richieste della stessa query,
// quando nella cache da 1.024 voci e' gia' stata sostituita.
type spazi struct {
	mu sync.Mutex
	n  map[string]int
}

func (s *spazi) testo(q query) string {
	if !*senzaCache {
		return q.Testo
	}
	s.mu.Lock()
	s.n[q.ID]++
	k := (s.n[q.ID]-1)%50 + 1
	s.mu.Unlock()
	return q.Testo + strings.Repeat(" ", k)
}

// risposte registra, per ogni query, l'impronta dei risultati così come il
// motore li scrive: per Koskidex l'array hits con id e punteggio, byte per byte.
func risposte(queries []query) map[string]interface{} {
	out := map[string]interface{}{}
	errori := 0
	for _, q := range queries {
		r, err := richiesta(q.Testo)
		if err != nil {
			errori++
			continue
		}
		resp, err := client.Do(r)
		if err != nil {
			errori++
			continue
		}
		b, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil || resp.StatusCode != 200 {
			errori++
			continue
		}
		var corpo struct {
			Hits json.RawMessage `json:"hits"`
		}
		if json.Unmarshal(b, &corpo) != nil {
			errori++
			continue
		}
		h := sha256.Sum256(corpo.Hits)
		n, _ := ids(b)
		out[q.Famiglia+"/"+q.ID] = map[string]interface{}{"sha256": hex.EncodeToString(h[:]), "risultati": len(n)}
	}
	fmt.Printf("risposte: %d query, %d errori\n", len(out), errori)
	return map[string]interface{}{"errori": errori, "query": out}
}

func verifica(queries []query) map[string]interface{} {
	diverse, errori := []string{}, 0
	for _, q := range queries {
		a := esegui(q.Testo, true)
		b := esegui(q.Testo+strings.Repeat(" ", 7), true)
		if a.err != nil || b.err != nil {
			errori++
			continue
		}
		if strings.Join(a.ids, ",") != strings.Join(b.ids, ",") {
			diverse = append(diverse, q.Famiglia+"/"+q.ID)
		}
	}
	fmt.Printf("verifica: %d query, %d con risultati diversi, %d errori\n", len(queries), len(diverse), errori)
	return map[string]interface{}{"query": len(queries), "diverse": diverse, "errori": errori}
}

func percentili(v []float64) map[string]interface{} {
	if len(v) == 0 {
		return map[string]interface{}{"n": 0}
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	p := func(q float64) float64 {
		i := int(math.Ceil(q*float64(len(s)))) - 1
		if i < 0 {
			i = 0
		}
		return math.Round(s[i]*1000) / 1000
	}
	somma := 0.0
	for _, x := range s {
		somma += x
	}
	return map[string]interface{}{"n": len(s), "p50": p(0.5), "p95": p(0.95), "p99": p(0.99), "max": p(1),
		"media": math.Round(somma/float64(len(s))*1000) / 1000}
}

func fascia(n int) string {
	switch {
	case n < 100:
		return "sotto 100"
	case n < 1000:
		return "100-999"
	}
	return "da 1000"
}

func sequenziale(queries []query) map[string]interface{} {
	sp := &spazi{n: map[string]int{}}
	rng := rand.New(rand.NewSource(*seme))
	type riga struct {
		Famiglia  string  `json:"famiglia"`
		ID        string  `json:"id"`
		Passata   int     `json:"passata"`
		Ms        float64 `json:"ms"`
		Byte      int64   `json:"byte"`
		Risultati int     `json:"risultati"`
	}
	var righe []riga
	errori := 0
	for p := 1; p <= *ripetizioni; p++ {
		ordine := rng.Perm(len(queries))
		for _, i := range ordine {
			q := queries[i]
			r := esegui(sp.testo(q), true)
			if r.err != nil {
				errori++
				continue
			}
			righe = append(righe, riga{q.Famiglia, q.ID, p, r.ms, r.byte, len(r.ids)})
		}
		fmt.Printf("passata %d di %d\n", p, *ripetizioni)
	}
	gruppi := map[string][]float64{}
	byteGruppi := map[string][]float64{}
	for _, r := range righe {
		chiavi := []string{"tutte", "famiglia " + r.Famiglia, "risultati " + fascia(r.Risultati)}
		if r.Passata == 1 {
			chiavi = append(chiavi, "prima passata")
		} else {
			chiavi = append(chiavi, "passate successive")
		}
		for _, k := range chiavi {
			gruppi[k] = append(gruppi[k], r.Ms)
			byteGruppi[k] = append(byteGruppi[k], float64(r.Byte))
		}
	}
	riassunto := map[string]interface{}{}
	for k, v := range gruppi {
		m := percentili(v)
		m["byte"] = percentili(byteGruppi[k])
		riassunto[k] = m
	}
	t := percentili(gruppi["tutte"])
	fmt.Printf("tutte: p50 %v ms, p95 %v ms, p99 %v ms, errori %d\n", t["p50"], t["p95"], t["p99"], errori)
	return map[string]interface{}{"errori": errori, "riassunto": riassunto, "richieste": righe}
}

type campione struct {
	T   float64 `json:"t"`
	MB  float64 `json:"mb"`
	CPU float64 `json:"cpu"`
}

func memoriaMB(s string) float64 {
	s = strings.TrimSpace(s)
	unita := map[string]float64{"GiB": 1024, "MiB": 1, "KiB": 1.0 / 1024, "GB": 1000, "MB": 1, "kB": 1.0 / 1000, "B": 1.0 / (1 << 20)}
	for _, u := range []string{"GiB", "MiB", "KiB", "GB", "MB", "kB", "B"} {
		if strings.HasSuffix(s, u) {
			v, _ := strconv.ParseFloat(strings.TrimSuffix(s, u), 64)
			return v * unita[u]
		}
	}
	return 0
}

func unCampione() (campione, bool) {
	switch {
	case *container != "":
		out, err := exec.Command("docker", "stats", "--no-stream", "--format", "{{.MemUsage}}|{{.CPUPerc}}", *container).Output()
		if err != nil {
			return campione{}, false
		}
		parti := strings.Split(strings.TrimSpace(string(out)), "|")
		if len(parti) != 2 {
			return campione{}, false
		}
		mem := strings.Split(parti[0], "/")[0]
		cpu, _ := strconv.ParseFloat(strings.TrimSuffix(strings.TrimSpace(parti[1]), "%"), 64)
		return campione{MB: memoriaMB(mem), CPU: cpu}, true
	case *pid != 0:
		out, err := exec.Command("ps", "-o", "rss=,%cpu=", "-p", strconv.Itoa(*pid)).Output()
		if err != nil {
			return campione{}, false
		}
		f := strings.Fields(string(out))
		if len(f) != 2 {
			return campione{}, false
		}
		kb, _ := strconv.ParseFloat(f[0], 64)
		cpu, _ := strconv.ParseFloat(f[1], 64)
		return campione{MB: kb / 1024, CPU: cpu}, true
	}
	return campione{}, false
}

// campionatore raccoglie campioni finché stop non si chiude.
func campionatore(stop <-chan struct{}) <-chan []campione {
	out := make(chan []campione, 1)
	go func() {
		var c []campione
		t0 := time.Now()
		for {
			if s, ok := unCampione(); ok {
				s.T = math.Round(time.Since(t0).Seconds()*10) / 10
				c = append(c, s)
			}
			select {
			case <-stop:
				out <- c
				return
			case <-time.After(500 * time.Millisecond):
			}
		}
	}()
	return out
}

func riassumiCampioni(c []campione) map[string]interface{} {
	if len(c) == 0 {
		return map[string]interface{}{"campioni": 0}
	}
	var mb, cpu []float64
	for _, s := range c {
		mb = append(mb, s.MB)
		cpu = append(cpu, s.CPU)
	}
	return map[string]interface{}{"campioni": len(c), "memoria_mb": percentili(mb), "cpu_percento": percentili(cpu)}
}

func campiona(d time.Duration) map[string]interface{} {
	stop := make(chan struct{})
	ris := campionatore(stop)
	time.Sleep(d)
	close(stop)
	c := <-ris
	r := riassumiCampioni(c)
	fmt.Printf("riposo: %v\n", r)
	return map[string]interface{}{"riassunto": r, "campioni": c}
}

func carico(queries []query) []map[string]interface{} {
	var out []map[string]interface{}
	sp := &spazi{n: map[string]int{}}
	for _, l := range strings.Split(*livelli, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(l))
		if err != nil || n < 1 {
			fmt.Fprintln(os.Stderr, "livello non valido:", l)
			os.Exit(2)
		}
		livello(queries, sp, n, *riscalda, false)
		stop := make(chan struct{})
		ris := campionatore(stop)
		lat, errori, trascorso := livello(queries, sp, n, *durata, true)
		close(stop)
		c := <-ris
		m := map[string]interface{}{
			"client":              n,
			"richieste":           len(lat),
			"errori":              errori,
			"ricerche_al_secondo": math.Round(float64(len(lat))/trascorso.Seconds()*10) / 10,
			"latenza_ms":          percentili(lat),
			"risorse":             riassumiCampioni(c),
		}
		p := m["latenza_ms"].(map[string]interface{})
		fmt.Printf("%2d client: %7.1f ricerche/s, p50 %v ms, p99 %v ms, errori %d\n", n, m["ricerche_al_secondo"], p["p50"], p["p99"], errori)
		out = append(out, m)
	}
	return out
}

// livello tiene n client occupati per d; con misura raccoglie le latenze.
func livello(queries []query, sp *spazi, n int, d time.Duration, misura bool) ([]float64, int64, time.Duration) {
	var mu sync.Mutex
	var lat []float64
	var errori int64
	var wg sync.WaitGroup
	fine := time.Now().Add(d)
	t0 := time.Now()
	for w := 0; w < n; w++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(*seme + int64(w)))
			var mie []float64
			for time.Now().Before(fine) {
				r := esegui(sp.testo(queries[rng.Intn(len(queries))]), false)
				if r.err != nil {
					atomic.AddInt64(&errori, 1)
					continue
				}
				mie = append(mie, r.ms)
			}
			if misura {
				mu.Lock()
				lat = append(lat, mie...)
				mu.Unlock()
			}
		}(w)
	}
	wg.Wait()
	return lat, errori, time.Since(t0)
}
