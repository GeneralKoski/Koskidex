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

## 2026-09-23 - BM25 al posto del punteggio euristico (difetto 1)

**Flag:** `Settings.ScoringMode`, `"legacy"` di default, `"bm25"` per il nuovo.
Vuoto trattato come `"legacy"`, stessa ragione del `RetrievalMode`: le settings
sono persistite.
**Parametri:** `k1=1.2`, `b=0.75`, i default di Lucene. **Non calibrati**: la
calibrazione è Fase 3, qui si misura la formula standard.

### Cosa sostituisce

Oggi il punteggio di un termine è `(10 - refusi + 2*esatti) * peso_campo`.
Nessun IDF, nessuna frequenza di termine, nessuna normalizzazione sulla
lunghezza. Un termine rarissimo pesa quanto uno comunissimo.

### Cosa manca nell'indice

Tre cose che BM25 richiede e che oggi non esistono:

- **df(t)**, in quanti documenti compare un termine. Ricavabile contando i
  `DocID` distinti nei postings, ma è O(postings) per termine comune: va
  mantenuta durante l'indicizzazione.
- **|D|**, la lunghezza del documento in token. `docToTerms` non serve, è
  **deduplicato**: contiene i termini distinti, non le occorrenze.
- **tf(t,D)**. Il campo `Posting.TF` esiste con scritto *"calculated later"* e
  non è mai stato riempito; `addDocumentLocked` calcola perfino un `termCounts`
  e poi lo butta via.

### Prima di misurare

1. **SciFact sale da 0,4936 verso il riferimento 0,6789.** Dichiaro la fascia:
   **fra 0,55 e 0,70**. Se restasse sotto 0,55, l'IDF conta meno di quanto la
   tesi assume e il capitolo va ripensato.
2. **NFCorpus sale da 0,2246 verso 0,3218.** Fascia: **fra 0,25 e 0,33**.
3. **Recall@100 resta praticamente fermo, entro ±0,02 su entrambe.** Questa è la
   più importante e non è una previsione, è una **guardia**: BM25 cambia come si
   ordina, non cosa si recupera. Se il recall si muove, ho cambiato il recupero
   per sbaglio e il numero sul nDCG non vale niente.
4. **`TestBaselineIdentifierTieBreaksByDocID` fallisce.** Oggi `d1` e `d3` fanno
   48 entrambi sulla query `2026/0173`. Con la normalizzazione sulla lunghezza i
   due si separano: è il test scritto apposta per rompersi qui, ed è il momento
   in cui deve rompersi.
5. **`TestBaselineRankingIsFrozen` passa a default, senza modifiche al test.**

La 3 è quella che può salvarmi da un risultato falso. La 4 è quella che chiude
il cerchio aperto il 23 settembre sul pareggio dei pari merito.

### Dopo

| | legacy | any | **bm25** | riferimento |
|---|---|---|---|---|
| SciFact nDCG@10 | 0,0246 | 0,4936 | **0,6197** | 0,6789 |
| SciFact Recall@100 | 0,0242 | 0,7571 | **0,8746** | - |
| SciFact MRR@10 | 0,0267 | 0,4641 | **0,5868** | - |
| NFCorpus nDCG@10 | 0,1659 | 0,2246 | **0,2810** | 0,3218 |
| NFCorpus Recall@100 | 0,0967 | 0,1898 | **0,2280** | - |
| NFCorpus MRR@10 | 0,2644 | 0,3774 | **0,4710** | - |

Query a vuoto invariate rispetto al disgiuntivo: 0 su SciFact, 24 su 323 su
NFCorpus. Il punteggio non cambia chi viene trovato, e infatti non lo cambia.

Previsioni:

1. **Presa.** SciFact 0,6197, dentro la fascia 0,55-0,70, il 91% del
   riferimento. Il divario che resta è quello che SOURCE.md aveva previsto:
   tokenizer diverso e nessuno stemmer.
2. **Presa.** NFCorpus 0,2810, dentro la fascia 0,25-0,33, l'87% del
   riferimento.
3. **Smentita.** Il recall si è mosso, e parecchio: +0,117 su SciFact, +0,038
   su NFCorpus, contro un ±0,02 dichiarato. Sotto sta l'indagine, perché una
   guardia che scatta o vuol dire che il risultato è falso o vuol dire che la
   guardia era scritta male.
4. **Mal posta da me.** `TestBaselineIdentifierTieBreaksByDocID` gira a
   settings di default, e il default è `legacy`: non poteva fallire. Il
   contenuto della previsione era però giusto, e sta in un test nuovo,
   `TestBM25BreaksTheIdentifierTieTheLegacyScorerCouldNot`: su `2026/0173` il
   legacy dà 48 a pari merito a `d1` e `d3`, BM25 li separa e mette davanti
   `d3` (2,850 contro 2,328), il documento più corto il cui titolo è
   sostanzialmente l'identificativo. È la normalizzazione sulla lunghezza che
   fa esattamente il suo mestiere, e chiude il pareggio congelato in mattinata.
5. **Presa.** `TestBaselineRankingIsFrozen` verde a default, tre esecuzioni
   consecutive, test non toccato.

#### Perché la guardia è scattata

L'ipotesi benevola era che `recall@100` sia sensibile all'ordine quando i
documenti che corrispondono sono più di 100. Andava verificata, non assunta,
perché l'ipotesi malevola - ho cambiato il recupero per sbaglio - produce lo
stesso sintomo.

Tre riscontri, in ordine di forza crescente:

- **Il taglio morde.** Su SciFact tutte e 300 le query arrivano al tetto dei
  100 risultati; su NFCorpus 202 su 323.
- **Sotto il taglio non si muove niente.** Le 121 query di NFCorpus che
  restituiscono meno di 100 documenti hanno recall **identico al dodicesimo
  decimale** fra `any` e `bm25`. Se il recupero fosse cambiato, sarebbero
  cambiate loro per prime: lì dentro l'ordine non può influire, ci stanno tutti.
  Le 202 sopra il taglio sono quelle che si muovono, 127 di esse.
- **Il codice non lo consente.** In `SearchScored` il filtro che decide chi
  resta legge solo `WordsMatched`; l'unica altra cancellazione sono i termini
  in NOT. Il punteggio non elimina mai nessuno. L'insieme dei candidati è
  indipendente da `ScoringMode` per costruzione.

Quindi: il recupero è invariato, la guardia era scritta male. `recall@k` non
misura "quanto ho recuperato", misura "quanti rilevanti sono entrati nei primi
k", e quando i candidati sono molti più di k è una **metrica di ordinamento**
come il nDCG. Su SciFact in modalità disgiuntiva la query mediana pesca 5182
documenti su 5183: dentro il taglio ci entra l'1,9% dei candidati. Aspettarsi
che il recall stesse fermo mentre l'ordinamento migliorava era una pretesa
incoerente.

Il +0,117 di SciFact, letto bene, non è un allarme: è il secondo risultato di
questa voce. BM25 non trova più documenti rilevanti, li **porta dentro i primi
cento** - da 75,7 a 87,5 su cento rilevanti esistenti.

#### La guardia nella forma corretta

Una guardia che si può controllare a mano una volta non è una guardia. Il
numero che serve - quanti documenti corrispondono in tutto, prima del taglio -
veniva buttato via dal troncamento, quindi ora il runner lo registra:
`Searcher.Search` restituisce anche il totale e ogni query nel file dei
risultati ha il campo `candidates`.

> **Guardia, d'ora in avanti:** una modifica al *punteggio* non deve cambiare
> `candidates` per nessuna query. Una modifica al *recupero* lo cambierà, ed è
> lì che va guardato. `recall@k` non è una guardia sul recupero quando
> `candidates > k`.

Applicata a questa voce: **0 query su 300 (SciFact) e 0 su 323 (NFCorpus)**
cambiano il numero di candidati fra `any` e `bm25`. Il recupero è intatto,
misurato e non argomentato.

I sei file dei risultati sono stati rigenerati con il campo nuovo: tutte le
metriche, per query e in media, sono venute identiche. Riproducibilità
verificata di sbieco.

#### Cosa resta sul tavolo

Il 9-13% che manca al riferimento non è rumore. Le due cause note - nessuno
stemmer, tokenizer diverso - sono già scritte in SOURCE.md e sono la materia
della Fase 2. Le 24 query vuote di NFCorpus restano tali: 9 su 10 dei loro
termini non esistono nel corpus in nessuna forma, e `leeks`→`leek` è l'unico
caso che uno stemmer salverebbe.

---

## 2026-09-23 - Recupero disgiuntivo (difetto 0)

**Flag:** `Settings.RetrievalMode`, `"all"` di default (comportamento attuale),
`"any"` per il disgiuntivo. Stringa vuota trattata come `"all"`: le settings
vengono persistite su disco, e un indice salvato prima di oggi non ha il campo.
**Tocca:** `internal/engine/inverted.go` (Settings), `ranker.go` (il filtro
`requiredMatches`)

### Perché

Misurato sopra: 290 query su 300 a vuoto su SciFact. Il filtro attuale tiene
solo i documenti che contengono **tutti** i termini della query.

### Prima di misurare

1. **Le query a vuoto crollano quasi a zero** su entrambe le collezioni. È
   quasi una tautologia, serve solo a confermare che il flag è collegato.
2. **SciFact: nDCG@10 sale molto sopra 0,0246 ma resta molto sotto 0,6789.**
   Tiro un numero: **fra 0,10 e 0,45**. Senza IDF un documento che contiene
   dieci termini comuni batte quello che contiene l'unico termine raro che
   conta, ed è esattamente il difetto 1. Se arrivasse vicino al riferimento,
   vorrebbe dire che l'IDF conta poco e la Fase 1 varrebbe meno.
3. **Recall@100 su SciFact sale tantissimo, sopra 0,50.** È l'effetto
   principale: il disgiuntivo è un cambiamento di recupero, non di ordinamento.
   Se il recall non salisse, il flag non starebbe facendo quello che credo.
4. **Su NFCorpus il nDCG potrebbe PEGGIORARE, e non sarebbe un errore.** Le
   query sono da 2-3 parole e lì l'AND funzionava da filtro di precisione: il
   50% di query a vuoto è il prezzo, ma quelle che rispondevano rispondevano
   bene (MRR@10 a 0,2644, più alto del nDCG). In disgiuntivo entrano molti
   documenti mediocri. Mi aspetto **recall su, nDCG incerto**.
5. `TestBaselineRankingIsFrozen` passa a default **senza modifiche al test**.

La 4 è quella su cui non so la risposta, ed è la più interessante: se il nDCG
peggiora su query corte e migliora su query lunghe, la conclusione di tesi non è
"il disgiuntivo è meglio" ma "la modalità giusta dipende dalla lunghezza della
query", che è un risultato più forte e si lega alla Fase 3.

### Dopo

File: `eval/results/{scifact,nfcorpus}-any.json`.

| SciFact | legacy (`all`) | **`any`** | riferimento BM25 |
|---|---|---|---|
| nDCG@10 | 0,0246 | **0,4936** | 0,6789 |
| Recall@100 | 0,0242 | **0,7571** | - |
| MRR@10 | 0,0267 | **0,4641** | - |
| Query a vuoto | 290 su 300 | **0** | - |

| NFCorpus | legacy (`all`) | **`any`** | riferimento BM25 |
|---|---|---|---|
| nDCG@10 | 0,1659 | **0,2246** | 0,3218 |
| Recall@100 | 0,0967 | **0,1898** | - |
| MRR@10 | 0,2644 | **0,3774** | - |
| Query a vuoto | 160 su 323 | **24 su 323** | - |

SciFact fa **20 volte** il nDCG di prima e **31 volte** il recall. Entrambe le
collezioni arrivano intorno al **70-73% del riferimento BM25**, partendo dal 4%
e dal 52%.

### Cosa avevo previsto giusto e cosa no

Tre su cinque.

- **1, query a vuoto crollano:** giusto. Zero su SciFact, 24 su NFCorpus.
- **2, SciFact fra 0,10 e 0,45:** **sbagliato**, ha fatto 0,4936, sopra la
  fascia che avevo dichiarato. Avevo sottostimato.
- **3, recall SciFact sopra 0,50:** giusto, 0,7571.
- **4, NFCorpus poteva peggiorare:** **sbagliato**. È migliorato su tutte e tre
  le metriche. Ed è l'errore più istruttivo dei due.
- **5, baseline congelato verde a default senza toccare il test:** giusto.

### Perché mi sbagliavo sulla 4, che è la cosa da mettere in tesi

Pensavo che l'AND facesse da filtro di precisione sulle query corte, e che
toglierlo avrebbe fatto entrare documenti mediocri peggiorando l'ordine. Non
succede, e il motivo è che **il filtro congiuntivo era ridondante rispetto al
punteggio che c'era già**: chi trova più termini prende più punti, quindi i
documenti con tutti i termini restano in testa da soli. Il filtro non li
promuoveva, si limitava a cancellare tutto il resto, compresi i documenti che
sarebbero stati in seconda o terza posizione a buon diritto.

Detto altrimenti: l'AND non aggiungeva precisione, toglieva recall. È una
conclusione più netta di quella che mi aspettavo e vale un paragrafo.

### Il residuo: 24 query NFCorpus ancora vuote

Sono tutte da un termine solo e raro: `eggnog`, `halibut`, `okra`, `Fosamax`,
`Mevacor`, `deafness`, `mesquite`, `myelopathy`, `Peoria`, `leeks`.

Controllato contro il vocabolario del corpus: **9 su 10 non compaiono proprio**,
in nessuna forma. Sono buchi veri di vocabolario, non difetti del motore, e
nemmeno Lucene troverebbe niente. L'unica recuperabile è `leeks`, perché il
corpus contiene `leek`: quella la prenderebbe uno stemmer, che Koskidex non ha.

Non è un problema aperto, è una nota: parte della distanza residua dal
riferimento è stemming, non ranking.

### Dove resta il divario

SciFact 0,4936 contro 0,6789. Il pezzo grosso che manca è **l'IDF**, cioè il
difetto 1: senza, un documento che contiene dieci termini comuni batte quello
che contiene l'unico termine raro che decide la query. È esattamente quello che
la Fase 1 deve misurare, e ora ha un baseline sensato contro cui farlo invece di
uno 0,0246 che non voleva dire niente.

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
