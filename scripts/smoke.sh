#!/usr/bin/env bash
# Smoke-тест пакета в чистом контейнере (SPEC 8.4, 9.4): установка → проверка
# программы, metadata, лицензий, completion, changelog и man → удаление.
# Использование: scripts/smoke.sh <образ> <каталог с пакетами> <amd64|arm64>
set -euo pipefail

image="$1" dir="$(cd "$2" && pwd)" arch="$3"
root="$(cd "$(dirname "$0")/.." && pwd)"
expected_version="$(awk '/^[0-9]+\.[0-9]+\.[0-9]+([-+][0-9A-Za-z.-]+)?$/ { print; exit }' "$root/CHANGELOG.md")"
expected_notes="$(awk -v version="$expected_version" '
    $0 == version { current=1; next }
    current && /^[0-9]+\.[0-9]+\.[0-9]+([-+][0-9A-Za-z.-]+)?$/ { exit }
    current && /^\* / {
        sub(/^\* /, "")
        sub(/ \([0-9a-f]+\)$/, "")
        print
    }
' "$root/CHANGELOG.md")"
if [ -z "$expected_version" ] || [ -z "$expected_notes" ]; then
    echo "current release notes are missing or empty" >&2
    exit 1
fi
case "$image" in
    ubuntu:*|debian:*) pkg="$(cd "$dir" && ls krot_*_"$arch".deb)" ;;
    almalinux:*|rockylinux:*)
        rpmarch="$arch"; [ "$arch" = amd64 ] && rpmarch=x86_64; [ "$arch" = arm64 ] && rpmarch=aarch64
        pkg="$(cd "$dir" && ls krot-*."$rpmarch".rpm)" ;;
    *) echo "unsupported image $image" >&2; exit 2 ;;
esac

docker run --rm --platform "linux/$arch" -v "$dir:/pkg:ro" -v "$root/packaging:/src:ro" \
    -e PKG="/pkg/$pkg" -e EXPECTED_VERSION="$expected_version" -e EXPECTED_NOTES="$expected_notes" \
    "$image" sh -euc '
    case "$PKG" in
        *.deb) dpkg "--path-include=/usr/share/man/*" "--path-include=/usr/share/doc/krot/*" -i "$PKG" ;;
        *.rpm) rpm -i "$PKG" ;;
    esac
    test "$(command -v krot)" = /usr/bin/krot
    krot --version
    for locale in C ru_RU.UTF-8 zh_CN.UTF-8 zh_TW.UTF-8 es_MX.UTF-8 pt_BR.UTF-8 ko_KR.UTF-8 ja_JP.UTF-8 ar_SA.UTF-8 fr_FR.UTF-8 de_DE.UTF-8; do
        LC_ALL="$locale" krot --help >/dev/null
    done
    help_ru="$(LC_ALL=ru_RU.UTF-8 krot --help)"
    case "$help_ru" in *"управление туннелями WireGuard"*) ;; *) echo "Russian help missing" >&2; exit 1 ;; esac

    expected_description="$(cat /src/description.txt)"
    case "$PKG" in
        *.deb)
            actual_description="$(dpkg-query -W -f="\${Description}" krot | sed -e "s/^ //" -e "s/^\.$//")"
            actual_version="$(dpkg-query -W -f="\${Version}" krot)"
            expected_package_version="$EXPECTED_VERSION"
            case "$expected_package_version" in
                *-*) expected_package_version="${expected_package_version%%-*}~${expected_package_version#*-}" ;;
            esac
            test "$actual_version" = "$expected_package_version-1"
            changelog="$(gzip -dc /usr/share/doc/krot/changelog.Debian.gz | sed -e "s/^[[:space:]]*//" -e "s/[[:space:]]*$//" | tr "\n" " ")"
            test -s /usr/share/doc/krot/copyright
            doc_paths="/usr/share/doc/krot/copyright /usr/share/doc/krot/changelog.Debian.gz /usr/share/lintian/overrides/krot"
            ;;
        *.rpm)
            actual_description="$(rpm -q --queryformat "%{DESCRIPTION}" krot)"
            actual_version="$(rpm -q --queryformat "%{VERSION}" krot)"
            test "$actual_version" = "$EXPECTED_VERSION"
            changelog="$(rpm -q --changelog krot | sed -e "s/^[[:space:]]*//" -e "s/[[:space:]]*$//" | tr "\n" " ")"
            doc_paths=""
            ;;
    esac
    if [ "$actual_description" != "$expected_description" ]; then
        printf "description mismatch\n--- expected ---\n%s\n--- actual ---\n%s\n" "$expected_description" "$actual_description" >&2
        exit 1
    fi
    printf "%s\n" "$EXPECTED_NOTES" | while IFS= read -r text; do
        test -n "$text" || continue
        case "$changelog" in *"$text"*) ;; *) printf "changelog misses %s\n" "$text" >&2; exit 1 ;; esac
    done
    for path in /usr/share/doc/krot/README.md /usr/share/doc/krot/LICENSE /usr/share/doc/krot/THIRD_PARTY_NOTICES; do
        test -s "$path"
    done

    krot --completion bash >/dev/null
    krot --completion zsh >/dev/null
    test -s /usr/share/bash-completion/completions/krot
    test -s /usr/share/zsh/site-functions/_krot
    man_paths="/usr/share/man/man1/krot.1.gz /usr/share/man/ru/man1/krot.1.gz /usr/share/man/zh_CN/man1/krot.1.gz /usr/share/man/zh_TW/man1/krot.1.gz /usr/share/man/es/man1/krot.1.gz /usr/share/man/pt_BR/man1/krot.1.gz /usr/share/man/ko/man1/krot.1.gz /usr/share/man/ja/man1/krot.1.gz /usr/share/man/ar/man1/krot.1.gz /usr/share/man/fr/man1/krot.1.gz /usr/share/man/de/man1/krot.1.gz"
    for path in $man_paths; do
        test -s "$path"
        gzip -t "$path"
    done
    bash -n /usr/share/bash-completion/completions/krot
    case "$PKG" in
        *.deb) dpkg -r krot ;;
        *.rpm) rpm -e krot ;;
    esac
    for path in /usr/bin/krot /usr/share/bash-completion/completions/krot /usr/share/zsh/site-functions/_krot \
        /usr/share/doc/krot/LICENSE /usr/share/doc/krot/THIRD_PARTY_NOTICES $doc_paths $man_paths; do
        if [ -e "$path" ]; then echo "$path still installed" >&2; exit 1; fi
    done
    echo "smoke OK: $PKG"
'
