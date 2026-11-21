#!/usr/bin/env bash
# SPEC 4.7, 9.4 T42: deterministic compressed manuals for DEB/RPM.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

for entry in \
    krot.1:man1 \
    krot.ru.1:ru/man1 \
    krot.zh.1:zh_CN/man1 \
    krot.zh-Hant.1:zh_TW/man1 \
    krot.es.1:es/man1 \
    krot.pt.1:pt_BR/man1 \
    krot.ko.1:ko/man1 \
    krot.ja.1:ja/man1 \
    krot.ar.1:ar/man1 \
    krot.fr.1:fr/man1 \
    krot.de.1:de/man1
do
    src="${entry%%:*}"
    dir="${entry#*:}"
    mkdir -p "dist/man/$dir"
    gzip -9n -c "packaging/$src" > "dist/man/$dir/krot.1.gz"
done
