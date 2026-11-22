#!/usr/bin/env bash
# SPEC 9.4 T47: policy checks on every built DEB/RPM and both architectures.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
dir="$(cd "$1" && pwd)"

docker run --rm -v "$dir:/pkg:ro" debian:13 sh -euc '
    export DEBIAN_FRONTEND=noninteractive
    apt-get update -qq
    apt-get install -y -qq --no-install-recommends lintian >/dev/null
    lintian --fail-on error,warning /pkg/*.deb
'

docker run --rm -v "$dir:/pkg:ro" -v "$root/packaging/rpmlintrc:/rpmlintrc:ro" almalinux:10 sh -euc '
    dnf install -y -q epel-release >/dev/null
    dnf install -y -q --setopt=install_weak_deps=False rpmlint >/dev/null
    rpmlint --strict -r /rpmlintrc /pkg/*.rpm
'
