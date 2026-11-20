# CLAUDE.md

Fork Dieffetech di [GeneralKoski/Koskidex](https://github.com/GeneralKoski/Koskidex), il motore di ricerca di un
collega. Lo usa in produzione Welintouch (ricerca pacchetti per significato). Il fork esiste per non dipendere dai
tempi del repo originale: correzioni e aggiornamenti li rilasciamo da qui, e li proponiamo sempre anche a lui.

## Il doppio lavoro: fork e originale allineati

Remote: `origin` = `Dieffetech/koskidex`, `upstream` = `GeneralKoski/Koskidex`.

```bash
git remote add upstream https://github.com/GeneralKoski/Koskidex.git   # una volta
git fetch upstream && git merge upstream/main                           # prima di iniziare a lavorare
```

1. **Si lavora su `main` del fork**, commit diretti. Niente branch nostri a lunga vita.
2. **Ogni push su `main` va proposto all'originale.** Lo fa da solo
   [`upstream-sync.yml`](.github/workflows/upstream-sync.yml): tiene aperta **una** PR `Dieffetech:main` →
   `GeneralKoski/Koskidex:main` e la aggiorna a ogni push. Il collega decide se e quando fare il merge.
   Gli chiediamo "Create a merge commit": con squash o rebase non si rompe niente (il confronto è sul
   contenuto), ma le storie si separano.
3. **Ogni lunedì, e a ogni push, `upstream/main` entra nel fork** con lo stesso workflow. Un conflitto apre
   una issue con etichetta `upstream-sync`: si risolve a mano con il merge sopra.
4. **Tutto quello che finisce su `main` finisce nella PR**, anche `.github/` e questo file. Una modifica che
   serve solo a noi va evitata. Se proprio serve, la si spiega nella PR, e se il collega la rifiuta resta qui
   come unica differenza nota.
5. **Le regole del repo originale valgono anche qui**, così la PR resta accettabile:
   [`CONTRIBUTING.md`](CONTRIBUTING.md) (commit Conventional Commits in inglese, test per ogni cambio di
   comportamento, dipendenze al minimo) e le regole di [`eval/BASELINE.md`](eval/BASELINE.md): un cambio
   di ordinamento dei risultati va dietro un campo di `Settings`, `TestBaselineRankingIsFrozen` non si
   tocca, e serve una voce in `eval/DIARIO.md`.
6. **Non riformattare i file altrui**: alcuni file non passano `gofmt` già nell'originale, e riformattarli
   gonfierebbe la PR. Si formatta solo ciò che si scrive.

La PR all'originale la apre il secret **`UPSTREAM_PR_TOKEN`**: token *classic* con scope `public_repo`
dell'account che firma la PR (`GITHUB_TOKEN` non può aprire PR in un altro repository), salvato su Bitwarden.
Senza secret il workflow si limita a un avviso.

## Controlli automatici

| Cosa | Dove | Quando |
|---|---|---|
| vet + test Go, lint + test + build del playground | [`ci.yml`](.github/workflows/ci.yml) | push su `main`, ogni PR |
| govulncheck, trivy sull'immagine, npm audit, fine supporto di Go/Alpine/Node | [`weekly-security.yml`](.github/workflows/weekly-security.yml) | lunedì 05:00 UTC, PR che toccano le dipendenze, a mano |
| PR di aggiornamento per Go, immagini Docker, npm, GitHub Actions | [`dependabot.yml`](.github/dependabot.yml) | ogni lunedì |

I risultati del controllo settimanale finiscono in **una** issue con etichetta `dependency-check`, aggiornata ogni
settimana e chiusa da sola quando torna tutto pulito. La issue cita `@GeneralKoski`, così la vede anche lui.
Il fallimento del workflow manda una mail a chi ha toccato per ultimo il `cron`.

Flusso di una correzione: si accetta la PR di Dependabot (o si corregge a mano) su `main`, la CI è verde, la PR
all'originale si aggiorna da sola, poi si rilascia su Welintouch (sotto).

## Rilascio su Welintouch

L'immagine non si pubblica da qui. Dal repo `welintouch_amedea`:

```bash
devs/scripts/push-koskidex.sh <percorso di questo repo>   # build da questo commit → ECR → compose → server
```

Lo script rifiuta una copia con modifiche non committate e avvisa se il commit non è ancora su `origin`. Usa come tag
il tag git, se c'è, altrimenti lo SHA corto.
Dettagli in `devs/docs/PROD.md` §5bis di quel repo.

## Comandi

```bash
make check                                   # go vet + go test (serve git nel PATH per internal/eval)
cd web && npm ci && npm test && npm run build
python3 .github/scripts/eol_check.py go.mod Dockerfile web/Dockerfile
```

Senza Go in locale: `docker run --rm -v "$PWD":/src -w /src golang:1.27-alpine sh -c 'apk add -q git && git config --global --add safe.directory /src && go test $(go list ./... | grep -v /node_modules/)'`.

## Convenzioni

- Identificatori in inglese. Commenti in inglese, come il resto del codice.
- Commit Conventional Commits in inglese, come nell'originale.
