#!/usr/bin/env bash
# SPEC 8.4, 9.4 T47: install, upgrade, reinstall and remove/purge via apt/dnf.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
work="$(mktemp -d)"
trap 'rm -rf "$work"' EXIT
cp -a "$root" "$work/source"
mkdir -p "$work/old" "$work/new"

for version in 0.0.1 0.0.2; do
    rm -rf "$work/source/dist"
    (
        cd "$work/source"
        ARCHES=amd64 VERSION="$version" DISPLAY_VERSION="v$version" COMMIT=none ./scripts/build-release.sh
    )
    if [ "$version" = 0.0.1 ]; then target="$work/old"; else target="$work/new"; fi
    cp "$work/source/dist/krot_${version}_amd64.deb" "$work/source/dist/krot-${version}-1.x86_64.rpm" "$target/"
done

docker run --rm -v "$work/old:/old:ro" -v "$work/new:/new:ro" ubuntu:24.04 sh -euc '
    check_files() {
        case "$(krot --version)" in *"$1"*) ;; *) return 1 ;; esac
        for path in /usr/bin/krot /usr/share/doc/krot/README.md \
            /usr/share/doc/krot/LICENSE /usr/share/doc/krot/THIRD_PARTY_NOTICES \
            /usr/share/doc/krot/copyright /usr/share/doc/krot/changelog.Debian.gz \
            /usr/share/man/man1/krot.1.gz /usr/share/bash-completion/completions/krot \
            /usr/share/zsh/site-functions/_krot /usr/share/lintian/overrides/krot; do
            test -s "$path"
        done
    }
    export DEBIAN_FRONTEND=noninteractive
    printf "%s\n" "path-include=/usr/share/man/*" "path-include=/usr/share/doc/krot/*" > /etc/dpkg/dpkg.cfg.d/krot-test
    apt-get install -y --no-install-recommends /old/*.deb
    test "$(dpkg-query -W -f="\${Version}" krot)" = 0.0.1-1
    check_files v0.0.1
    apt-get install -y --no-install-recommends /new/*.deb
    test "$(dpkg-query -W -f="\${Version}" krot)" = 0.0.2-1
    check_files v0.0.2
    apt-get install -y --reinstall --no-install-recommends /new/*.deb
    check_files v0.0.2
    apt-get purge -y krot
    for path in /usr/bin/krot /usr/share/doc/krot/README.md \
        /usr/share/doc/krot/LICENSE /usr/share/doc/krot/THIRD_PARTY_NOTICES \
        /usr/share/doc/krot/copyright /usr/share/doc/krot/changelog.Debian.gz \
        /usr/share/man/man1/krot.1.gz /usr/share/bash-completion/completions/krot \
        /usr/share/zsh/site-functions/_krot /usr/share/lintian/overrides/krot; do
        test ! -e "$path"
    done
'

docker run --rm -v "$work/old:/old:ro" -v "$work/new:/new:ro" almalinux:10 sh -euc '
    check_files() {
        case "$(krot --version)" in *"$1"*) ;; *) return 1 ;; esac
        for path in /usr/bin/krot /usr/share/doc/krot/README.md \
            /usr/share/doc/krot/LICENSE /usr/share/doc/krot/THIRD_PARTY_NOTICES \
            /usr/share/man/man1/krot.1.gz /usr/share/bash-completion/completions/krot \
            /usr/share/zsh/site-functions/_krot; do
            test -s "$path"
        done
    }
    dnf install -y -q --setopt=install_weak_deps=False /old/*.rpm
    test "$(rpm -q --queryformat "%{VERSION}-%{RELEASE}" krot)" = 0.0.1-1
    check_files v0.0.1
    dnf upgrade -y -q --setopt=install_weak_deps=False /new/*.rpm
    test "$(rpm -q --queryformat "%{VERSION}-%{RELEASE}" krot)" = 0.0.2-1
    check_files v0.0.2
    dnf reinstall -y -q --setopt=install_weak_deps=False /new/*.rpm
    check_files v0.0.2
    dnf remove -y -q krot
    for path in /usr/bin/krot /usr/share/doc/krot/README.md \
        /usr/share/doc/krot/LICENSE /usr/share/doc/krot/THIRD_PARTY_NOTICES \
        /usr/share/man/man1/krot.1.gz /usr/share/bash-completion/completions/krot \
        /usr/share/zsh/site-functions/_krot; do
        test ! -e "$path"
    done
'

echo "package lifecycle OK: apt and dnf install/upgrade/reinstall/remove"
