# Implementations — Koskidex

Tracking delle migliorie per portare il progetto a maturità di produzione.
Il core (engine, server, storage) è già feature-complete; qui si colmano le lacune
di documentazione, sicurezza e testing.

Legenda: ✅ fatto · 🚧 in corso · ⬜ da fare

## Bug & correttezza

- ✅ **Documenti senza `id` scartati in silenzio** — `AddDocuments` ora ritorna
  `{added, skipped}` invece di un generico "count", così il client sa quanti
  documenti sono stati realmente indicizzati.
- ✅ **Race condition in `UpdateSettings`** — il re-index leggeva i doc fuori dal
  lock e faceva lo swap del puntatore `idx.Engine` (data race + perdita di doc
  concorrenti). Ora gli writer del manager (`AddDocuments`, `DeleteDocument`,
  `UpdateSettings`) sono serializzati su `m.mu` e il re-index avviene in-place via
  `InvertedIndex.Reindex` (sotto il lock dell'engine, nessuno swap).
- ✅ **Validazione input** — nome indice trimmato/validato (lunghezza, `/`, `\`,
  `..`); limite di 1000 caratteri sulla query di ricerca.

## API

- ✅ **`GET /indexes/{name}/documents`** — endpoint di listing paginato dei
  documenti (ordinati per id) che prima mancava.

## Priorità alta

- ✅ **LICENSE** — il README dichiara MIT ma il file non esisteva.
- ✅ **Sanitizzazione errori API** — gli handler ritornavano `err.Error()` grezzo,
  esponendo dettagli interni. Ora `sendInternalError` logga l'errore reale via
  `slog` e ritorna un messaggio generico al client.
- ✅ **Test frontend** — setup Vitest + Testing Library (jsdom). Test per `NotFound`
  e `ErrorBoundary`. Script `npm test` / `npm run test:watch`.
- ✅ **SECURITY.md** — policy di segnalazione vulnerabilità.

## Priorità media

- ✅ **.env.example funzionale** — `main.go` ignorava le env var (solo flag). Ora
  i flag usano le `KOSKIDEX_*` come default (flag override env) e l'`.env.example`
  documenta tutte le variabili realmente lette (porta, data dir, api key, log, rate
  limit, TLS).
- ✅ **CONTRIBUTING.md** — linee guida per i contributor.
- ⬜ **CHANGELOG.md** — storico versioni (Keep a Changelog).
- ℹ️ **GitHub Actions rimosse** — CI (`ci.yml`) e deploy (`deploy.yml`) eliminati.
  Il deploy è ora manuale via lo script centralizzato `deploy.sh` (SSH + `docker compose`
  sul VPS); test e `govulncheck` vanno lanciati in locale.

## Produzione & SEO (audit completo)

### Backend
- ✅ **Timeout HTTP + graceful shutdown** — `*http.Server` con Read/Write/Idle
  timeout (anti-Slowloris) e shutdown su SIGINT/SIGTERM con drain delle connessioni.
- ✅ **Sitemap/robots hardened** — escaping XML del `<loc>`, build URL senza hack
  di string-replace, cap a 50k URL, `<lastmod>`; robots espone le direttive
  `Sitemap:` per indice.
- ✅ **Limiti dimensione body** — `MaxBytesReader` su create/settings/search (1MB)
  e upload documenti (64MB).
- ✅ **CORS configurabile** — `--cors-origin` / `KOSKIDEX_CORS_ORIGIN` (default `*`).
- ✅ **Health check arricchito** — riporta numero indici e documenti totali.
- ✅ **Rate limiter** — goroutine di cleanup arrestabile via `Stop()` allo shutdown.

### Frontend / SEO
- ✅ **Meta tag per-route** — hook `useDocumentMeta` (title/description/OG/canonical
  distinti per Home, Documentation, NotFound).
- ✅ **og-image** — da ~5 MB a ~107 KB (1200×633 JPEG) + `og:image:width/height`.
- ✅ **sitemap.xml** — aggiunto `/docs`, date aggiornate, `changefreq`/`priority`.
- ✅ **manifest.json + theme-color + apple-touch-icon**.
- ✅ **nginx** — gzip, security header (nosniff, X-Frame-Options, Referrer-Policy,
  Permissions-Policy), `no-cache` su index.html, `immutable` sugli asset hashati,
  `X-Forwarded-Proto` al backend.

### Docker
- ✅ **Container non-root** (utente `koskidex`) + `COPY go.sum` per build riproducibili.

### Test
- ✅ Test backend per cache, filtri, escaping sitemap, robots, listing documenti.

## Priorità bassa (residuo)

- ⬜ Documentazione formato WAL/GOB e runbook backup/recovery.
- ⬜ Completare i pacchetti client in `examples/` (laravel, nodejs, python).
- ⬜ CSP in nginx (richiede test: inline styles Tailwind + Google Fonts).
- ⬜ HSTS (da impostare sul reverse proxy TLS, non su nginx :80).
