# Diario delle modifiche al ranking

Una voce per ogni modifica che cambia l'ordine dei risultati, la più recente in
cima. Ogni voce si scrive in due tempi: la sezione **Prima** si committa *prima*
di lanciare la valutazione, la sezione **Dopo** si aggiunge a misura fatta.

Non è contabilità. È quello che a giugno permette di scrivere perché un
parametro è stato scelto così, ed è la risposta alla domanda che in discussione
arriva: le metriche sono state scelte dopo aver visto i risultati? Un'ipotesi
scritta prima e smentita dai numeri vale più di un risultato pulito senza storia.

Regole, da `piano-autunno-2026.md`:

1. Ogni modifica che cambia l'ordine sta **dietro un campo di `Settings`**, con
   il comportamento attuale come default.
2. `TestBaselineRankingIsFrozen` deve continuare a passare a default, **senza
   modifiche al test**.
3. Ogni modifica ha la sua voce qui.

---

## 2026-09-23 - Ordinamento deterministico dei pari merito

**Flag:** nessuno. È l'eccezione alla regola 1, motivata sotto.
**Tocca:** `internal/engine/ranker.go`, criterio finale di `sort.Slice` in `SearchScored`

### Il problema

Scrivendo il baseline (Task A3) è saltato fuori che sulla query `2026/0173` i
documenti `d1` e `d3` escono con punteggio identico (48) e tutti i tiebreaker
identici: 2 parole trovate, 0 refusi, 2 match esatti. I risultati si raccolgono
da una `map` e si ordinano con `sort.Slice`, che **non è stabile**. Su 30
esecuzioni: 26 hanno dato `[d1 d3]`, 4 hanno dato `[d3 d1]`.

Stesso codice, stessi dati, ordine diverso.

### Perché senza flag

Il flag esiste per tenere eseguibile il comportamento di partenza. Qui il
comportamento di partenza **è casuale**, e una cosa casuale non è un baseline
che valga la pena preservare: non si può confrontare niente contro una monetina.

E non stiamo esprimendo un'opinione su quale dei due documenti sia più
pertinente. Il motore dice già che sono equivalenti, e continuerà a dirlo: stiamo
solo scegliendo una delle due facce e fermandola. Il criterio è l'id crescente,
che è arbitrario di proposito, perché qualunque criterio "sensato" sarebbe una
modifica al ranking travestita.

Questa è l'unica eccezione prevista alla regola del flag. Ogni modifica
successiva che cambia l'ordine di documenti con punteggi **diversi** è una
modifica al ranking e il flag lo vuole.

### Prima di misurare

Cosa mi aspetto, scritto prima di lanciare:

1. `d1` e `d3` si fermano su `[d1 d3]`, perché `d1 < d3` come stringa. È anche
   l'ordine che usciva 26 volte su 30, quindi il test congelato non dovrebbe
   nemmeno sembrare cambiato.
2. **Nessuno degli altri casi congelati cambia.** Tutti gli altri hanno punteggi
   distinti (48/24, 24/12, 18/9, 44/24), quindi il nuovo criterio non viene mai
   raggiunto. Se anche uno solo cambia, ho sbagliato qualcosa: non è una
   sorpresa interessante, è un bug.
3. Venti esecuzioni consecutive danno sempre lo stesso ordine.

Se la 2 fosse smentita, la modifica va annullata e ricontrollata, non sanata.

### Dopo

Fix: un ultimo criterio in `sort.Slice`, `results[i].DocID < results[j].DocID`.
Sette righe, di cui cinque di commento.

**Tutte e tre le predizioni confermate.**

1. L'ordine si è fermato su `[d1 d3]`, come previsto. **40 esecuzioni su 40.**
2. **Nessun altro caso congelato è cambiato.** Verificato nel modo giusto, cioè
   lanciando `TestBaselineRankingIsFrozen` e `TestBaselineHybridIsFrozen`
   **senza toccarli**, 20 volte: sempre verdi. Solo dopo ho aggiunto la classe
   `identificatore` fra quelle congelate.
3. Suite completa verde: 86 test, 0 falliti.

**Nessuna sorpresa.** È il risultato noioso che ci si aspetta da un fix di
determinismo, ed è quello giusto: se fosse cambiato qualcos'altro avrebbe voluto
dire che il criterio nuovo veniva raggiunto in casi dove non doveva.

### Una cosa che avevo sbagliato

Quando avevo scritto `TestBaselineIdentifierTieIsNotDeterministic` avevo detto
che sarebbe "fallito rumorosamente" una volta risolto il pareggio. **Non è
vero**, e me ne sono accorto applicando il fix: quel test verificava la parità
dei *punteggi*, non l'ordine, e i punteggi sono rimasti identici (48 entrambi).
Sarebbe rimasto verde, lasciando credere che il problema fosse ancora aperto.

L'ho sostituito con `TestBaselineIdentifierTieBreaksByDocID`, che dice la cosa
giusta: i punteggi sono ancora in parità, l'ordine ora lo decide il DocID, e il
test fallirà quando **BM25** separerà davvero i due documenti. Quello sì che è
il momento in cui deve rompersi.

### Cosa resta aperto, ed è il punto

Il pareggio **non è stato risolto, è stato solo reso stabile.** `d1` e `d3`
fanno ancora 48 entrambi perché senza IDF un token condiviso da tutti e due pesa
quanto uno che li separerebbe. È esattamente il primo difetto, e questa query è
il caso di prova pronto per la Fase 1: quando arriva BM25, i due punteggi devono
divergere, e di quanto è un numero da mettere in tesi.
