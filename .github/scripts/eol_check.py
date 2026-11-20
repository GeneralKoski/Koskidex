"""Fail when a runtime the project builds on is past its end of life.

Reads the Go version from go.mod and the golang/alpine/node base images from
the Dockerfiles given on the command line, and asks endoflife.date about each.
Exit code 1 if any of them is end of life, 2 if the check itself failed.
"""

import json
import re
import sys
import urllib.request

PRODUCTS = {"golang": "go", "alpine": "alpine-linux", "node": "nodejs"}


def cycles(paths):
    found = set()
    for path in paths:
        text = open(path, encoding="utf-8").read()
        if path.endswith("go.mod"):
            m = re.search(r"^go (\d+\.\d+)", text, re.M)
            if m:
                found.add(("go", m.group(1), path))
            continue
        for image, tag in re.findall(r"^FROM\s+([\w./-]+):([\w.-]+)", text, re.M | re.I):
            product = PRODUCTS.get(image.split("/")[-1])
            m = re.match(r"(\d+(?:\.\d+)?)", tag)
            if product and m:
                cycle = m.group(1)
                # Go cycles are major.minor; alpine and node tags already are.
                found.add((product, cycle, path))
    return sorted(found)


def main():
    rows, eol = [], False
    for product, cycle, path in cycles(sys.argv[1:]):
        url = f"https://endoflife.date/api/v1/products/{product}/releases/{cycle}"
        try:
            with urllib.request.urlopen(url, timeout=20) as resp:
                info = json.load(resp)["result"]
        except Exception as exc:  # noqa: BLE001 - report and fail the step
            print(f"{product} {cycle} ({path}): check failed: {exc}")
            sys.exit(2)
        latest = (info.get("latest") or {}).get("name", "?")
        if info.get("isEol"):
            eol = True
            rows.append(f"- **{product} {cycle}** ({path}): END OF LIFE since {info.get('eolFrom')}, latest {latest}")
        else:
            rows.append(f"- {product} {cycle} ({path}): supported until {info.get('eolFrom') or 'n/a'}, latest {latest}")
    print("\n".join(rows))
    sys.exit(1 if eol else 0)


if __name__ == "__main__":
    main()
