#!/usr/bin/env bash
# Запускает интеграционные тесты (SPEC 8.3) в privileged docker-контейнере с systemd.
# У контейнера свой сетевой namespace, тесты создают внутри него дополнительные netns,
# поэтому сеть хоста не затрагивается. Ядро общее: при создании wg-интерфейса ядро хоста
# может подгрузить модуль wireguard.
# Аргументы передаются тестовому бинарю, например: -test.run TestFullCycle
set -euo pipefail

root="$(cd "$(dirname "$0")/.." && pwd)"
work="$(mktemp -d)"
name="krot-it-$$"
cleanup() {
    docker rm -f "$name" >/dev/null 2>&1 || true
    rm -rf "$work"
}
trap cleanup EXIT

cd "$root"
CGO_ENABLED=0 go build -o "$work/krot" ./cmd/krot
CGO_ENABLED=0 go test -c -tags integration -o "$work/integration.test" ./test/integration
chmod 0755 "$work" "$work/krot" "$work/integration.test"

docker build -q -t krot-it -f test/integration/Dockerfile test/integration >/dev/null
docker run -d --name "$name" --privileged --cgroupns=private \
    --tmpfs /run --tmpfs /run/lock --tmpfs /tmp \
    -v /lib/modules:/lib/modules:ro -v "$work:/work:ro" \
    krot-it >/dev/null

# Ждём, пока systemd в контейнере поднимется (degraded тоже годится).
for _ in $(seq 1 60); do
    state="$(docker exec "$name" systemctl is-system-running 2>/dev/null || true)"
    case "$state" in running|degraded) break ;; esac
    sleep 1
done

docker exec -e KROT_BIN=/work/krot "$name" \
    /work/integration.test -test.v -test.count=1 -test.timeout=30m "$@"
