# Test data

| File | What it is | Source | Licence |
|---|---|---|---|
| `voc.txt`, `output.txt` | Porter's test vocabulary: 23531 English words and their stems | tartarus.org/martin/PorterStemmer | published by the author for testing implementations |
| `itlight.txt` | 35494 Italian words and the stem Lucene's ItalianLightStemmer gives each one, tab-separated | `itlight.txt` inside `lucene/analysis/common/src/test/org/apache/lucene/analysis/it/itlighttestdata.zip` of apache/lucene, branch main, downloaded 25/09/2026 (zip sha256 `f439b91c8e7f9f8889d9c7dcea90f4cfce6e958552e7bfd679400ae3da1ba20a`, file dated 2010) | Apache License 2.0 |

`itlight.txt` is copied out of the zip unchanged (sha256
`4f8bf5f3924e22301603ac625280e2573e87d54166cbfc841c080d60eeb9c58f`). It is the
file Lucene's own `TestItalianLightStemFilter` checks the stemmer against.
