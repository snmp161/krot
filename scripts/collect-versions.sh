#!/usr/bin/env bash
# Собирает реальные выводы версий компонентов AmneziaWG для golden-тестов (SPEC 3.1, задача T0).
# Компоненты собираются из исходников на закреплённых тегах внутри docker-контейнера;
# хост не изменяется, результат пишется в internal/awg/testdata/versions/.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
out="$root/internal/awg/testdata/versions"
mkdir -p "$out"

docker run --rm -e HOST_UID="$(id -u)" -e HOST_GID="$(id -g)" -v "$out:/out" golang:1.27.2-bookworm bash -euo pipefail -c '
cd /tmp
repo=https://github.com/amnezia-vpn

# amneziawg-tools, сборка из git. Makefile берёт версию из `git describe` (без --tags), а теги
# upstream lightweight, поэтому до v1.0.20260618 печатается заглушка из version.h (1.0.20210914).
for tag in v1.0.20240213 v1.0.20250901 v3.1.20260812; do
  git clone -q -c advice.detachedHead=false --depth 1 --branch "$tag" "$repo/amneziawg-tools" "tools-$tag"
  make -s -C "tools-$tag/src" >/dev/null 2>&1
  bin="tools-$tag/src/awg"; [ -x "$bin" ] || bin="tools-$tag/src/wg"
  "$bin" --version > "/out/awg-tools-$tag-git.txt"
done

# amneziawg-tools, сборка из tarball (без .git версия берётся из version.h)
tag=v1.0.20250901
mkdir "tarball-$tag"
curl -fsSL "$repo/amneziawg-tools/archive/refs/tags/$tag.tar.gz" | tar xz -C "tarball-$tag" --strip-components=1
make -s -C "tarball-$tag/src" >/dev/null 2>&1
bin="tarball-$tag/src/awg"; [ -x "$bin" ] || bin="tarball-$tag/src/wg"
"$bin" --version > "/out/awg-tools-$tag-tarball.txt"

# amneziawg-go, сборка make (версия = git describe) и plain go build
for tag in v0.2.15 v3.1.20260828; do
  git clone -q -c advice.detachedHead=false --depth 1 --branch "$tag" "$repo/amneziawg-go" "go-$tag"
  make -s -C "go-$tag" >/dev/null 2>&1
  "go-$tag/amneziawg-go" --version > "/out/amneziawg-go-$tag-make.txt"
done
# plain go build — в отдельном чистом клоне: make переписывает version.go
tag=v3.1.20260828
git clone -q -c advice.detachedHead=false --depth 1 --branch "$tag" "$repo/amneziawg-go" "go-$tag-plain"
(cd "go-$tag-plain" && go build -o amneziawg-go . )
"go-$tag-plain/amneziawg-go" --version > "/out/amneziawg-go-$tag-gobuild.txt"

chown -R "$HOST_UID:$HOST_GID" /out
'
echo "written to $out"
