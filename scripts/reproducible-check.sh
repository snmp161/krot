#!/usr/bin/env bash
# SPEC 9.5 T46: two independent clones must produce identical release artifacts.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT

for name in one two; do
    git clone --quiet --no-local --no-hardlinks "$root" "$work/$name"
done
(
    cd "$work/one"
    TZ=UTC VERSION=0.0.0 DISPLAY_VERSION=dev COMMIT=none ./scripts/build-release.sh
)
(
    cd "$work/two"
    TZ=Pacific/Honolulu VERSION=0.0.0 DISPLAY_VERSION=dev COMMIT=none ./scripts/build-release.sh
)

cmp "$work/one/dist/SHA256SUMS" "$work/two/dist/SHA256SUMS"
if ! diff -qr "$work/one/dist" "$work/two/dist"; then
    echo "release artifacts are not reproducible" >&2
    exit 1
fi
echo "reproducible build OK: all artifacts are byte-for-byte identical"
