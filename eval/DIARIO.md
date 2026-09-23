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

## 2026-09-23 - Prima misura del baseline legacy su C1

**Flag:** nessuna modifica al motore. È la prima misura, non un cambiamento.
**Serve a:** collaudare l'impianto di valutazione e dare al legacy un numero
contro cui BM25 dovrà staccare.

### Prima di misurare

Configurazione: campo unico `title + text`, peso 1.0, refusi spenti
(`fuzziness="0"` e `TypoTolerance.Enabled=false`), profondità 100.

Cosa mi aspetto, scritto prima di lanciare:

1. **SciFact starà nettamente sotto 0,6789**, il riferimento BM25. Il punteggio
   legacy è `(10 - refusi + 2*esatti) * peso_campo` senza nessun IDF, quindi su
   query di più parole premia chi contiene molti termini comuni. Tiro un numero
   per non barare a posteriori: **fra 0,30 e 0,55**.
2. **NFCorpus starà sotto 0,3218**, ma il distacco sarà minore in valore
   assoluto perché il riferimento stesso è basso.
3. **Recall@100 sarà molto più alto di nDCG@10** su entrambe: il legacy trova i
   documenti, il problema è come li ordina. Se anche il recall fosse basso, il
   problema non sarebbe il ranking ma il matching, e cambierebbe la tesi.
4. Nessuna query saltata su SciFact: tutti i 339 giudizi hanno rilevanza 1.

La 3 è la più informativa. Se fosse smentita, i tre capitoli previsti
sarebbero sul difetto sbagliato.

### Dopo

File: `eval/results/scifact-legacy.json`, `eval/results/nfcorpus-legacy.json`.

| | SciFact | NFCorpus |
|---|---|---|
| **nDCG@10** | **0,0246** | **0,1659** |
| Recall@100 | 0,0242 | 0,0967 |
| MRR@10 | 0,0267 | 0,2644 |
| **Query a vuoto** | **290 su 300 (97%)** | **160 su 323 (50%)** |
| Riferimento BM25 | 0,6789 | 0,3218 |

**La predizione 3 e' sbagliata, e sbagliata nel modo piu' utile possibile.**

Avevo scritto che il recall sarebbe stato molto piu' alto del nDCG, perche' "il
legacy trova i documenti, il problema e' come li ordina". Su SciFact recall@100
fa 0,0242 contro un nDCG@10 di 0,0246: sono lo stesso numero. Il motore non
ordina male. **Non trova.**

Avevo anche scritto cosa avrebbe significato: *"se anche il recall fosse basso,
il problema non sarebbe il ranking ma il matching, e cambierebbe la tesi"*.
Ecco, e' successo.

### La causa, verificata nel codice

`ParseQuery` in `internal/engine/ranker.go:33` mette **ogni parola** in
`MustTerms`, a meno che non si scriva `OR` a mano. Il recupero e' puramente
**congiuntivo**: un documento deve contenere *tutti* i termini della query.
Lucene, e quindi il riferimento BM25, e' disgiuntivo con punteggio: un documento
che ne contiene 8 su 12 si piazza bene.

Su una query SciFact da 12 parole nessun documento le contiene tutte, quindi
zero risultati. Provato su una query sola: intera 0 risultati, il primo termine
da solo 7, i primi due 4.

### La prova che e' la lunghezza della query, non altro

Stessa collezione, stesso indice, stessa configurazione:

| | query con risultati | query a vuoto |
|---|---|---|
| **NFCorpus** | 163 query, **media 2,0 parole** | 160 query, **media 4,7 parole** |
| **SciFact** | 10 query, media 8,7 parole | 290 query, media 12,6 parole |

La lunghezza media delle query di test e' 3,3 parole su NFCorpus e 12,5 su
SciFact, ed e' esattamente la differenza fra il 50% e il 97% di query a vuoto.
E' una relazione dose-effetto, non una coincidenza.

### Cosa cambia per la tesi

**Non e' un bug, ed e' importante dirlo in questi termini.** L'AND e' una scelta
ragionevole per il caso d'uso per cui Koskidex era nato: la barra di ricerca di
un e-commerce, due o tre parole, dove restituire chi le contiene tutte e'
giusto. E' il tipo di query a rivelarlo.

Ma cambia due cose:

1. **C'e' un quarto difetto, e viene prima degli altri tre.** Con il 97% di
   query a vuoto, nessun miglioramento del punteggio puo' fare niente: BM25
   applicato a un recupero congiuntivo riordinerebbe il nulla. La Fase 1 come
   scritta misurerebbe zero.
2. **E' il difetto piu' rilevante per il documentale**, che e' il banco di prova
   della tesi. Li' le query arrivano da persone che scrivono a lingua naturale, e
   sempre piu' spesso da un LLM che riformula: query lunghe, non da due parole.
   E' esattamente il regime in cui il motore collassa.

### Conseguenza operativa

L'ordine delle fasi cambia: **il recupero disgiuntivo va prima di BM25.** E'
anche una buona notizia per la tesi, perche' e' il capitolo con l'effetto
misurabile piu' grande: si parte da 0,0246 e c'e' tutto lo spazio del mondo.

Da riportare in `piano-autunno-2026.md`.

### Una nota sull'impianto

A parte la sorpresa, la Fase 0 ha fatto il suo lavoro. NFCorpus a 0,1659 contro
un riferimento di 0,3218 e' un numero plausibile per un motore senza IDF: se
l'impianto avesse avuto un bug grosso avremmo visto zero anche li'. E il
contatore delle query a vuoto, che non era previsto, e' stato aggiunto al runner
proprio perche' e' il numero che ha rivelato tutto: una media non distingue un
motore che ordina male da uno che non trova, e i due vogliono fix opposti.

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
