#!/usr/bin/env bash
# SPEC 9.1 T36: сборка DEB/RPM для обеих архитектур и representative smoke на amd64.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

DESCRIPTION="$(cat packaging/description.txt)"
export DESCRIPTION
./scripts/build-release.sh

scripts/smoke.sh ubuntu:24.04 dist amd64
scripts/smoke.sh almalinux:9 dist amd64
