# Baseline di partenza

Stato del repository nel momento in cui inizia il lavoro di tesi. Serve a sapere,
fra sei mesi, con quale toolchain e da quale codice sono nati i primi numeri.

Task A0 di `piano-autunno-2026.md` (repository `Universita-Martin`,
`Magistrale/Tesi/`).

## 23 settembre 2026

**Commit:** `0fd8190` - *docs: remove unsupported performance claims from README*
**Branch:** `main`, working tree pulito

### Toolchain

```
go version go1.27.1 darwin/arm64
```

`go.mod` dichiara `go 1.25`, che è il minimo. Homebrew ha installato la 1.27.1,
che è quella con cui sono stati prodotti tutti i risultati da qui in avanti.
**Se in futuro i numeri non tornano, la versione del compilatore è la prima cosa
da confrontare**, prima del codice.

### Esito dei test

```
?   github.com/GeneralKoski/Koskidex                    [no test files]
ok  github.com/GeneralKoski/Koskidex/internal/engine    0.984s
?   github.com/GeneralKoski/Koskidex/internal/manager   [no test files]
?   github.com/GeneralKoski/Koskidex/internal/server    [no test files]
ok  github.com/GeneralKoski/Koskidex/internal/storage   6.573s
?   github.com/GeneralKoski/Koskidex/scripts            [no test files]
```

**76 test eseguiti, 0 falliti.** `go vet ./...` non riporta niente.

Il verde di partenza è reale: ogni confronto successivo parte da qui.

### Cosa non è coperto da test

`internal/manager` e `internal/server` non hanno test propri. Sono coperti solo
di rimbalzo dai test di `tests/`, che passano dall'API. Non è un problema per la
tesi, che lavora su `internal/engine`, ma va saputo prima di dare per scontato
che una modifica al server sia sicura.

### Una stranezza da tenere d'occhio

`go test ./...` compila anche
`web/node_modules/flatted/golang/pkg/flatted`: dentro `node_modules` c'è un
package Go, e il pattern `./...` se lo prende. Oggi non dà fastidio perché non ha
test e non ha problemi di vet, ma è rumore che entra nei comandi, e se un domani
quel package si rompe fa fallire un controllo che non c'entra niente con
Koskidex.

## Regole da qui in poi

Dal Task A3 in poi vale la disciplina scritta nel piano:

1. Ogni modifica che cambia l'ordine dei risultati sta **dietro un campo di
   `Settings`**, con il comportamento attuale come default.
2. `TestBaselineRankingIsFrozen` deve continuare a passare a default, **senza
   modifiche al test**.
3. Ogni modifica al ranking ha la sua voce in `eval/DIARIO.md`, con la sezione
   *Prima di misurare* committata **prima** di lanciare la valutazione.
