#!/usr/bin/env python3
"""Scarica gli albi pretori pubblici che fanno da corpus di dominio.

Due fonti, scelte perche' sono le uniche due vive fra quelle censite il
2026-09-23 (le altre, e il perche' sono state scartate, stanno in
c3-albo/SOURCE.md):

  crispiano  563 atti con il PDF allegato  -> il corpus col testo integrale
  fvg        9457 atti di 171 enti, solo metadati -> il corpus a sole schede

Lo script si ferma qui: produce dati grezzi. A trasformarli in documenti ci
pensa Documentale, con il suo comando di import, perche' un corpus costruito
scavalcando il sistema non direbbe niente sul sistema.

Solo libreria standard, come il resto del progetto. Serve pdftotext (poppler)
per il testo e openssl per le buste di firma .p7m.

    ./fetch-albo.py --fonte tutte
    ./fetch-albo.py --fonte crispiano --limite 20   # per provare
"""

import argparse
import hashlib
import json
import os
import re
import ssl
import subprocess
import sys
import time
import urllib.error
import urllib.request
import xml.etree.ElementTree as ET
from email.utils import parsedate_to_datetime

FEED_CRISPIANO = "https://www.trasparenzacrispiano.it/rss.xml"
API_FVG = "https://www.dati.friuliveneziagiulia.it/resource/vny3-2fkg.json"

# Farsi riconoscere: sono server di comuni, non CDN.
UA = "Koskidex-eval/1.0 (tesi magistrale, Universita' di Parma)"

# Marcatori di dato personale. Non servono a filtrare - sono atti pubblici per
# legge e toglierli falserebbe il corpus - ma a sapere cosa si ha in mano prima
# di mostrare un esempio in giro.
PERSONALI = re.compile(
    r"(?i)\b(sig\.?r?a?\.?|nato a|nata a|nato in|nata in|codice fiscale|c\.f\.|"
    r"residente in|pubblicazione di matrimonio)\b"
)


def scarica(url, timeout=60):
    req = urllib.request.Request(url, headers={"User-Agent": UA})
    ctx = ssl.create_default_context()
    with urllib.request.urlopen(req, timeout=timeout, context=ctx) as r:
        return r.read()


def estrai_testo(pdf, txt):
    """Ritorna i caratteri estratti, oppure -1 se pdftotext non ce l'ha fatta."""
    try:
        subprocess.run(
            ["pdftotext", "-enc", "UTF-8", pdf, txt],
            check=True, capture_output=True, timeout=120,
        )
    except (subprocess.CalledProcessError, subprocess.TimeoutExpired):
        return -1
    try:
        return os.path.getsize(txt)
    except OSError:
        return -1


def sbusta_p7m(p7m, pdf):
    """Un .p7m e' un PDF dentro una busta PKCS#7. La firma non la verifichiamo:
    interessa il contenuto, non chi l'ha firmato."""
    try:
        subprocess.run(
            ["openssl", "smime", "-verify", "-noverify", "-inform", "DER",
             "-in", p7m, "-out", pdf],
            check=True, capture_output=True, timeout=60,
        )
        return os.path.getsize(pdf) > 0
    except (subprocess.CalledProcessError, subprocess.TimeoutExpired, OSError):
        return False


def crispiano(dest, limite, pausa):
    dir_atti = os.path.join(dest, "crispiano")
    dir_pdf = os.path.join(dir_atti, "pdf")
    dir_txt = os.path.join(dir_atti, "txt")
    for d in (dir_pdf, dir_txt):
        os.makedirs(d, exist_ok=True)

    print(f"crispiano: scarico il feed da {FEED_CRISPIANO}")
    radice = ET.fromstring(scarica(FEED_CRISPIANO))
    item = radice.findall("./channel/item")
    print(f"crispiano: {len(item)} atti nel feed")
    if limite:
        item = item[:limite]
        print(f"crispiano: mi fermo ai primi {limite}")

    righe, falliti = [], []
    conta = {"pdf": 0, "p7m": 0, "altro": 0, "senza_testo": 0, "personali": 0}

    for i, it in enumerate(item, 1):
        ident = (it.findtext("title") or "").strip()
        oggetto = (it.findtext("description") or "").strip()
        url = (it.findtext("link") or "").strip()
        data = (it.findtext("pubDate") or "").strip()
        try:
            data = parsedate_to_datetime(data).date().isoformat()
        except (TypeError, ValueError):
            pass

        if not ident or not url:
            falliti.append((ident or f"riga-{i}", "item senza id o senza link"))
            continue

        estensione = os.path.splitext(url)[1].lower()
        conta[{".pdf": "pdf", ".p7m": "p7m"}.get(estensione, "altro")] += 1
        if estensione not in (".pdf", ".p7m"):
            falliti.append((ident, f"allegato non gestito: {estensione or 'nessuna estensione'}"))
            continue

        pdf = os.path.join(dir_pdf, f"{ident}.pdf")
        txt = os.path.join(dir_txt, f"{ident}.txt")

        if not os.path.exists(pdf):
            try:
                corpo = scarica(url)
            except (urllib.error.URLError, OSError, TimeoutError) as e:
                falliti.append((ident, f"download fallito: {e}"))
                continue
            if estensione == ".p7m":
                grezzo = pdf + ".p7m"
                with open(grezzo, "wb") as f:
                    f.write(corpo)
                if not sbusta_p7m(grezzo, pdf):
                    falliti.append((ident, "busta .p7m non aperta"))
                    os.remove(grezzo)
                    continue
                os.remove(grezzo)
            else:
                with open(pdf, "wb") as f:
                    f.write(corpo)
            time.sleep(pausa)

        n = estrai_testo(pdf, txt)
        if n <= 0:
            conta["senza_testo"] += 1
            falliti.append((ident, "pdftotext non ha estratto niente: probabile scansione"))

        if PERSONALI.search(oggetto):
            conta["personali"] += 1

        righe.append({
            "id": ident,
            "oggetto": oggetto,
            "data_pubblicazione": data,
            "url": url,
            "firmato": estensione == ".p7m",
            "file_pdf": os.path.relpath(pdf, dest),
            "file_testo": os.path.relpath(txt, dest) if n > 0 else None,
            "caratteri_testo": max(n, 0),
            "ente": "Comune di Crispiano",
            "fonte": "trasparenzacrispiano.it",
            "licenza": "CC BY 4.0",
        })

        if i % 50 == 0:
            print(f"  {i}/{len(item)}...")

    scrivi(os.path.join(dir_atti, "atti.jsonl"), righe)
    scrivi_falliti(os.path.join(dir_atti, "falliti.tsv"), falliti)

    testi = [r["caratteri_testo"] for r in righe if r["caratteri_testo"] > 0]
    print(f"crispiano: {len(righe)} atti scritti, {len(falliti)} falliti")
    print(f"  allegati: {conta['pdf']} pdf, {conta['p7m']} firmati .p7m, {conta['altro']} altro")
    if testi:
        print(f"  testo estratto da {len(testi)} atti, media {sum(testi)//len(testi)} caratteri")
    if conta["senza_testo"]:
        print(f"  ATTENZIONE: {conta['senza_testo']} senza testo estraibile")
    print(f"  {conta['personali']} oggetti con marcatori di dato personale")
    return len(righe)


def fvg(dest, limite):
    dir_atti = os.path.join(dest, "fvg")
    os.makedirs(dir_atti, exist_ok=True)

    n = limite or 50000
    url = f"{API_FVG}?$limit={n}&$order=data_inizio_pubblicazione"
    print(f"fvg: scarico da {API_FVG}")
    dati = json.loads(scarica(url, timeout=120))
    print(f"fvg: {len(dati)} atti")

    righe, senza_oggetto, personali = [], 0, 0
    for d in dati:
        oggetto = (d.get("oggetto") or "").strip()
        if not oggetto:
            senza_oggetto += 1
            continue
        if PERSONALI.search(oggetto):
            personali += 1
        ente = (d.get("ente") or "").strip()
        numero = (d.get("numero_atto") or "").strip()
        # ente + tipologia + numero + data non identifica: il 9% degli atti ha
        # un numero_atto che non e' un numero ('-', '.', '///', o l'oggetto
        # stesso ricopiato), e cosi' 37 atti diversi finivano sullo stesso id.
        # Il discriminante e' l'impronta dell'oggetto e non un contatore, perche'
        # deve venire uguale a ogni riscaricamento anche se l'ordine cambia.
        # Due atti con lo stesso oggetto restano un id solo: quelli sono copie.
        impronta = hashlib.sha1(oggetto.encode("utf8")).hexdigest()[:8]
        righe.append({
            "id": f"{ente}|{d.get('tipologia_atto','')}|{numero}|{d.get('data_inizio_pubblicazione','')[:10]}|{impronta}",
            "oggetto": oggetto,
            "ente": ente,
            "tipologia_atto": (d.get("tipologia_atto") or "").strip(),
            "ufficio_competente": (d.get("ufficio_competente") or "").strip(),
            "numero_atto": numero,
            "data_inizio_pubblicazione": (d.get("data_inizio_pubblicazione") or "")[:10],
            "data_fine_pubblicazione": (d.get("data_fine_pubblicazione") or "")[:10],
            "url": (d.get("link_albo_comunale") or "").strip(),
            "file_testo": None,
            "fonte": "dati.friuliveneziagiulia.it/resource/vny3-2fkg",
            "licenza": "IODL 2.0 - Regione autonoma Friuli Venezia Giulia",
        })

    # Gli id sono la chiave con cui Documentale riconosce un atto gia' importato:
    # se non sono unici, un reimport ne perde per strada in silenzio.
    unici = len({r["id"] for r in righe})
    numerati = sum(1 for r in righe if re.fullmatch(r"\d+([/-]\d+)*", r["numero_atto"].strip()))
    scrivi(os.path.join(dir_atti, "atti.jsonl"), righe)
    print(f"fvg: {len(righe)} atti scritti, {senza_oggetto} scartati senza oggetto")
    print(f"  enti: {len({r['ente'] for r in righe})}, tipologie: {len({r['tipologia_atto'] for r in righe})}")
    print(f"  id distinti: {unici}" + ("" if unici == len(righe) else f" ({len(righe)-unici} atti pubblicati due volte con lo stesso oggetto)"))
    print(f"  con un numero d'atto vero: {numerati} ({100*numerati//len(righe)}%) - sono quelli su cui si possono generare query known-item")
    print(f"  {personali} oggetti con marcatori di dato personale")
    return len(righe)


def scrivi(path, righe):
    with open(path, "w", encoding="utf8") as f:
        for r in righe:
            f.write(json.dumps(r, ensure_ascii=False) + "\n")


def scrivi_falliti(path, falliti):
    """Il file si scrive sempre, anche vuoto: un file assente non si distingue
    da un file che nessuno ha guardato."""
    with open(path, "w", encoding="utf8") as f:
        f.write("id\tmotivo\n")
        for ident, motivo in falliti:
            f.write(f"{ident}\t{motivo}\n")


def main():
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("--fonte", choices=["crispiano", "fvg", "tutte"], default="tutte")
    p.add_argument("--dest", default=os.path.join(os.path.dirname(os.path.abspath(__file__)), "c3-albo"))
    p.add_argument("--limite", type=int, default=0, help="quanti atti al massimo, 0 = tutti")
    p.add_argument("--pausa", type=float, default=0.3, help="secondi fra un download e l'altro")
    a = p.parse_args()

    if a.fonte in ("crispiano", "tutte"):
        for strumento in ("pdftotext", "openssl"):
            if subprocess.run(["which", strumento], capture_output=True).returncode != 0:
                sys.exit(f"manca {strumento}: senza non si ricava il testo dai PDF")

    os.makedirs(a.dest, exist_ok=True)
    totale = 0
    if a.fonte in ("crispiano", "tutte"):
        totale += crispiano(a.dest, a.limite, a.pausa)
    if a.fonte in ("fvg", "tutte"):
        totale += fvg(a.dest, a.limite)

    print(f"\n{totale} atti in {a.dest}. Ora tocca a Documentale importarli.")


if __name__ == "__main__":
    main()
