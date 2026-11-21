#!/usr/bin/env bash
# SPEC 9.2, 9.5 T46: shared deterministic binary and DEB/RPM build.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

version="${VERSION:-0.0.0}"
display_version="${DISPLAY_VERSION:-dev}"
commit="${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo none)}"
SOURCE_DATE_EPOCH="${SOURCE_DATE_EPOCH:-$(git show -s --format=%ct HEAD 2>/dev/null || echo 0)}"
export SOURCE_DATE_EPOCH
build_date="$(date -u -d "@$SOURCE_DATE_EPOCH" +%Y-%m-%dT%H:%M:%SZ)"
DESCRIPTION="$(cat packaging/description.txt)"
export DESCRIPTION
nfpm_bin="$(go env GOPATH)/bin/nfpm"

mkdir -p dist
for arch in amd64 arm64; do
    CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -buildvcs=false -trimpath \
        -ldflags "-s -w -X main.version=$display_version -X main.commit=$commit -X main.date=$build_date" \
        -o "dist/krot-linux-$arch" ./cmd/krot
done
cp LICENSE THIRD_PARTY_NOTICES dist/

dist/krot-linux-amd64 --completion bash > dist/krot.bash
dist/krot-linux-amd64 --completion zsh > dist/_krot
bash -n dist/krot.bash
scripts/compress-man.sh

for arch in amd64 arm64; do
    cp "dist/krot-linux-$arch" dist/krot
    for format in deb rpm; do
        GOARCH="$arch" VERSION="$version" "$nfpm_bin" package -f packaging/nfpm.yaml -p "$format" -t dist/
    done
done
rm dist/krot dist/krot.bash dist/_krot
rm -rf dist/man

(
    cd dist
    export LC_ALL=C
    sha256sum -- LICENSE THIRD_PARTY_NOTICES krot-linux-* *.deb *.rpm > SHA256SUMS
)
