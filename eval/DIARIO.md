# Diario delle modifiche al ranking

Una voce per ogni modifica che cambia l'ordine dei risultati, la più recente in
cima. Ogni voce si scrive in due tempi: la sezione **Prima** si committa *prima*
di lanciare la valutazione, la sezione **Dopo** si aggiunge a misura fatta.

Non è contabilità. È quello che a giugno permette di scrivere perché un
parametro è stato scelto così, ed è la risposta alla domanda che in discussione
arriva: le metriche sono state scelte dopo aver visto i risultati? Un'ipotesi
scritta prima e smentita dai numeri vale più di un risultato pulito senza storia.

Regole, oggi in `Magistrale/Tesi/piano.md` del repository `Universita-Martin`:

1. Ogni modifica che cambia l'ordine sta **dietro un campo di `Settings`**, con
   il comportamento attuale come default.
2. `TestBaselineRankingIsFrozen` deve continuare a passare a default, **senza
   modifiche al test**.
3. Ogni modifica ha la sua voce qui.

I file `eval/results/<nome>.json` citati nelle voci del 23/09/2026 non sono più
nel repository dal 25/09/2026. L'ultima versione di ciascuno è identica, byte
per byte, a `risultati/koskidex-beir/2026-09-23T155*_<nome>.json` nell'archivio
della tesi, che è il posto dove stanno le esecuzioni che contano.

---

## 2026-09-25 - BM25: le espansioni pesate con la frequenza mescolata

**Flag:** `Settings.BM25Expansion`, vuoto = comportamento di oggi; `"blended"`
= la correzione. Conta solo con `ScoringMode = "bm25"`.

**Perché.** Sulle 300 known-item `<numero> <comune>` BM25 fa MRR@10 0,709
contro 0,975 dell'euristico. In 61 dei 115 fallimenti il primo classificato
non contiene nessun termine della query: combacia solo per prefisso, `190` con
`1900129` (`risultati/esperimenti/2026-09-25_bm25-numeri/` nel repository
della tesi). `findDocsForToken` attribuisce il match al termine trovato, e
`bm25Locked` usa la frequenza documentale di quel termine: un'espansione rara
ha un IDF altissimo e batte il termine esatto, che è più comune. L'euristico non
ci cade perché dà `+2` ai match esatti.

### Cosa cambia

Con `"blended"`, tutte le espansioni di un termine della query (prefisso e
refusi, termine esatto compreso) usano la **frequenza documentale più alta** fra
i termini che quel termine della query ha trovato nell'indice. È quello che fa
Lucene per le ricerche fuzzy (`TopTermsBlendedFreqScoringRewrite`): un
termine espanso non può pesare più del più comune dei suoi fratelli, quindi un
codice raro che comincia con `190` pesa quanto `190`, non di più. Il TF resta
quello del termine trovato nell'atto, come oggi.

Non si tocca l'euristico, non si tocca il recupero: cambiano solo i punteggi,
quindi gli insiemi restano identici.

### Prima di misurare

Numeri di partenza: known-item MRR@10 0,709 (BM25 di serie), 0,854 con la
ricerca per prefisso spenta, 0,975 l'euristico. SciFact 0,6197 e NFCorpus
0,2810 di nDCG@10 con BM25 senza analisi, 0,6641 e 0,3182 con le stopword.

1. **Known-item: MRR@10 sopra 0,85**, almeno quanto spegnere il prefisso. Con
   la frequenza mescolata l'espansione rara pesa quanto il numero esatto, e
   l'atto giusto, che ha anche le parole del comune, torna davanti. Può fare
   meglio di spegnere il prefisso, perché il prefisso resta e continua a
   trovare le parole scritte a metà.
2. **Known-item: resta sotto l'euristico.** Il divario che il prefisso non
   spiega (da 0,854 a 0,975) non è toccato da questa modifica.
3. **SciFact e NFCorpus cambiano meno di 0,01 di nDCG@10**, in un verso o
   nell'altro. Lì i refusi sono spenti ma il prefisso no, quindi la modifica
   agisce anche lì; ma sulle query in inglese in linguaggio naturale le
   espansioni per prefisso sono rare e pesano poco sul totale.
4. **Gli insiemi non cambiano**: stessi `candidates` per query in ogni file.

Da non fare: scegliere fra questa e altre varianti guardando le 300 query. Se
la 1 è smentita, la variante successiva si scrive qui con la sua ipotesi, prima
di misurarla.

---

## 2026-09-23 - Analisi lessicale: stopword e stemmer (Task E1)

**Flag:** `Settings.Analyzer`, vuoto = comportamento di oggi. Terzo campo con
questa semantica dopo `RetrievalMode` e `ScoringMode`, stessa ragione: le
settings sono persistite e un indice salvato prima non deve cambiare
comportamento da solo.

**Non è uno dei quattro difetti.** È ampliamento di perimetro, deciso
esplicitamente: serve a capire quanto del divario dai riferimenti sia mio e
quanto sia dell'analizzatore. Un BM25 corretto che resta al 90% del riferimento
è difendibile solo se so dire perché.

### Cosa cambia

`Tokenize` (`tokenizer.go:27`) oggi fa tre cose: minuscolo, rimozione degli
accenti, e spezza su tutto ciò che non è lettera o numero. `DefaultSettings`
(`inverted.go:281`) inizializza `StopWords` a una mappa **vuota**. I riferimenti
pubblicati girano su Lucene con `EnglishAnalyzer`: stemming Porter più la sua
lista di stopword.

La firma di `Tokenize` passa a prendere le `Settings` intere invece delle sole
stopword. Non è estetica: i punti di chiamata sono sette, fra indicizzazione,
`ParseQuery` e ricerca, e se l'analizzatore arrivasse solo ad alcuni l'indice
conterrebbe termini che la query non produce più. Il sintomo sarebbe "non trova
niente", che è il più difficile da ricondurre alla causa. Con le `Settings`
nella firma, un punto di chiamata dimenticato non compila.

### Prima di misurare

Numeri di partenza: SciFact 0,6197 e NFCorpus 0,2810 di nDCG@10, contro
riferimenti 0,6789 e 0,3218. Query a vuoto: 0 su SciFact, 24 su 323 su NFCorpus.

1. **SciFact finisce fra 0,63 e 0,70.** Se restasse sotto 0,63 vuol dire che il
   divario non era l'analizzatore e la spiegazione scritta in `SOURCE.md` prima
   di misurare era sbagliata.
2. **NFCorpus finisce fra 0,29 e 0,34.**
3. **Lo stemmer porta molto più delle stopword, e le stopword da sole quasi
   niente** - diciamo sotto +0,01 su entrambe. È la previsione che mi aspetto
   contestata, quindi la scrivo: con BM25 l'IDF già schiaccia i termini
   frequentissimi, quindi togliere le stopword toglie lavoro all'indice ma non
   sposta l'ordinamento. Sotto il punteggio euristico sarebbe stato diverso,
   perché lì un termine comune pesava quanto uno raro. Le tre configurazioni si
   misurano separate apposta: solo stopword, solo stemmer, tutte e due.
4. **`candidates` aumenta, ed è giusto così.** Questa è l'opposto della guardia
   del BM25 di stamattina, e la scrivo per non ripetere l'errore: lì il
   punteggio non poteva cambiare il recupero, qui lo stemming lo cambia per
   definizione, perché fonde varianti dello stesso termine. Un `candidates`
   fermo significherebbe che l'analizzatore non è arrivato all'indicizzazione.
5. **La guardia vera è un'altra: le query a vuoto non devono aumentare.** Lo
   stemming può solo rendere il confronto più permissivo, ma la rimozione delle
   stopword può svuotare una query fatta di sole parole comuni. Se il numero
   sale, ho tolto troppo.
6. **Delle 24 query vuote di NFCorpus se ne recupera una, non ventiquattro.**
   Dall'indagine del 23/09: su dieci termini assenti dal corpus, nove non ci
   sono in nessuna forma e solo `leeks` → `leek` è una mancanza di stemming.
   Quindi il numero atteso è 23, non 0. Se scendesse molto più in basso, la mia
   indagine di stamattina era fatta male.
7. **`TestBaselineRankingIsFrozen` passa a default, senza modifiche al test.**

La 3 e la 6 sono quelle che possono smentirmi in modo interessante. La 5 è la
guardia, e stavolta è scritta su una quantità che la modifica non può muovere
per costruzione.

### Dopo

Nel codice il campo si chiama `Settings.Stemmer` (`json:"stemmer"`, commit
`00b50ec`), non `Analyzer` come nella sezione qui sopra, scritta prima.

**SciFact** (riferimento 0,6789)

| analisi | nDCG@10 | delta | Recall@100 | a vuoto | candidati medi |
|---|---|---|---|---|---|
| nessuna | 0,6197 | - | 0,8746 | 0/300 | 4833 |
| **solo stopword** | **0,6641** | **+0,0444** | 0,8792 | 0/300 | 2522 |
| solo stemmer | 0,6243 | +0,0046 | 0,8980 | 0/300 | 4908 |
| entrambi | 0,6585 | +0,0388 | **0,9103** | 0/300 | 3086 |

**NFCorpus** (riferimento 0,3218)

| analisi | nDCG@10 | delta | Recall@100 | a vuoto | candidati medi |
|---|---|---|---|---|---|
| nessuna | 0,2810 | - | 0,2280 | 24/323 | 1415 |
| solo stopword | 0,2900 | +0,0090 | 0,2346 | 24/323 | 600 |
| solo stemmer | 0,2861 | +0,0051 | 0,2373 | **15/323** | 1466 |
| **entrambi** | **0,2941** | **+0,0131** | **0,2435** | **15/323** | 696 |

Dal 91% al **97%** del riferimento su SciFact, dall'87% al **91%** su NFCorpus.

Previsioni: **1 presa, 2 presa, 3 demolita e ribaltata, 4 sbagliata,
5 tenuta, 6 sbagliata.** Tre su sei.

#### La 3, che è quella che vale

Avevo scritto: *lo stemmer porta molto più delle stopword, e le stopword da sole
quasi niente, sotto +0,01.* È esattamente il contrario. Su SciFact le stopword
danno **+0,0444** e lo stemmer **+0,0046**: le stopword fanno dieci volte tanto,
e lo stemmer da solo sta sotto il tetto che avevo messo alle stopword.

Il mio ragionamento era: con BM25 l'IDF schiaccia già i termini frequentissimi,
quindi toglierli non sposta l'ordinamento. Il ragionamento è giusto **sul
punteggio** ed è irrilevante, perché il guadagno non viene dal punteggio.

Verificato invece che dedotto. Su SciFact i candidati medi passano da 4833 a
2522: in modalità disgiuntiva ogni documento che contiene «the» è un candidato.
Separando le query per quanti candidati hanno perso:

| | delta nDCG medio |
|---|---|
| query con taglio sopra la mediana (n=150) | **+0,0523** |
| query con taglio sotto la mediana (n=150) | +0,0364 |
| query che non hanno perso **nessun** candidato (n=25) | **+0,0035** |

Le 25 query che non contengono nessuna stopword non guadagnano praticamente
niente. Il guadagno è **recupero, non punteggio**: togliere le stopword non
riordina meglio, impedisce a mezzo corpus di entrare.

Quel +0,0035 residuo sulle query che non dovrebbero cambiare affatto è un
effetto di secondo ordine che vale la pena notare: l'analizzatore accorcia i
documenti, quindi cambia `|D|` e `avgdl`, quindi cambia la normalizzazione BM25
**per tutti**, anche per chi non ha stopword nella query. Le statistiche di
collezione non sono neutre rispetto all'analisi.

#### Lo schema che si ripete, ed è il motivo per cui tengo questo diario

**Due volte nella stessa giornata ho ragionato sul punteggio dimenticando il
recupero.**

- Stamattina: *«BM25 cambia come si ordina, non cosa si recupera»*, scritto come
  guardia. È scattata, perché con i candidati molto più numerosi del taglio il
  recall è una metrica di ordinamento.
- Stasera: *«le stopword non spostano l'ordinamento perché l'IDF le schiaccia»*.
  Vero sul punteggio, e irrilevante, perché le stopword agiscono sul recupero.

È lo stesso punto cieco: tratto «cambia il punteggio» e «cambia cosa viene
recuperato» come separabili, e in modalità disgiuntiva non lo sono. Da qui in
avanti, prima di scrivere un'ipotesi: **questa modifica tocca chi entra, o solo
in che ordine?** E se tocca chi entra, il numero da guardare è `candidates`.

#### La 6, e un errore di stamattina

Avevo scritto che delle 24 query vuote di NFCorpus se ne sarebbe recuperata
**una**, `leeks` → `leek`, sulla base dell'indagine del mattino: «su dieci
termini assenti, nove non ci sono in nessuna forma». Ne sono state recuperate
**nove su 24**:

`deafness`, `leeks`, `pineapples`, `turnips`, `whiting`, `airport scanners`,
`antinutrients`, `bagels`, `canker sores`

Sono quasi tutti plurali il cui singolare nel corpus c'è eccome. L'indagine di
stamattina guardava un campione e ne ha tratto il caso generale: non era
sbagliata nel metodo, era sbagliata nella conclusione, e l'avevo scritta come se
fosse una misura. **Un campione di dieci su ventiquattro non è un censimento.**

#### La 4, sbagliata per come l'ho posta

Avevo scritto «`candidates` aumenta». Aumenta col solo stemmer (4833 → 4908),
**crolla** con le stopword (4833 → 2522). Avevo in testa lo stemmer e ho scritto
la previsione sulla modifica intera, che sono due stadi con effetti opposti sul
recupero. Se le avessi misurate insieme e basta, avrei visto un calo e concluso
che qualcosa era rotto.

Motivo in più per misurare gli stadi separati: **non per curiosità, per poter
interpretare il totale.**

#### La 5, la guardia: tenuta

Query a vuoto: SciFact 0 → 0, NFCorpus 24 → 24 con le stopword e 24 → 15 con lo
stemmer. Non sono salite da nessuna parte. Stavolta la guardia era scritta su
una quantità che la modifica poteva muovere solo in una direzione sbagliata, ed
è rimasta ferma.

#### Una cosa che non mi spiego, e la lascio aperta

Su SciFact la configurazione migliore è **solo stopword** (0,6641), non tutte e
due (0,6585). Aggiungere lo stemmer alle stopword **peggiora** il nDCG@10 di
0,0056, mentre alza il Recall@100 da 0,8792 a 0,9103. Su NFCorpus invece
aggiungerlo migliora entrambi.

Lettura plausibile: lo stemmer porta dentro più documenti rilevanti in
profondità (recall su), ma aggiunge anche rumore in testa (nDCG@10 giù). Resta
il fatto che il riferimento usa entrambi e arriva più in alto di tutte e quattro
le mie configurazioni, quindi la spiegazione non è completa. **Non la forzo:**
va misurata quando servirà, non raccontata adesso.

#### Cosa resta del divario

SciFact è al 97%, NFCorpus al 91%. Le due cause dichiarate in `SOURCE.md` prima
di misurare erano lo stemmer e il tokenizer: lo stemmer è fatto, e da solo ha
reso poco. Quello che resta è il tokenizer, e una differenza nota e non
corretta: Koskidex toglie gli accenti (`removeAccents`), `EnglishAnalyzer` di
Lucene no. Non l'ho toccata perché per l'italiano togliere gli accenti è
probabilmente giusto, ed è una decisione da prendere quando arriva
l'analizzatore italiano, non adesso di straforo.

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

---

## 23/09/2026 - Il corpus di dominio arriva, e porta due trappole

L'azienda ha abbandonato Documentale, quindi il corpus non viene piu' dal suo
archivio ma da albi pretori pubblici: 563 atti del Comune di Crispiano col PDF e
il testo integrale, 9.457 atti della Regione Friuli Venezia Giulia con le sole
schede. Importati in Documentale dal suo modello vero, riesportati col suo
comando: **10.018 documenti**, in due varianti che differiscono in esattamente
563 testi e in nient'altro. Provenienza e licenze in `eval/corpora/c3-albo/SOURCE.md`.

Le due varianti sono le due opzioni del Task C0, e la loro differenza e' la
misura di quanto vale indicizzare il corpo dei documenti. Prima di misurarla
sono saltate fuori due cose.

### Trappola 1: `scripts/compare` leggeva la chiave sbagliata, in silenzio

Il primo confronto sul corpus vero ha dato **1 risultato su 10.018 documenti**,
sempre lo stesso, per query diverse.

Causa: `scripts/compare` dichiarava `json:"id"`, ma da quando l'export di
Documentale scrive in formato BEIR la chiave e' `_id`. Tutti i documenti
arrivavano con id vuoto, `AddDocument("")` li sovrascriveva l'uno sull'altro, e
l'indice si riduceva a un documento solo. Nessun errore, nessun avviso: lo
strumento stampava tabelle di confronto come se niente fosse.

**Il confronto Elasticsearch/Koskidex del 23/09 non e' invalidato.** Verificato
nella storia di Documentale: quando e' stato fatto, l'export scriveva ancora
`id`, e il passaggio a `_id` e' di poche ore dopo (commit `6aa6d7b`). Ma
chiunque avesse rilanciato quel confronto dopo, me compreso, avrebbe ottenuto
numeri falsi senza un solo segnale.

Corretto: `_id`, piu' il `title` indicizzato come campo a se' con peso 5 come fa
Elasticsearch, invece di essere impastato nel testo. E soprattutto
`controllaIdentificatori`, che rifiuta di partire se il corpus ha id vuoti o
ripetuti. **La lezione e' che un identificatore sbagliato non fa rumore da solo:
va fatto rumoreggiare.**

### Trappola 2: su un corpus bimodale BM25 non misura la pertinenza

Corretto il primo difetto, il confronto gira davvero, e dice una cosa che non mi
aspettavo. Su tre query di prova, contando quanti dei primi 10 risultati sono
documenti col testo integrale (che sono il 5,4% del corpus):

| Query | and + euristico | or + euristico | or + BM25 |
|---|---|---|---|
| `determina noleggio veicolo` | 9 su 10 | 7 su 10 | **0 su 10** |
| `fornitura libri di testo scuole` | 2 su 10 | 2 su 10 | **0 su 10** |
| `ordinanza circolazione stradale` | 10 su 10 | 10 su 10 | **0 su 10** |

Zero su trenta. Non e' pertinenza, e' la normalizzazione della lunghezza: con
9.455 schede da ~200 caratteri e 563 documenti da ~9.000, la lunghezza media del
corpus e' tirata giu' dalle schede, e il fattore `b = 0,75` penalizza
sistematicamente tutto quello che e' lungo. BM25 sta facendo esattamente il suo
mestiere; e' il corpus a essere due corpora messi in un indice solo.

Guardando i primi risultati si vede a occhio: l'euristico mette in testa le
determine di noleggio veicoli, BM25 mette una determina sul *fermo
amministrativo di un veicolo* e due sul *noleggio di estintori e bagni chimici*
a una fiera - documenti corti che contengono le parole giuste.

**Conseguenza operativa: i due corpora non si misurano in un indice unico.**
Vanno tenuti separati, e la domanda del C0 va posta come confronto fra due
misure sullo stesso insieme di documenti - i 563 di Crispiano, una volta con la
sola scheda e una volta col testo - non fra due sottoinsiemi di un indice misto.
Se avessi misurato prima e guardato dopo, la risposta sarebbe stata "il testo
integrale peggiora il recupero", che e' falso ed e' un artefatto di `b`.

### Cosa e' scritto **prima** di misurare

Quando si misurera' sul serio, sui soli 563 di Crispiano e con query e giudizi
veri, l'ipotesi e' questa:

1. Il testo integrale **aumenta il richiamo** e basta: fa trovare documenti che
   la scheda non nomina. Sul nDCG@10 mi aspetto un guadagno piccolo, perche' la
   scheda di un atto amministrativo e' gia' un buon riassunto - l'oggetto di una
   determina e' scritto apposta per dire cos'e'.
2. Il guadagno e' **piu' grande sulle query lunghe** che su quelle di una o due
   parole, per la stessa ragione per cui il recupero disgiuntivo ha reso tanto:
   piu' parole ci sono, piu' e' probabile che qualcuna stia solo nel corpo.
3. Sulle query di tipo known-item con il numero d'atto il testo integrale **non
   serve a niente**, perche' il numero sta gia' nel titolo. Se invece rendesse,
   vuol dire che il titolo non e' indicizzato come credo, e va guardato li'.
4. `b` andra' rimisurato sul corpus di soli documenti lunghi: il valore 0,75 e'
   un default tarato su collezioni di articoli, e su un archivio dove tutti i
   documenti si somigliano in lunghezza dovrebbe contare meno.

---

## 23-24/09/2026 - Contro Elasticsearch sul corpus vero: il modello è lo stesso, il matching no

I dati di questa voce stanno nell'archivio della tesi
(`Universita-Martin/Magistrale/Tesi/risultati/`), non qui: `confronto/` per le
esecuzioni, `esperimenti/2026-09-23_cause-divergenza/` e
`esperimenti/2026-09-23_numero-atto/` per i due esperimenti, ciascuno con il
codice per rifarlo.

**Cosa credevo.** Stamattina del 23, su 14 documenti inventati, Koskidex e
Elasticsearch restituivano lo stesso insieme su ogni query, e ne avevo concluso
che Koskidex è un'implementazione fedele dello stesso modello di recupero.

**Cosa ho trovato.** Su 10.018 atti e 24 query, gli insiemi coincidono in **8
casi su 24**. Il modello è davvero lo stesso (congiuntivo, con refusi), ma il
matching differisce in quattro punti, che ho misurato dando a Koskidex
esattamente ciò che Elasticsearch ha indicizzato e correggendo una causa alla
volta, dietro interruttori temporanei mai entrati nel codice:

1. `best_fields` + `operator: and` vuole tutte le parole **nello stesso campo**;
   Koskidex le accetta sparse. Pesa più di tutto: 326 dei 447 documenti in più.
2. Soglie dei refusi: `AUTO` dà 1 refuso da 3 caratteri e 2 da 6, Koskidex 1 da
   4 e 2 da 8.
3. `fuzzySearchTermsLocked` accetta **qualunque termine che inizia con la parola
   cercata**, a qualsiasi distanza (`fuzzy.go:94`). Eredità dell'e-commerce.
4. `prefix_length: 1`: in Documentale la prima lettera è esatta.

Con tutte e quattro, 15 query su 24. Il resto non è diagnosticato e lo scrivo
così: probabilmente la tokenizzazione dei numeri e il tetto di 50 espansioni
delle query fuzzy di Elasticsearch, nessuna delle due verificata.

**La lezione è la stessa di BM25 su NFCorpus, al contrario.** Lì un controllo
scattava su un campione piccolo per il motivo sbagliato; qui un campione piccolo
non faceva scattare niente per il motivo sbagliato. Su 14 documenti le quattro
differenze non avevano occasione di vedersi. Nessuna conclusione su "stesso
comportamento" vale se non su dati veri, e con un numero di query che dia alle
differenze il modo di comparire.

**La scoperta che vale di più non riguarda Koskidex.** In produzione Documentale
mette l'atto cercato per numero al 12° posto su 85 (`ordinanza 187`) e al 5° su 8
(`determina 1223`). La causa è la tolleranza ai refusi sui numeri: `187` combacia
con `18` e `17`. Spenti i refusi, l'atto è primo e unico. `pathinfo()` peggiora
le cose, riducendo *"ORDINANZA N. 18.2026"* a *"ORDINANZA N. 18"*.

**E una cosa da spiegare su Koskidex.** Nella stessa prova, `or + BM25` mette
l'atto giusto al 3° e al 5° posto, dove l'euristico lo mette primo. BM25
peggiora la ricerca per numero. Non ancora indagato; il sospetto naturale è
l'IDF dei numeri corti, che in un archivio di atti numerati sono ovunque.

**Conseguenza per la tesi.** L'argomento "i miglioramenti misurati su Koskidex
valgono anche per Documentale" non regge più così com'era. Due strade, non si
escludono: insegnare a Koskidex a riprodurre il matching di Elasticsearch, dietro
flag, e mostrare insiemi identici; oppure misurare Elasticsearch direttamente,
che ora gira in locale sullo stesso corpus e può entrare nello stesso pool.

### Gli strumenti, da qui in poi

Ogni esecuzione di `scripts/evaluate` e `scripts/compare` scrive un file nuovo
nell'archivio della tesi, con il commit, se l'albero era pulito, l'impronta del
corpus, ogni impostazione e i tempi in millisecondi. Sistemarlo ha trovato tre
difetti, tutti miei: tempi arrotondati a zero sotto il microsecondo; i file di
risultato che, stando nel repository, facevano dichiarare "non committato"
ogni esecuzione dopo la prima; un contatore di collisione che finiva dopo il
nome e faceva sfuggire il file ai glob.

## 24-25/09/2026 - Parte F: Koskidex completo, e gli stessi insiemi di Elasticsearch

La regola era di non innestare in Documentale un motore incompleto. I buchi
trovati provandolo contro il contratto di Documentale sono chiusi (F1-F7 del
piano della tesi), ognuno con test scritti prima e visti fallire e con mutazioni
per controllare che i test mordano. Due volte le mutazioni hanno trovato test
che non mordevano: un punteggio costante passava il controllo d'ordine, e
`prefix_length` contato in byte passava un test su "perché", perché la
normalizzazione toglie gli accenti e l'italiano diventa ASCII.

**F7 ha chiuso il conto con Elasticsearch: 24 query su 24 con lo stesso
insieme.** Le due ipotesi che avevo scritto qui sui residui erano sbagliate
tutte e due, ed è giusto lasciarle scritte sopra:

- i 42 documenti "solo ES" di `ordinanza 187` non erano tokenizzazione dei
  numeri ma un **difetto mio**: i candidati fuzzy venivano solo dai bigrammi in
  comune, e `17` non ne ha con `187`. Una modifica rompe fino a due bigrammi,
  una trasposizione tre, e sotto una certa lunghezza non resta garanzia. Con le
  soglie predefinite capitava ad `atre` contro `arte`. Il motore prometteva la
  distanza di Damerau e non la manteneva: corretto per tutti, con una scansione
  del vocabolario per le parole corte;
- `max_expansions` non c'entra: rilanciata con 10.000, Elasticsearch dà gli
  stessi insiemi.

L'ultima causa l'ha data `_explain` su un documento: *"dell'illuminazione"*. Il
tokenizer standard di Elasticsearch tiene l'apostrofo fra due lettere dentro la
parola, e con le stesse regole unisce date e decimali. Ora è un'impostazione.

**Per Documentale è un difetto di produzione**: chi cerca "illuminazione" non
trova gli atti che scrivono "dell'illuminazione". Koskidex spezza, e per le
elisioni ha ragione; per le date, dove `18` non dovrebbe combaciare dentro
`18.01.2026`, ha ragione Elasticsearch. Il tokenizer predefinito va ripensato
per i numeri, non per le parole.

La lezione, un'altra volta: le ipotesi scritte senza verificarle erano
plausibili e sbagliate. Quella che ha risolto è venuta dal chiedere a
Elasticsearch perché non trovava un documento preciso.
