#!/usr/bin/env bash
# SPEC 9.2 T43: generate and validate curated release notes.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"
chglog="${CHGLOG:-$(go env GOPATH)/bin/chglog}"
tmp=""

cleanup() {
    if [ -n "$tmp" ]; then rm -f "$tmp"; fi
}
trap cleanup EXIT

generate() {
    "$chglog" format -i packaging/changelog.yaml -p krot -t repo -o "$1"
}

check_current() {
    tmp="$(mktemp)"
    generate "$tmp"
    if ! cmp -s "$tmp" CHANGELOG.md; then
        echo "CHANGELOG.md is stale; run make changelog" >&2
        return 1
    fi
}

case "${1:-}" in
    "")
        generate CHANGELOG.md
        ;;
    --check)
        check_current
        ;;
    v*)
        tag="$1"
        version="${tag#v}"
        check_current
        tag_date=""
        tag_epoch="$(TZ=UTC git for-each-ref "refs/tags/$tag" --format='%(taggerdate:unix)')"
        if [ -n "$tag_epoch" ]; then tag_date="$(date -u -d "@$tag_epoch" +%F)"; fi
        if [ -z "$tag_date" ]; then
            echo "$tag must be an existing annotated tag" >&2
            exit 1
        fi
        entry_date="$(awk -v version="$version" '
            $0 == version { found=1; getline; next }
            found && NF { print; exit }
        ' CHANGELOG.md)"
        if [ -z "$entry_date" ]; then
            echo "no release notes for $version" >&2
            exit 1
        fi
        if [ "$entry_date" != "$tag_date" ]; then
            echo "release notes date $entry_date does not match tag date $tag_date" >&2
            exit 1
        fi
        mkdir -p dist
        awk -v version="$version" '
            $0 == version { found=1; print; next }
            found && $0 ~ /^[0-9]+\.[0-9]+\.[0-9]+([-+][0-9A-Za-z.-]+)?$/ { exit }
            found { print }
        ' CHANGELOG.md > dist/release-notes.md
        notes="$(cat dist/release-notes.md)"
        if [[ "$notes" != *$'\n* '* ]]; then
            echo "release notes for $version have no changes" >&2
            exit 1
        fi
        ;;
    *)
        echo "usage: $0 [--check|vX.Y.Z]" >&2
        exit 2
        ;;
esac
