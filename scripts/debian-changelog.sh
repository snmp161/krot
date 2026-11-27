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

decode_scalar() {
    local value="$1" inner char escaped
    local i=0
    decoded_scalar=""

    if [[ "$value" == \"* ]]; then
        if (( ${#value} < 2 )) || [[ "${value: -1}" != '"' ]]; then
            printf 'invalid or unsupported YAML scalar: %s\n' "$value" >&2
            return 1
        fi
        inner="${value:1:${#value}-2}"
        while (( i < ${#inner} )); do
            char="${inner:i:1}"
            case "$char" in
                '"')
                    printf 'unescaped quote in YAML scalar: %s\n' "$value" >&2
                    return 1
                    ;;
                \\)
                    (( i += 1 ))
                    if (( i >= ${#inner} )); then
                        printf 'unfinished escape in YAML scalar: %s\n' "$value" >&2
                        return 1
                    fi
                    escaped="${inner:i:1}"
                    case "$escaped" in
                        '"'|\\) decoded_scalar+="$escaped" ;;
                        *)
                            printf 'unsupported escape in YAML scalar: %s\n' "$value" >&2
                            return 1
                            ;;
                    esac
                    ;;
                *) decoded_scalar+="$char" ;;
            esac
            (( i += 1 ))
        done
    elif [[ "$value" == \'* ]]; then
        if (( ${#value} < 2 )) || [[ "${value: -1}" != "'" ]]; then
            printf 'invalid or unsupported YAML scalar: %s\n' "$value" >&2
            return 1
        fi
        inner="${value:1:${#value}-2}"
        while (( i < ${#inner} )); do
            char="${inner:i:1}"
            if [[ "$char" != "'" ]]; then
                decoded_scalar+="$char"
            else
                (( i += 1 ))
                if (( i >= ${#inner} )) || [[ "${inner:i:1}" != "'" ]]; then
                    printf 'single quote in YAML scalar must be doubled: %s\n' "$value" >&2
                    return 1
                fi
                decoded_scalar+="'"
            fi
            (( i += 1 ))
        done
    else
        decoded_scalar="$value"
    fi
}

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
        "      note: "*)
            decode_scalar "${line#      note: }"
            notes+=("$decoded_scalar")
            ;;
    esac
done < packaging/changelog.yaml
flush

gzip -9n -c "$out" > "$out.gz"
rm "$out"
