# C3 - albi pretori pubblici, il corpus di dominio

Questo **sostituisce il C2**, il corpus che doveva venire dall'archivio di
Documentale. Dieffetech ha abbandonato il progetto: niente documenti di clienti,
e nessuno a cui chiedere il permesso di conservarne il testo integrale.

Documentale resta il sistema misurato - gira in locale, l'export funziona, il
confronto con Elasticsearch e' gia' stato fatto. Cambia solo cosa ci sta dentro.

I dati non stanno in git. Si riscaricano con `../fetch-albo.py`.

Task E2 di `piano-autunno-2026.md` (repository `Universita-Martin`,
`Magistrale/Tesi/`; rimosso il 25/09/2026, l'ultima versione è nel commit
`5f9b080`). Cercate, provate e scelte il **23 settembre 2026**.

## Perche' l'albo pretorio

Un documentale non e' una collezione di articoli: e' l'archivio di **una sola
organizzazione**, con generi ricorrenti, un vocabolario suo, e gente che cerca
un atto che sa gia' esistere. L'albo pretorio ha esattamente questa forma -
determine, delibere, ordinanze, avvisi, liquidazioni - ed e' pubblico per
obbligo di legge, quindi riscaricabile da chiunque legga la tesi.

## Le due fonti

### `crispiano/` - il corpus col testo integrale

Albo pretorio del Comune di Crispiano (TA), feed RSS, licenza **CC BY 4.0**.
Censito su dati.gov.it e sul CKAN della Regione Puglia.

```
https://www.trasparenzacrispiano.it/rss.xml
```

| | |
|---|---|
| Atti nel feed | 563 |
| Periodo | 03/11/2025 - 22/09/2026 |
| Con link diretto al PDF | 551 |
| In busta di firma `.p7m` | 12 |
| Generi | 390 determine, 82 delibere, 58 ordinanze, il resto avvisi e pubblicazioni |
| Testo | **nativo**, estraibile con `pdftotext`: nessun OCR in mezzo |
| Campione di 5 PDF | 2-6 pagine, da 5.472 a 13.311 caratteri |

**Trappola, e costa cara.** La scheda del dataset sul CKAN dice che il feed non
porta i PDF e rimanda alla piattaforma dell'ente. E' **sbagliata**: il `<link>`
di ogni item *e'* il PDF. Fermandosi alla descrizione si scarta l'unica fonte
buona che esista.

I `.p7m` sono PDF dentro una busta PKCS#7 e si aprono con
`openssl smime -verify -noverify -inform DER`. La firma non si verifica:
interessa il contenuto, non chi l'ha firmato.

### `fvg/` - il corpus a sole schede

Albo pretorio della Regione Friuli Venezia Giulia, API Socrata senza chiave,
licenza **IODL 2.0**, attribuzione *Regione autonoma Friuli Venezia Giulia*.

```
https://www.dati.friuliveneziagiulia.it/resource/vny3-2fkg.json
```

| | |
|---|---|
| Atti | 9.457 |
| Periodo | 14/04/2011 - 23/09/2026 (aggiornato il giorno stesso) |
| Enti | 171 |
| Tipologie di atto | 46 |
| Campi | `ente`, `tipologia_atto`, `ufficio_competente`, `oggetto`, `numero_atto`, date, `link_albo_comunale` |
| Lunghezza media dell'oggetto | 201 caratteri |
| Testo integrale | **assente**: c'e' solo l'oggetto |

I 171 enti si comportano come 171 clienti di un documentale multi-tenant, e
`tipologia_atto` e `ufficio_competente` sono un `meta_true` gia' scritto da chi
ha pubblicato il dato - cioe' i campi che in Documentale l'IA deve indovinare,
qui ci sono per davvero.

**Due cose trovate contando, non leggendo.**

L'id non puo' essere `ente + tipologia + numero + data`: il 9% degli atti ha un
`numero_atto` che non e' un numero (`-`, `.`, `///`, o l'oggetto ricopiato), e
cosi' 37 atti diversi collassavano sullo stesso id. La chiave porta in coda
l'impronta SHA-1 dell'oggetto: e' deterministica, quindi viene uguale a ogni
riscaricamento anche se l'ordine cambia. Restano 2 id ripetuti, e sono atti
pubblicati due volte con lo stesso oggetto: quelli e' giusto che collassino.

Gli atti con un numero vero sono 8.577, il 90%. Sono quelli su cui si possono
generare le query known-item automatiche del tipo *"determina 1223 Crispiano"*.

## Le fonti scartate, e perche'

**AlboPOP** (`albopop.it`) era la strada che sembrava giusta: il progetto che
converte gli albi pretori in feed di formato comune, 253 comuni schedati, 215
con un feed dichiarato. Provati tutti e 215:

- 105 feed rispondono ancora e hanno almeno un item
- 71 dichiarano allegati, 9.564 in totale
- scaricandone 14 a campione, **3 arrivano**. Gli altri rispondono `200 OK` con
  944 byte di HTML: una pagina di errore travestita da successo

I feed vivi sono quasi tutti fermi al 2021, venivano dagli scraper del progetto
sulla ricostruzione post-terremoto, che non girano piu'. Resta un ottimo
catalogo di chi pubblica cosa; come sorgente di documenti non regge. **Il modo
in cui non regge - 200 su un errore - e' la ragione per cui andava provato
scaricando invece che contando le righe del catalogo.**

Su **dati.gov.it** ci sono 297 dataset che parlano di albo pretorio, ma solo 2
espongono un feed o un XML, e sono le due fonti tenute qui sopra.

Provata anche l'ipotesi che lo schema `trasparenza<comune>.it/rss.xml` valesse
per altri comuni serviti dalla stessa piattaforma: dieci provati, zero risolvono
il DNS.

## Dati personali

Gli atti dell'albo contengono nomi di persone. Cercando marcatori espliciti
(`sig.`, `nato a`, `codice fiscale`, `residente in`, `pubblicazione di
matrimonio`):

- FVG: **79 oggetti su 9.457**
- Crispiano: le pubblicazioni di matrimonio riportano nomi, date e luoghi di
  nascita nel corpo del PDF

Sono dati pubblicati per obbligo di legge, quindi usarli per misurare un motore
di ricerca e' legittimo. Ma **il corpus non si committa**: una cosa e' un dato
pubblico sul sito di un comune, un'altra e' ripubblicarlo in un archivio su
GitHub. Nel repository stanno gli script che lo riscaricano; in tesi, gli
esempi si citano anonimizzati.

Lo script conta questi atti e lo stampa a ogni esecuzione. Non li toglie: sono
parte dell'archivio, e toglierli falserebbe il corpus.

## Cosa fa lo script, e cosa non fa

`../fetch-albo.py` si ferma ai dati grezzi: `atti.jsonl` per fonte, piu' `pdf/`
e `txt/` per Crispiano, piu' `falliti.tsv` con gli atti che non sono arrivati e
il motivo.

**Non** produce il formato BEIR. A trasformare gli atti in documenti ci pensa
Documentale, con il suo comando di import e il suo modello vero
(`Document` + `currentVersion` + campi), e il corpus esce dall'export del C1.
Un corpus costruito scavalcando il sistema non direbbe niente sul sistema, e il
confronto con Elasticsearch non varrebbe niente.
