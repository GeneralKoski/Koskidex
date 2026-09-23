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

Da compilare a fix applicato.
