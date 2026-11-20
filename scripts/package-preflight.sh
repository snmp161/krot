#!/usr/bin/env bash
# SPEC 9.1 T36: сборка DEB/RPM для обеих архитектур и representative smoke на amd64.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

version="${VERSION:-0.0.0}"
nfpm_bin="$(go env GOPATH)/bin/nfpm"
mkdir -p dist
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o dist/krot-linux-amd64 ./cmd/krot
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -o dist/krot-linux-arm64 ./cmd/krot
dist/krot-linux-amd64 --completion bash > dist/krot.bash
dist/krot-linux-amd64 --completion zsh > dist/_krot

for arch in amd64 arm64; do
    cp "dist/krot-linux-$arch" dist/krot
    for format in deb rpm; do
        GOARCH="$arch" VERSION="$version" "$nfpm_bin" package -f packaging/nfpm.yaml -p "$format" -t dist/
    done
done
rm dist/krot dist/krot.bash dist/_krot

scripts/smoke.sh ubuntu:24.04 dist amd64
scripts/smoke.sh almalinux:9 dist amd64
