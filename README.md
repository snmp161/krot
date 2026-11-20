# krot

**English** | [Русский](README.ru.md) | [简体中文](README.zh.md) | [繁體中文](README.zh-Hant.md) | [Español](README.es.md) | [Português](README.pt.md) | [한국어](README.ko.md) | [日本語](README.ja.md) | [العربية](README.ar.md) | [Français](README.fr.md) | [Deutsch](README.de.md)

`krot` is a command-line utility for managing **WireGuard** and **AmneziaWG**
(1.x, 2.0, 3.1) tunnels on Linux servers: creating and deleting interfaces,
managing clients (peers), and generating client configs, QR codes, and
AmneziaVPN keys (`vpn://…`).

The name **krot** means "mole" in Russian and several other Slavic languages —
an animal that digs tunnels.

- A single static binary with no dependencies (Go, `CGO_ENABLED=0`).
- Client operations **never restart or recreate the interface**: only targeted
  `wg|awg set <if> peer …` calls — other clients are unaffected.
- The interface config is the single source of truth: no separate databases;
  krot metadata is stored in comments, and configs not created by krot are
  read and preserved.
- Predictable output: tables for humans, `--json` and exit codes for automation.
- krot does not configure NAT, forwarding, or firewall rules.

Full technical specification: [`docs/SPEC.md`](docs/SPEC.md).

## Requirements

| | |
|---|---|
| OS | Ubuntu 22.04 / 24.04 / 26.04, Debian 12 / 13, AlmaLinux (Rocky, RHEL) 9 / 10 |
| Architectures | `amd64`, `arm64` |
| For `--type wg` | `wireguard-tools` (`wg`, `wg-quick`), WireGuard kernel module |
| For `--type awg*` | `amneziawg-tools` (`awg`, `awg-quick`) and AmneziaWG kernel module **or** `amneziawg-go` |
| Always | `iproute2` (`ip`), `systemd` |
| Privileges | root (except `--help`, `--version`, `--doctor`, `--udp-checker`; `--udp-listener` requires root only for ports `<1024` or when `/proc/net/udp{,6}` is unreadable) |

krot does not install any of these itself. If a required program is missing,
it exits with code `6` and names the package to install.

### AmneziaWG versions

Before `--create`, krot checks whether the installed AmneziaWG supports the
requested type:

| Type | amneziawg-tools | kernel module | netlink family | amneziawg-go |
|---|---|---|---|---|
| `awg` (1.0) | 1.0.20240213 | — | ≥ 1 | 0.1.0 |
| `awg` with `--i1` (1.5) | 1.0.20250901 | 1.0.20251009 | ≥ 2 | 0.2.13 |
| `awg2` (2.0) | 1.0.20250901 | 1.0.20251009 | ≥ 2 | 0.2.15 |
| `awg3` (3.1) | 3.1.20260812 | 3.1.20260812 | ≥ 3 | 3.1.20260812 |

Some builds report a placeholder version (for example, `awg --version` on
1.x tools almost always prints `1.0.20210914`, and a module built with `make`
reports `1.0.0`). Such versions produce a warning, not an error; code `6` is
returned only when the version is reliably detected and too old. If the version
is actually too old, the unit will fail to start and krot will roll back the
changes.

## Installation

Install from packages or binaries on the releases page
<https://github.com/snmp161/krot/releases>, and always verify the checksum.

```sh
VERSION=0.7.2
BASE=https://github.com/snmp161/krot/releases/download/v$VERSION

# download the package and the checksum list
wget "$BASE/krot_${VERSION}_amd64.deb" "$BASE/SHA256SUMS"
# verify: should print "krot_..._amd64.deb: OK"
sha256sum --check --ignore-missing SHA256SUMS
# install
sudo dpkg -i "krot_${VERSION}_amd64.deb"
```

RPM-based systems (AlmaLinux, Rocky, RHEL):

```sh
wget "$BASE/krot-${VERSION}-1.x86_64.rpm" "$BASE/SHA256SUMS"
sha256sum --check --ignore-missing SHA256SUMS
sudo rpm -i "krot-${VERSION}-1.x86_64.rpm"
```

Binary without a package: download `krot-linux-amd64` (or `krot-linux-arm64`)
and `SHA256SUMS`, verify the same way, then install with
`sudo install -m 0755 krot-linux-amd64 /usr/bin/krot`.

For `arm64`, use the `*_arm64.deb`, `*.aarch64.rpm`, and `krot-linux-arm64` files.

## Usage

```
krot <action> [flags]
```

Exactly one action per run; flags use only the long form `--flag value` or
`--flag=value`. A flag that doesn't apply to the action is an error (code `2`).
Full help: `krot --help`.

### Tunnels

```sh
# create AmneziaWG 2.0 (dual-stack); name is auto-selected: awg0, awg1, …
sudo krot --create --type awg2 --ip 10.8.0.1/24,fd00:8::1/64 --port 51820 \
          --endpoint vpn.example.com --dns 1.1.1.1,9.9.9.9

# WireGuard with an explicit name and MTU
sudo krot --create --type wg --interface wg0 --ip 10.9.0.1/24 --port 51821 \
          --endpoint 203.0.113.10 --mtu 1420

# AmneziaWG 1.5 with an I1 signature packet (krot validates CPS syntax and length — up to 2048 chars)
sudo krot --create --type awg --ip 10.10.0.1/24 --port 51822 --endpoint vpn.example.com \
          --i1 '<b 0xf6ab3267fa><r 16>'

# AmneziaWG 3.1 (Header Protection, RandomTrailers, DisableCookies are always enabled)
sudo krot --create --type awg3 --ip 10.11.0.1/24 --port 51823 --endpoint vpn.example.com

sudo krot --list                         # all tunnels
sudo krot --list --type awg2 --json      # filters --type/--interface/--ip/--port, ANDed
sudo krot --down --interface awg0        # systemctl stop awg-quick@awg0
sudo krot --up --interface awg0          # systemctl start; repeated --up prints "already up", code 0
sudo krot --delete --interface awg0      # asks for confirmation if clients exist
sudo krot --delete --interface awg0 --yes
```

### Diagnostics

```sh
krot --doctor   # works without root: tools, kernel modules, AmneziaWG versions, directories
```

Output is a `CHECK/STATUS/DETAIL` table with `ok` / `warn` / `fail` statuses.
Exit code is `0` if there are no `fail` entries, otherwise `6`. It changes
nothing and takes no lock.

#### UDP port diagnostics

`krot` can check which UDP ports actually pass between two hosts. This is
useful when a hosting provider or its uplink filters some ports (for example,
due to DDoS rules).

Start the listener on the server:

```sh
# single port
sudo krot --udp-listener --port 51820

# port range (up to 1024 ports, one socket per port)
sudo krot --udp-listener --port 51820:51830
```

The listener responds to three datagram types:
- `ping SEQ:N` → `pong SEQ:N`;
- `ECHO <text>` → `<text>`;
- `TIME` → current Unix time in milliseconds.

Run the checker on the client:

```sh
# check a single port
krot --udp-checker --address 203.0.113.1 --port 51820

# check a range
krot --udp-checker --address 203.0.113.1 --port 51820:51830

# increase echo payload size (1–1400 bytes, default 64) — helps find the best MTU
krot --udp-checker --address 203.0.113.1 --port 51820 --echo-size 1400

# set the worker count and show progress
krot --udp-checker --address 203.0.113.1 --port 51820:51830 --workers 64 --verbose
```

By default, the checker sends 3 packets of each type per port (`--count`,
1–100) and waits for a response within `--timeout` (default 2s). It checks
32 ports concurrently; use `--workers` to set 1–100 workers. `--verbose` logs
the range and per-port results to stderr as they complete; on the listener,
it logs client addresses and request types. Output:

```
PORT     STATUS   PING    ECHO    TIME (min/avg/max)   DETAIL
51820    ok       3/3     3/3     12/24/45 ms          -
51821    filtered 0/3     0/3     -                    timeout
```

Statuses:
- `ok` — all ping and echo packets received valid responses;
- `filtered` — at least one packet timed out (the port is likely filtered);
- `error` — an invalid response was received.

With `--json`, output is an array of objects with fields `port`, `status`,
`ping_sent`, `ping_ok`, `echo_sent`, `echo_ok`, `time_sent`, `time_ok`,
`rtt_min_ms`, `rtt_avg_ms`, `rtt_max_ms`, `detail`.

On `Ctrl+C`/`SIGTERM`, the checker prints results for fully checked ports in the
selected format and exits with code `10`; unfinished ports are omitted.

`--udp-checker` works without root. `--udp-listener` requires root only if
the range includes ports `<1024` or `/proc/net/udp` and `/proc/net/udp6` are
not readable by the current user. The listener runs until `Ctrl+C`/`SIGTERM`.

Example `--list`:

```
INTERFACE  TYPE  VERSION  ADDRESS                   PORT   STATUS  CLIENTS
awg0       awg2  2.0      10.8.0.1/24,fd00:8::1/64  51820  up      12 (1 locked)
wg0        wg    -        10.9.0.1/24               51821  down    3
```

Default MTU: `wg` — 1420, `awg`/`awg2` — 1376, `awg3` — 1280. Obfuscation
parameters (Jc, S1–S4, H1–H4, …) are generated automatically in safe ranges
and validated (section 7 of the spec).

### Clients

```sh
# add a client: addresses are auto-assigned (first free /32 and /128);
# the client config is printed ONCE — the client private key is never stored
sudo krot --add-client --interface awg0 --id phone-sergei --allowed-ips 0.0.0.0/0,::/0

# with a PresharedKey and a QR code in the terminal
sudo krot --add-client --interface awg0 --id laptop --allowed-ips 0.0.0.0/0 --psk --qr

# additionally print an AmneziaVPN key: a vpn://… line after the config
sudo krot --add-client --interface awg0 --id tablet --allowed-ips 0.0.0.0/0,::/0 --amnezia-key

# KeeneticOS CLI commands: connect a Keenetic router as a client (wg only, IPv4 only)
sudo krot --add-client --interface wg0 --id keenetic --allowed-ips 0.0.0.0/0 --keenetic

# specific client address
sudo krot --add-client --interface awg0 --id router --allowed-ips 10.8.0.0/24 --ip 10.8.0.50/32

# foreign config without krot metadata: pass the endpoint explicitly (not written to the server config)
sudo krot --add-client --interface office --id guest --allowed-ips 0.0.0.0/0 --endpoint 203.0.113.10

# custom DNS for a single client (not written to the server config)
sudo krot --add-client --interface awg0 --id kid-phone --allowed-ips 0.0.0.0/0 --dns 94.140.14.15

sudo krot --list-client --interface awg0
sudo krot --list-client --interface awg0 --id phone-sergei --json
sudo krot --lock-client   --interface awg0 --id laptop    # repeated — "already locked", code 0
sudo krot --unlock-client --interface awg0 --id laptop
sudo krot --delete-client --interface awg0 --id laptop
```

`--allowed-ips` sets `AllowedIPs` in the **client** config (what the client
routes through the tunnel). Client DNS: `--dns` if given; otherwise the
interface's `# krot:dns` metadata (`--dns` at `--create`); otherwise the `DNS`
lines from the server config's `[Interface]`; if there is no DNS anywhere, the
client config has no `DNS` line.

Client addresses are allocated from the interface subnet, excluding the
network address, broadcast address, and server address. Point-to-point subnets
`/31` and `/127` (RFC 3021, RFC 6164) are also supported: both addresses are
usable, so a "server + one client" tunnel can be created with
`--create … --ip 10.8.0.0/31,fd00:8::/127`.

Save the generated config immediately: it cannot be retrieved again; you can
only delete the client and add it again. QR codes are printed with Unicode
half-blocks and assume a dark terminal background.

`--amnezia-key` prints an AmneziaVPN key after the config — a single `vpn://…`
line to paste into the AmneziaVPN app ("Insert key"), for example, by sending
it through a messenger. It contains the same client config: `vpn://` +
base64url without `=` of Qt's `qCompress` (4-byte big-endian length + zlib).
The key works for all types, including `wg`. The QR code from `--qr` contains
the config text — AmneziaVPN and AmneziaWG can scan it; for `wg`, WireGuard
can too.

`--keenetic` prints a block of KeeneticOS CLI commands after the config —
connecting a Keenetic router as a tunnel client. The block follows the
running-config syntax of KeeneticOS and can be pasted into the router CLI
(SSH/telnet or the web interface "Command-line interface") as-is. KeeneticOS
supports AmneziaWG obfuscation (ASC parameters): AWG 1.0 since KeeneticOS 4.2,
AWG 1.5/2.0 since 5.1; for awg tunnels the block includes a `wireguard asc …`
line (CPS strings I1–I5 are quoted; importing the config file via the web UI
is more reliable). AWG 3.1 is not supported. IPv4 only; if the tunnel MTU
differs from the default 1420, a note is printed. See section 6.13 of the spec
for the format.

Example `--list-client`:

```
ID            PUBLIC KEY  ADDRESS                    STATUS  LAST HANDSHAKE  RX       TX
phone-sergei  aB3d…Xy9=   10.8.0.2/32,fd00:8::2/128  active  1m ago          12.4MiB  1.1GiB
laptop-old    Qw7e…Zz1=   10.8.0.3/32                locked  -               -        -
```

### Mimicry (I1–I5 packets)

Starting with version 1.5, AmneziaWG can send up to five signature packets
`I1`–`I5` before each handshake: packets that look like the beginning of
another protocol (DNS, STUN, NTP, SSDP, DTLS). krot generates such chains
itself via the `--mimic <profile>` flag:

```sh
# default profile for all clients of an interface
krot --create --type awg --ip 10.8.0.1/24 --port 51820 --endpoint vpn.example.com --mimic dns
# override for a single client
krot --add-client --interface awg0 --id phone --allowed-ips 0.0.0.0/0 --mimic stun
```

Profiles: `dns` (three A/AAAA/HTTPS queries), `stun` (Binding Request), `ntp`,
`ssdp` (M-SEARCH), `dtls` (ClientHello). The chain is generated at
`--add-client` (each client gets its own) and written only to the client
config: I-packets are sent by the handshake initiator, so the server doesn't
need them. The client requires AmneziaWG 1.5+ — see section 6.12 of the spec.

In addition to built-in profiles, krot accepts a ready-made CPS string via
`--i1` at `--create` — for example, for QUIC: built-in QUIC profiles are not
generated because a single QUIC Initial is 2400–2500 characters and exceeds
the 2048 limit (due to
[amneziawg-tools#69](https://github.com/amnezia-vpn/amneziawg-tools/issues/69)).
A ready-made string can be obtained from third-party generators (both
MIT-licensed, not affiliated with krot):

- [payloadGen](https://github.com/Sketchystan1/payloadGen) — a web page: open
  `index.html` in a browser, select a protocol, and take the first line (I1);
- `cpsgen.py` from [awg-multi-script](https://github.com/pumbaX/awg-multi-script)
  (`src/py/cpsgen.py`) — a console version of payloadGen in Python 3, no root
  required. Profiles: `dns`, `ntp`, `rtp`, `stun`, `sip`, `ssdp`, `webrtc`,
  `quic`, `curl_quic`; second argument is the domain.

```sh
# download the generator and review its code before running
git clone --depth 1 https://github.com/pumbaX/awg-multi-script
less awg-multi-script/src/py/cpsgen.py

# generate only I1 and check its length (must be ≤ 2048 characters)
I1=$(python3 awg-multi-script/src/py/cpsgen.py dns example.com --only-i1)
echo ${#I1}

sudo krot --create --type awg2 --ip 10.8.0.1/24 --port 51820 --endpoint vpn.example.com --i1 "$I1"
```

Notes:

- `I1` is set only at `--create`, written to the server config, and copied to
  all client configs. `I2`–`I5` are not accepted by krot.
- Length must not exceed 2048 characters (see [Limitations](#limitations)),
  otherwise code `3`. The `quic` and `curl_quic` profile packets are about
  2400 characters and won't fit. Shorter profiles work: `rtp` (~30 chars),
  `dns` and `ntp` (~100), `stun` (~250–350), `ssdp` (~400), `sip` (~1200).
- `--type awg` with `--i1` means AmneziaWG 1.5 with its own minimum versions
  (see [AmneziaWG versions](#amneziawg-versions)). The `<rc N>`/`<rd N>` tags
  (produced by the `sip` profile) are understood by amneziawg-go since 0.2.15,
  so for 1.5 prefer profiles without them.
- The client must support AmneziaWG 1.5 or newer.
- With a long `I1`, the config may not fit in a QR code: `--qr` then exits
  with code `3` (the client is already added and the config is printed).
  Transfer the config as a file or use `--amnezia-key`.

### JSON

`--list --json` outputs an array of objects with `interface`, `type`,
`version`, `addresses[]`, `port`, `status`, `clients_total`, `clients_locked`.
`--list-client --json` outputs an array of objects with `id`, `public_key`,
`addresses[]`, `status`, `latest_handshake` (unix time or `null`), `rx_bytes`,
`tx_bytes`, `endpoint` (or `null`), `created` (or `null` for peers not managed by krot).
An empty result is `[]`.

## Exit codes

| Code | Meaning |
|---|---|
| 0 | success |
| 1 | internal or unexpected error (including failure to apply a change to a running interface — config rolled back) |
| 2 | CLI usage error (flags; deleting with clients without `--yes` when stdin is not a terminal) |
| 3 | value validation error (including missing endpoint, "config too large for QR") |
| 4 | object not found (interface, client) |
| 5 | conflict: already exists, address or port in use, no free addresses |
| 6 | environment: no root, missing tools, unsupported AmneziaWG version |
| 7 | could not acquire the `/run/lock/krot.lock` lock within 10 seconds |
| 10 | operation cancelled by user |

Output goes to stdout; errors and warnings go to stderr with a `krot:` prefix.

## Data storage

| Type | Config | systemd unit | Tool |
|---|---|---|---|
| `wg` | `/etc/wireguard/<if>.conf` | `wg-quick@<if>` | `wg` |
| `awg`, `awg2`, `awg3` | `/etc/amnezia/amneziawg/<if>.conf` | `awg-quick@<if>` | `awg` |

Metadata is stored as comments that `wg-quick`/`awg-quick` ignore:

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

- A locked client is a `[Peer]` block commented out with the `#~ ` prefix.
- The commented-out `PreUp`/`PostUp`/`PreDown`/`PostDown` hooks can be
  uncommented and filled in (for example, with NAT rules) — krot doesn't
  touch them.
- Configs without krot metadata (created manually) are supported: `--list`
  shows them, `--list-client` prints their peers with id `-` (operations by
  `--id` don't apply to such peers), and any write preserves all original
  lines, comments, and ordering byte-for-byte.
- Writes are atomic (temp file → `fsync` → `rename` → directory `fsync`),
  permissions `0600`.
- All mutating actions take an exclusive `flock` on `/run/lock/krot.lock` —
  concurrent runs (for example, Salt and a human) don't lose changes.
- krot does not make backups.

## Limitations

- NAT, forwarding, firewall, sysctl are out of scope for krot (use the
  `PostUp`/`PostDown` hooks).
- Installing WireGuard/AmneziaWG and migrating tunnels between types are not
  supported. Mimicry signature generation uses built-in profiles (`--mimic`);
  QUIC profiles and custom signatures go through `--i1` and third-party
  generators, see [Mimicry](#mimicry-i1i5-packets).
- `--i1` must not exceed 2048 characters: due to
  [amneziawg-tools#69](https://github.com/amnezia-vpn/amneziawg-tools/issues/69),
  long `I1`–`I5` (roughly 3600+ characters) make `awg show` hang and
  `awg set`/`setconf` fail.
- Client private keys are not stored; client configs cannot be reissued.

## Development

You need Go from `go.mod` (currently 1.27.2) and Docker for integration tests.

```sh
make check        # all local checks: formatting, tests, modules, security and workflow linters
make fuzz-smoke   # bounded fuzzing of all targets
make build        # static bin/krot (CGO_ENABLED=0), version from git describe
make build-arm64
make integration  # integration tests in a privileged Docker container with systemd
make package-preflight # DEB/RPM for both architectures + Ubuntu 24.04/AlmaLinux 9 smoke
```

Repository rules: [`AGENTS.md`](AGENTS.md) — spec first, tests before code,
one commit per task.

- **Unit tests** don't require root and don't touch the system: directories
  are replaced with temporary ones, external commands (`wg`, `awg`,
  `systemctl`, `ip`, …) are faked with call recording. Separate tests verify
  that client operations call only `wg|awg show` and `wg|awg set <if> peer …`.
- **Golden files** live in package `testdata/`. Generated outputs
  (`internal/app`, `internal/qr`) are updated only deliberately — with
  `go test ./internal/app ./internal/qr -update` and an explanation in the
  commit; other golden files are written manually from the spec.
- **Integration tests** (`test/integration`, build tag `integration`) require
  root and systemd and run only in separate network namespaces: test interface
  units are placed into netns via the `NetworkNamespacePath` drop-in. They
  cover the full cycle for all types, client-operation transparency
  (continuous ping without loss, unchanged ifindex and handshake), IPv6,
  foreign configs, and concurrent runs. Locally via
  `scripts/integration-docker.sh`, in CI under `sudo`.
- `scripts/collect-versions.sh` rebuilds real AmneziaWG version outputs for
  golden tests.
- **CI** (`.github/workflows/ci.yml`): local quality gates, shuffled unit tests
  with `-race` and coverage (minimum 90%), bounded fuzzing, integration tests,
  package preflight, and `amd64`/`arm64` builds. **Release** (`release.yml`, tag
  `v*`): binaries, `.deb`/`.rpm` packages (nfpm, `packaging/nfpm.yaml`),
  package smoke tests on all OSes (`scripts/smoke.sh`), GitHub Release with
  `SHA256SUMS`.

## License

[Apache License 2.0](LICENSE). Copyright 2026 Sergey Ovchinnikov.
