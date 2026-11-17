#!/usr/bin/env bash
# Smoke-тест пакета в чистом контейнере (SPEC 8.4): установка → krot --version →
# krot --help → проверка completion → удаление пакета.
# Использование: scripts/smoke.sh <образ> <каталог с пакетами> <amd64|arm64>
set -euo pipefail

image="$1" dir="$(cd "$2" && pwd)" arch="$3"
case "$image" in
    ubuntu:*|debian:*) pkg="$(cd "$dir" && ls krot_*_"$arch".deb)" ;;
    almalinux:*|rockylinux:*)
        rpmarch="$arch"; [ "$arch" = amd64 ] && rpmarch=x86_64; [ "$arch" = arm64 ] && rpmarch=aarch64
        pkg="$(cd "$dir" && ls krot-*."$rpmarch".rpm)" ;;
    *) echo "unsupported image $image" >&2; exit 2 ;;
esac

docker run --rm --platform "linux/$arch" -v "$dir:/pkg:ro" -e PKG="/pkg/$pkg" "$image" sh -euc '
    case "$PKG" in
        *.deb) dpkg -i "$PKG" ;;
        *.rpm) rpm -i "$PKG" ;;
    esac
    test "$(command -v krot)" = /usr/bin/krot
    krot --version
    krot --help >/dev/null
    krot --completion bash >/dev/null
    krot --completion zsh >/dev/null
    test -s /usr/share/bash-completion/completions/krot
    test -s /usr/share/zsh/site-functions/_krot
    bash -n /usr/share/bash-completion/completions/krot
    case "$PKG" in
        *.deb) dpkg -r krot ;;
        *.rpm) rpm -e krot ;;
    esac
    for path in /usr/bin/krot /usr/share/bash-completion/completions/krot /usr/share/zsh/site-functions/_krot; do
        if [ -e "$path" ]; then echo "$path still installed" >&2; exit 1; fi
    done
    echo "smoke OK: $PKG"
'
