# krot

`krot` — консольная утилита для управления туннелями **WireGuard** и **AmneziaWG**
(1.x, 2.0, 3.1) на Linux-серверах: создание и удаление интерфейсов, управление клиентами
(peer'ами), выдача клиентских конфигов, QR-кодов и ключей AmneziaVPN (`vpn://…`).

- Один статический бинарь без зависимостей (Go, `CGO_ENABLED=0`).
- Операции с клиентами **не перезапускают и не пересоздают интерфейс**: только точечные
  `wg|awg set <if> peer …` — остальные клиенты ничего не замечают.
- Конфиг интерфейса — единственный источник правды: никаких отдельных баз, метаданные krot
  хранятся в комментариях, чужие конфиги читаются и не портятся.
- Предсказуемый вывод: таблицы для людей, `--json` и коды выхода для автоматизации.
- NAT, forwarding и firewall krot не настраивает.

Полное техническое задание — [`docs/SPEC.md`](docs/SPEC.md).

## Требования

| | |
|---|---|
| ОС | Ubuntu 22.04 / 24.04 / 26.04, Debian 12 / 13, AlmaLinux (Rocky, RHEL) 9 / 10 |
| Архитектуры | `amd64`, `arm64` |
| Для `--type wg` | `wireguard-tools` (`wg`, `wg-quick`), модуль ядра WireGuard |
| Для `--type awg*` | `amneziawg-tools` (`awg`, `awg-quick`) и модуль ядра AmneziaWG **или** `amneziawg-go` |
| Всегда | `iproute2` (`ip`), `systemd` |
| Права | root (кроме `--help` и `--version`) |

krot сам ничего из этого не устанавливает. Если программы нет — ошибка с кодом `6` и
названием пакета.

### Версии AmneziaWG

Перед `--create` krot проверяет, что установленная AmneziaWG поддерживает запрошенный тип:

| Тип | amneziawg-tools | модуль ядра | netlink-семейство | amneziawg-go |
|---|---|---|---|---|
| `awg` (1.0) | 1.0.20240213 | — | ≥ 1 | 0.1.0 |
| `awg` с `--i1` (1.5) | 1.0.20250901 | 1.0.20251009 | ≥ 2 | 0.2.13 |
| `awg2` (2.0) | 1.0.20250901 | 1.0.20251009 | ≥ 2 | 0.2.15 |
| `awg3` (3.1) | 3.1.20260812 | 3.1.20260812 | ≥ 3 | 3.1.20260812 |

Некоторые сборки сообщают версию-«заглушку» (например, `awg --version` у tools 1.x почти
всегда печатает `1.0.20210914`, модуль, собранный `make`, — `1.0.0`). Такие версии дают
предупреждение, а не ошибку; ошибка `6` — только если версия надёжно распознана и ниже нужной.
Если версия на самом деле старая, не запустится unit, и krot откатит изменения.

## Установка

Устанавливайте из пакетов или бинарей со страницы релизов
<https://github.com/snmp161/krot/releases>, обязательно проверяя контрольную сумму.

```sh
VERSION=0.1.0
BASE=https://github.com/snmp161/krot/releases/download/v$VERSION

# скачать пакет и список контрольных сумм
wget "$BASE/krot_${VERSION}_amd64.deb" "$BASE/SHA256SUMS"
# проверить: должно быть "krot_..._amd64.deb: OK"
sha256sum --check --ignore-missing SHA256SUMS
# установить
sudo dpkg -i "krot_${VERSION}_amd64.deb"
```

RPM-системы (AlmaLinux, Rocky, RHEL):

```sh
wget "$BASE/krot-${VERSION}-1.x86_64.rpm" "$BASE/SHA256SUMS"
sha256sum --check --ignore-missing SHA256SUMS
sudo rpm -i "krot-${VERSION}-1.x86_64.rpm"
```

Бинарь без пакета: скачайте `krot-linux-amd64` (или `krot-linux-arm64`) и `SHA256SUMS`,
проверьте так же и установите `sudo install -m 0755 krot-linux-amd64 /usr/bin/krot`.

Для `arm64` используйте файлы `*_arm64.deb`, `*.aarch64.rpm`, `krot-linux-arm64`.

## Использование

```
krot <действие> [флаги]
```

Ровно одно действие за запуск; флаги только в длинной форме `--flag value` или `--flag=value`.
Флаг, не подходящий к действию, — ошибка `2`. Полная справка — `krot --help`.

### Туннели

```sh
# создать AmneziaWG 2.0 (dual-stack), имя выберется автоматически: awg0, awg1, …
sudo krot --create --type awg2 --ip 10.8.0.1/24,fd00:8::1/64 --port 51820 \
          --endpoint vpn.example.com --dns 1.1.1.1,9.9.9.9

# WireGuard с явным именем и MTU
sudo krot --create --type wg --interface wg0 --ip 10.9.0.1/24 --port 51821 \
          --endpoint 203.0.113.10 --mtu 1420

# AmneziaWG 1.5 с пакетом-сигнатурой I1 (krot проверяет только синтаксис CPS)
sudo krot --create --type awg --ip 10.10.0.1/24 --port 51822 --endpoint vpn.example.com \
          --i1 '<b 0xf6ab3267fa><r 16>'

# AmneziaWG 3.1 (Header Protection, RandomTrailers, DisableCookies включены всегда)
sudo krot --create --type awg3 --ip 10.11.0.1/24 --port 51823 --endpoint vpn.example.com

sudo krot --list                         # все туннели
sudo krot --list --type awg2 --json      # фильтры --type/--interface/--ip/--port, по И
sudo krot --down --interface awg0        # systemctl stop awg-quick@awg0
sudo krot --up --interface awg0          # systemctl start; повторный --up — «already up», код 0
sudo krot --delete --interface awg0      # спросит подтверждение, если есть клиенты
sudo krot --delete --interface awg0 --yes
```

Пример `--list`:

```
INTERFACE  TYPE  VERSION  ADDRESS                   PORT   STATUS  CLIENTS
awg0       awg2  2.0      10.8.0.1/24,fd00:8::1/64  51820  up      12 (1 locked)
wg0        wg    -        10.9.0.1/24               51821  down    3
```

MTU по умолчанию: `wg` — 1420, `awg`/`awg2` — 1376, `awg3` — 1280. Параметры обфускации
(Jc, S1–S4, H1–H4, …) генерируются автоматически в безопасных диапазонах и проверяются
валидатором (раздел 7 ТЗ).

### Клиенты

```sh
# добавить клиента: адреса выдаются автоматически (первый свободный /32 и /128),
# клиентский конфиг печатается ОДИН раз — приватный ключ клиента нигде не хранится
sudo krot --add-client --interface awg0 --id phone-sergei --allowed-ips 0.0.0.0/0,::/0

# с PresharedKey и QR-кодом в терминале
sudo krot --add-client --interface awg0 --id laptop --allowed-ips 0.0.0.0/0 --psk --qr

# дополнительно ключ для приложения AmneziaVPN: строка vpn://… после конфига
sudo krot --add-client --interface awg0 --id tablet --allowed-ips 0.0.0.0/0,::/0 --amnezia-key

# конкретный адрес клиента
sudo krot --add-client --interface awg0 --id router --allowed-ips 10.8.0.0/24 --ip 10.8.0.50/32

# чужой конфиг без метаданных krot: endpoint передаётся явно (в конфиг сервера не пишется)
sudo krot --add-client --interface office --id guest --allowed-ips 0.0.0.0/0 --endpoint 203.0.113.10

# свой DNS для одного клиента (в конфиг сервера не пишется)
sudo krot --add-client --interface awg0 --id kid-phone --allowed-ips 0.0.0.0/0 --dns 94.140.14.15

sudo krot --list-client --interface awg0
sudo krot --list-client --interface awg0 --id phone-sergei --json
sudo krot --lock-client   --interface awg0 --id laptop    # повторно — «already locked», код 0
sudo krot --unlock-client --interface awg0 --id laptop
sudo krot --delete-client --interface awg0 --id laptop
```

`--allowed-ips` — это `AllowedIPs` в конфиге **клиента** (что клиент направляет в туннель).
DNS клиента: `--dns`, если задан; иначе `# krot:dns` интерфейса (`--dns` при `--create`); иначе
строки `DNS` из `[Interface]` конфига сервера; если DNS нет нигде — строки `DNS` в конфиге клиента нет.
Адреса клиентам выдаются из подсети интерфейса, кроме адреса сети, broadcast и адреса сервера.
Подсети `/31` и `/127` (RFC 3021, RFC 6164) тоже поддерживаются: в них оба адреса рабочие, поэтому
туннель «сервер + один клиент» можно создать так: `--create … --ip 10.8.0.0/31,fd00:8::/127`.
Сохраните выданный конфиг сразу: повторно получить его нельзя, только удалить клиента и
добавить заново. QR выводится полублоками Unicode и рассчитан на тёмный фон терминала.

`--amnezia-key` печатает после конфига ключ AmneziaVPN — одну строку `vpn://…`, которую
вставляют в приложении AmneziaVPN («Вставить ключ»), например, переслав через мессенджер. Внутри
тот же клиентский конфиг: `vpn://` + base64url без `=` от `qCompress` из Qt (4 байта длины
big-endian + zlib). Ключ работает для всех типов, включая `wg`. QR-код при `--qr` содержит текст
конфига — его сканируют AmneziaVPN и AmneziaWG, а для типа `wg` — и WireGuard.

Пример `--list-client`:

```
ID            PUBLIC KEY  ADDRESS                    STATUS  LAST HANDSHAKE  RX       TX
phone-sergei  aB3d…Xy9=   10.8.0.2/32,fd00:8::2/128  active  1m ago          12.4MiB  1.1GiB
laptop-old    Qw7e…Zz1=   10.8.0.3/32                locked  -               -        -
```

### JSON

`--list --json` — массив объектов `interface`, `type`, `version`, `addresses[]`, `port`,
`status`, `clients_total`, `clients_locked`. `--list-client --json` — массив объектов `id`,
`public_key`, `addresses[]`, `status`, `latest_handshake` (unix time или `null`), `rx_bytes`,
`tx_bytes`, `endpoint` (или `null`), `created` (или `null` у чужих peer'ов). Пустой результат — `[]`.

## Коды выхода

| Код | Значение |
|---|---|
| 0 | успех |
| 1 | внутренняя или неожиданная ошибка (в т.ч. не удалось применить изменение к работающему интерфейсу — конфиг откатан) |
| 2 | ошибка использования CLI (флаги; удаление с клиентами без `--yes`, когда stdin не терминал) |
| 3 | ошибка валидации значений (в т.ч. нет endpoint, «config too large for QR») |
| 4 | объект не найден (интерфейс, клиент) |
| 5 | конфликт: уже существует, адрес или порт занят, нет свободных адресов |
| 6 | окружение: нет root, нет утилит, неподходящая версия AmneziaWG |
| 7 | не удалось получить блокировку `/run/lock/krot.lock` за 10 секунд |
| 10 | операция отменена пользователем |

Вывод — в stdout, ошибки и предупреждения — в stderr с префиксом `krot:`.

## Где и как хранятся данные

| Тип | Конфиг | systemd unit | Утилита |
|---|---|---|---|
| `wg` | `/etc/wireguard/<if>.conf` | `wg-quick@<if>` | `wg` |
| `awg`, `awg2`, `awg3` | `/etc/amnezia/amneziawg/<if>.conf` | `awg-quick@<if>` | `awg` |

Метаданные — комментарии, которые `wg-quick`/`awg-quick` игнорируют:

```ini
# krot:type=awg2
# krot:endpoint=vpn.example.com
# krot:dns=1.1.1.1,9.9.9.9
[Interface]
PrivateKey = ...
Address = 10.8.0.1/24, fd00:8::1/64
ListenPort = 51820
MTU = 1376
Jc = 6
...
# PreUp =
# PostUp =
# PreDown =
# PostDown =

# krot:id=phone-sergei
# krot:created=2026-10-17T01:30:00Z
[Peer]
PublicKey = ...
AllowedIPs = 10.8.0.2/32, fd00:8::2/128

# krot:id=laptop-old
# krot:created=2026-10-01T10:00:00Z
# krot:locked=2026-10-16T12:00:00Z
#~ [Peer]
#~ PublicKey = ...
#~ AllowedIPs = 10.8.0.3/32
```

- Заблокированный клиент — блок `[Peer]`, закомментированный префиксом `#~ `.
- Закомментированные хуки `PreUp`/`PostUp`/`PreDown`/`PostDown` можно раскомментировать и заполнить
  (например, правилами NAT) — krot их не трогает.
- Конфиги без метаданных krot (созданные вручную) поддерживаются: `--list` их показывает,
  `--list-client` выводит их peer'ов с id `-` (к таким peer'ам операции по `--id` не применимы),
  а при любой записи все чужие строки, комментарии и порядок сохраняются байт-в-байт.
- Запись атомарная (временный файл → `fsync` → `rename` → `fsync` каталога), права `0600`.
- Все изменяющие действия берут эксклюзивный `flock` на `/run/lock/krot.lock` — параллельные
  запуски (например, Salt и человек) не теряют изменений.
- Бэкапы krot не делает.

## Ограничения

- NAT, forwarding, firewall, sysctl — вне рамок krot (используйте хуки `PostUp`/`PostDown`).
- Установка WireGuard/AmneziaWG, миграция туннеля между типами, генерация сигнатур `I1`–`I5`
  (только приём готовой строки через `--i1`) не поддерживаются.
- Приватные ключи клиентов не хранятся, повторная выдача клиентского конфига невозможна.

## Разработка

Нужен Go из `go.mod` (сейчас 1.27.2) и docker для интеграционных тестов.

```sh
make check        # gofmt, go vet (с тегом integration тоже), go test -race ./...
make build        # статический bin/krot (CGO_ENABLED=0), версия из git describe
make build-arm64
make integration  # интеграционные тесты в privileged docker-контейнере с systemd
```

Правила работы в репозитории — [`AGENTS.md`](AGENTS.md): ТЗ первично, сначала тесты, коммит на
каждую задачу.

- **Unit-тесты** не требуют root и не трогают систему: каталоги подменяются временными,
  внешние команды (`wg`, `awg`, `systemctl`, `ip`, …) — фейковым исполнителем с записью вызовов.
  Отдельные тесты проверяют, что клиентские операции вызывают только `wg|awg show` и
  `wg|awg set <if> peer …`.
- **Golden-файлы** лежат в `testdata/` пакетов. Сгенерированные выводы (`internal/app`,
  `internal/qr`) обновляются только осознанно — `go test ./internal/app ./internal/qr -update`,
  с объяснением изменений в коммите; остальные golden-файлы написаны вручную по ТЗ.
- **Интеграционные тесты** (`test/integration`, build tag `integration`) требуют root и systemd и
  работают только в отдельных network namespace: unit'ы тестовых интерфейсов помещаются в netns
  через drop-in `NetworkNamespacePath`. Проверяются полный цикл для всех типов, незаметность
  клиентских операций (непрерывный ping без потерь, неизменный ifindex и handshake), IPv6,
  чужие конфиги и параллельные запуски. Локально — `scripts/integration-docker.sh`, в CI — под `sudo`.
- `scripts/collect-versions.sh` пересобирает реальные выводы версий AmneziaWG для golden-тестов.
- **CI** (`.github/workflows/ci.yml`): gofmt, vet, unit-тесты с `-race` и покрытием, интеграционные
  тесты, сборка `amd64`/`arm64`. **Релиз** (`release.yml`, тег `v*`): бинари, пакеты `.deb`/`.rpm`
  (nfpm, `packaging/nfpm.yaml`), smoke-тесты пакетов на всех ОС (`scripts/smoke.sh`), GitHub Release
  с `SHA256SUMS`.

## Лицензия

[Apache License 2.0](LICENSE). Copyright 2026 Sergey Ovchinnikov.
