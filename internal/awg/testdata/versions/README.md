# Golden-выводы версий AmneziaWG / WireGuard (SPEC 3.1, задача T0)

| Файл | Происхождение |
|---|---|
| `awg-tools-*-git.txt`, `awg-tools-*-tarball.txt` | реальный `awg --version`, собрано `scripts/collect-versions.sh` из тегов upstream |
| `amneziawg-go-*-make.txt`, `amneziawg-go-*-gobuild.txt` | реальный `amneziawg-go --version` (сборка `make` и `go build`), тот же скрипт |
| `wg-tools-host.txt` | реальный `wg --version` (Ubuntu 24.04, wireguard-tools 1.0.20210914) |
| `genl-nlctrl-host.txt` | реальный `genl ctrl get name nlctrl` (iproute2 6.1) — эталон формата |
| `genl-amneziawg-missing-host.txt` | реальный вывод `genl ctrl get name amneziawg` без модуля (exit 1) |
| `genl-amneziawg-v{1,2,3}.txt` | формат взят из `genl-nlctrl-host.txt`; имя и версия семейства — по исходникам модуля (`WG_GENL_VERSION`: 1 до v1.0.20241112, 2 с v1.0.20251004, 3 в v3.x) |
| `modinfo-wireguard-host.txt` | реальный `modinfo -F version wireguard` (пример строки `1.0.0`) |
| `module-*.txt` | содержимое `/sys/module/amneziawg/version` / `modinfo -F version amneziawg` по `src/version.h` тегов модуля; `1.0.0` — сборка `make`/DKMS (Makefile задаёт `WIREGUARD_VERSION = 1.0.0`) |

Наблюдения из реальных сборок:
- amneziawg-tools 1.x печатает заглушку `1.0.20210914` даже при сборке из git: Makefile вызывает
  `git describe` без `--tags`, а теги upstream lightweight. Реальная версия видна с v1.0.20260618 и в 3.x.
- amneziawg-go при `go build` печатает `0.0.20250522` (из `version.go`), при `make` — тег (`v3.1.20260828`).
