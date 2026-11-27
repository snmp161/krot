#!/usr/bin/env bash
# SPEC 9.1: bounded smoke for every fuzz target discovered in the repository.
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root"

packages="$(go list ./...)"
for pkg in $packages; do
    targets="$(go test "$pkg" -list '^Fuzz')"
    for target in $targets; do
        case "$target" in
            Fuzz*) go test "$pkg" -run='^$' -fuzz="^${target}$" -fuzztime=10s ;;
        esac
    done
done
