#!/usr/bin/env bash
# Scarica le collezioni pubbliche usate come C1 e ne verifica l'integrita'.
# I dati non stanno in git: si riscaricano con questo script.
# Provenienza, numeri di riferimento e trappole: c1-public/SOURCE.md
set -euo pipefail

cd "$(dirname "$0")"
DEST="c1-public"
BASE="https://public.ukp.informatik.tu-darmstadt.de/thakur/BEIR/datasets"

# nome:md5 dello zip, fissati il 2026-09-23
DATASETS=(
  "scifact:5f7d1de60b170fc8027bb7898e2efca1"
  "nfcorpus:a89dba18a62ef92f7d323ec890a0d38d"
)

md5_of() {
  if command -v md5sum >/dev/null 2>&1; then md5sum "$1" | cut -d' ' -f1
  else md5 -q "$1"
  fi
}

mkdir -p "$DEST"

for entry in "${DATASETS[@]}"; do
  name="${entry%%:*}"
  want="${entry##*:}"
  zip="$DEST/$name.zip"

  if [ -d "$DEST/$name" ]; then
    echo "$name: gia' presente, salto"
    continue
  fi

  echo "$name: scarico..."
  curl -fsSL -o "$zip" "$BASE/$name.zip"

  got="$(md5_of "$zip")"
  if [ "$got" != "$want" ]; then
    echo "$name: MD5 NON CORRISPONDE" >&2
    echo "  atteso:  $want" >&2
    echo "  trovato: $got" >&2
    echo "  Lo zip e' cambiato all'origine. Non usarlo per misurare senza prima" >&2
    echo "  capire cosa e' cambiato e aggiornare SOURCE.md." >&2
    rm -f "$zip"
    exit 1
  fi

  unzip -oq "$zip" -d "$DEST"
  rm -f "$zip"
  echo "$name: ok, $(wc -l < "$DEST/$name/corpus.jsonl" | tr -d ' ') documenti"
done

echo
echo "Collezioni pronte in $DEST/. Ricorda che per confrontarsi con i numeri"
echo "pubblicati servono campo unico e tolleranza ai refusi spenta: SOURCE.md."
