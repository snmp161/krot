#!/usr/bin/env bash
# SPEC 9.4 T47: render valid Debian changelog from curated release notes.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"
mkdir -p dist
out="dist/changelog.Debian"

version=""
date_value=""
packager=""
notes=()

flush() {
    if [ -z "$version" ]; then return; fi
    deb_version="$version"
    if [[ "$deb_version" == *-* ]]; then
        base="${deb_version%%-*}"
        prerelease="${deb_version#*-}"
        deb_version="${base}~${prerelease}"
    fi
    printf 'krot (%s-1) stable; urgency=low\n' "$deb_version" >> "$out"
    for note in "${notes[@]}"; do
        wrapped="$(printf '%s\n' "$note" | fold -s -w 72)"
        first=1
        while IFS= read -r part; do
            if [ "$first" = 1 ]; then
                printf '  * %s\n' "$part" >> "$out"
                first=0
            else
                printf '    %s\n' "$part" >> "$out"
            fi
        done <<< "$wrapped"
    done
    printf '\n -- %s  %s\n\n' "$packager" "$(date -u -d "$date_value" --rfc-email)" >> "$out"
}

: > "$out"
while IFS= read -r line; do
    case "$line" in
        "- semver: "*)
            flush
            version="${line#- semver: }"
            date_value=""
            packager=""
            notes=()
            ;;
        "  date: "*) date_value="${line#  date: }" ;;
        "  packager: "*) packager="${line#  packager: }" ;;
        "      note: "*) notes+=("${line#      note: }") ;;
    esac
done < packaging/changelog.yaml
flush

gzip -9n -c "$out" > "$out.gz"
rm "$out"
