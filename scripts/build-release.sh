#!/usr/bin/env bash
# SPEC 9.2, 9.5 T46: shared deterministic binary and DEB/RPM build.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

version="${VERSION:-$(awk '/^ *- semver:/ { print $3; exit }' packaging/changelog.yaml)}"
display_version="${DISPLAY_VERSION:-dev}"
arches="${ARCHES:-amd64 arm64}"
commit="${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo none)}"
SOURCE_DATE_EPOCH="${SOURCE_DATE_EPOCH:-$(git show -s --format=%ct HEAD 2>/dev/null || echo 0)}"
export SOURCE_DATE_EPOCH
build_date="$(date -u -d "@$SOURCE_DATE_EPOCH" +%Y-%m-%dT%H:%M:%SZ)"
DESCRIPTION="$(cat packaging/description.txt)"
export DESCRIPTION
nfpm_bin="$(go env GOPATH)/bin/nfpm"

rm -rf dist
mkdir -p dist
for arch in $arches; do
    CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -buildvcs=false -trimpath \
        -ldflags "-s -w -X main.version=$display_version -X main.commit=$commit -X main.date=$build_date" \
        -o "dist/krot-linux-$arch" ./cmd/krot
done
cp LICENSE THIRD_PARTY_NOTICES dist/

dist/krot-linux-amd64 --completion bash > dist/krot.bash
dist/krot-linux-amd64 --completion zsh > dist/_krot
bash -n dist/krot.bash
scripts/compress-man.sh
scripts/debian-changelog.sh
awk '!/^changelog:/' packaging/nfpm.yaml > dist/nfpm-deb.yaml

for arch in $arches; do
    cp "dist/krot-linux-$arch" dist/krot
    for format in deb rpm; do
        target="dist/"
        # Keep the established public DEB filename while the control version carries
        # the Debian revision. nFPM accepts a filename as target.
        config="packaging/nfpm.yaml"
        if [ "$format" = deb ]; then
            target="dist/krot_${version}_$arch.deb"
            config="dist/nfpm-deb.yaml"
        fi
        GOARCH="$arch" VERSION="$version" "$nfpm_bin" package -f "$config" -p "$format" -t "$target"
    done
done
rm dist/krot dist/krot.bash dist/_krot
rm dist/nfpm-deb.yaml dist/changelog.Debian.gz
rm -rf dist/man

(
    cd dist
    export LC_ALL=C
    sha256sum -- LICENSE THIRD_PARTY_NOTICES krot-linux-* *.deb *.rpm > SHA256SUMS
)
