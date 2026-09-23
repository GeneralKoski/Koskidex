# C1 - collezioni pubbliche di riferimento

Servono a **una cosa sola**: dire se l'implementazione di BM25 ha un bug. Non
rispondono alla domanda di tesi, che vive sul corpus di dominio C2.

I dati non stanno in git. Si riscaricano con `../fetch.sh`, che verifica l'MD5
degli zip contro i valori fissati qui sotto.

Task B1 di `piano-autunno-2026.md` (repository `Universita-Martin`,
`Magistrale/Tesi/`). Scelte e scaricate il **23 settembre 2026**.

## Provenienza

Entrambe vengono dal benchmark [BEIR](https://github.com/beir-cellar/beir),
distribuite dalla UKP di Darmstadt:

```
https://public.ukp.informatik.tu-darmstadt.de/thakur/BEIR/datasets/{nome}.zip
```

Download diretto, nessuna registrazione, nessuna licenza da accettare.

| | zip | MD5 |
|---|---|---|
| `scifact` | 2.816.079 byte | `5f7d1de60b170fc8027bb7898e2efca1` |
| `nfcorpus` | 2.448.432 byte | `a89dba18a62ef92f7d323ec890a0d38d` |

## Cosa contengono, verificato a mano

| | **SciFact** | **NFCorpus** |
|---|---|---|
| Documenti | 5.183 | 3.633 |
| Query nel file `queries.jsonl` | 1.109 | 3.237 |
| Query di test (con giudizi) | 300 | 323 |
| Giudizi di test | 339 (1,1 per query) | 12.334 (38,2 per query) |
| Livelli di rilevanza | binaria (solo `1`) | graduata (`1` e `2`) |
| Lunghezza media documento | 215 parole | - |

Attenzione: `queries.jsonl` contiene le query di **tutti** gli split, e il
divario e' grosso in entrambe: SciFact ne ha 1.109 in totale contro 300 di test,
NFCorpus 3.237 contro 323. **Le query da valutare si ricavano dai qrels, non dal
file delle query**, altrimenti si misura su query che non hanno nessun giudizio
e ogni media crolla.

## Formato

Identico per entrambe.

`corpus.jsonl`, un documento per riga:

```json
{"_id": "MED-10", "title": "Statin Use and Breast Cancer Survival...", "text": "Recent studies have suggested...", "metadata": {"url": "..."}}
```

`queries.jsonl`, una query per riga:

```json
{"_id": "PLAIN-3", "text": "Breast Cancer Cells Feed on Cholesterol", "metadata": {"url": "..."}}
```

`qrels/test.tsv`, con riga di intestazione:

```
query-id	corpus-id	score
PLAIN-2	MED-2427	2
```

## Valori di riferimento

BM25 *flat*, nDCG@10, implementazione Anserini/Lucene:

| Collezione | BM25 nDCG@10 |
|---|---|
| **SciFact** | **0,6789** |
| **NFCorpus** | **0,3218** |

Fonti: [Pyserini 2CR BEIR](https://castorini.github.io/pyserini/2cr/beir.html),
[regression Anserini](https://github.com/castorini/anserini/blob/master/docs/fatjar-regressions/fatjar-regressions-v1.7.0.md),
[paper BEIR](https://www.alphaxiv.org/abs/2104.08663).

## Perché due e non una

**SciFact è la prova di fumo.** Il riferimento è alto, 0,68: se l'implementazione
ne fa 0,30 il bug è evidente al primo colpo. E con 1,1 documenti rilevanti per
query c'è una risposta giusta sola, quindi una singola query si verifica a mano
in un minuto. Risponde a "funziona per niente?".

**NFCorpus è la prova vera.** 38 giudizi per query e rilevanza graduata danno un
nDCG stabile, e struttura più simile a un documentale, dove molti documenti sono
parzialmente pertinenti invece che uno solo giusto. Risponde a "il numero è
affidabile?".

Costano 2,5 MB l'una e hanno lo stesso formato, quindi la seconda è quasi gratis
una volta scritto il lettore.

## Tre trappole, che contano piu' della scelta della collezione

**1. Koskidex non riprodurra' 0,3218, e va bene cosi'.** Il riferimento e'
Lucene: stemming, sue stopword, suo tokenizer. Koskidex ha un tokenizer diverso e
nessuno stemmer. Il criterio di accettazione **non e'** "fa lo stesso numero".
E':

- cade in una fascia plausibile attorno al riferimento, e
- **stacca nettamente il punteggio legacy attuale**.

La seconda meta' e' quella che dimostra qualcosa. Il confronto interno contro il
baseline congelato e' il risultato di tesi; l'aggancio al valore pubblicato serve
solo a escludere che si stia misurando spazzatura.

**2. La tolleranza ai refusi va spenta.** In `DefaultSettings` sta a
`Enabled: true` e Lucene non ne ha nessuna. Lasciandola accesa si misura una cosa
diversa da quella pubblicata, e non si vede dal numero.

**3. Il riferimento e' "flat", campo unico.** I documenti BEIR hanno `title` e
`text`. Indicizzandoli con pesi diversi, come fa il corpus del baseline (2.0 e
1.0), il numero non e' confrontabile. Per C1 serve campo unico o pesi uguali.

Tutte e tre vanno rimesse a posto prima di scrivere il primo numero nel diario,
non dopo averlo scritto.
