# krot

[English](README.md) | [Русский](README.ru.md) | [简体中文](README.zh.md) | [繁體中文](README.zh-Hant.md) | [Español](README.es.md) | [Português](README.pt.md) | [한국어](README.ko.md) | [日本語](README.ja.md) | [العربية](README.ar.md) | [Français](README.fr.md) | **Deutsch**

`krot` ist ein Befehlszeilenprogramm zur Verwaltung von **WireGuard**- und
**AmneziaWG**-Tunneln (1.x, 2.0, 3.1) auf Linux-Servern: Es erstellt und löscht
Interfaces, verwaltet Clients (Peers) und erzeugt Client-Konfigurationen, QR-Codes
und AmneziaVPN-Schlüssel (`vpn://…`).

Der Name **krot** bedeutet im Russischen und in mehreren anderen slawischen
Sprachen „Maulwurf“ — ein Tier, das Tunnel gräbt.

- Eine einzelne statische Binärdatei ohne Abhängigkeiten (Go, `CGO_ENABLED=0`).
- Client-Operationen **starten oder erstellen das Interface niemals neu**:
  Es erfolgen ausschließlich gezielte Aufrufe von `wg|awg set <if> peer …`;
  andere Clients bleiben unbeeinflusst.
- Die Konfiguration des Interfaces ist die alleinige maßgebliche Datenquelle:
  keine separaten Datenbanken; krot-Metadaten werden in Kommentaren gespeichert,
  und nicht von krot erstellte Konfigurationen werden gelesen und beibehalten.
- Vorhersehbare Ausgabe: Tabellen für Menschen, `--json` und Exit-Codes für die
  Automatisierung.
- krot konfiguriert weder NAT noch Forwarding oder Firewall-Regeln.

Vollständige technische Spezifikation: [`docs/SPEC.md`](docs/SPEC.md).

## Voraussetzungen

| | |
|---|---|
| Betriebssystem | Ubuntu 22.04 / 24.04 / 26.04, Debian 12 / 13, AlmaLinux (Rocky, RHEL) 9 / 10 |
| Architekturen | `amd64`, `arm64` |
| Für `--type wg` | `wireguard-tools` (`wg`, `wg-quick`), WireGuard-Kernelmodul |
| Für `--type awg*` | `amneziawg-tools` (`awg`, `awg-quick`) und AmneziaWG-Kernelmodul **oder** `amneziawg-go` |
| Immer | `iproute2` (`ip`), `systemd` |
| Berechtigungen | root (außer `--help`, `--version`, `--doctor`, `--udp-checker`; `--udp-listener` benötigt root nur für Ports `<1024` oder wenn `/proc/net/udp{,6}` nicht lesbar ist) |

krot installiert diese Komponenten nicht selbst. Fehlt ein benötigtes Programm,
endet krot mit Code `6` und nennt das zu installierende Paket.

### AmneziaWG-Versionen

Vor `--create` prüft krot, ob das installierte AmneziaWG den angeforderten Typ
unterstützt:

| Typ | amneziawg-tools | Kernelmodul | Netlink-Familie | amneziawg-go |
|---|---|---|---|---|
| `awg` (1.0) | 1.0.20240213 | — | ≥ 1 | 0.1.0 |
| `awg` mit `--i1` (1.5) | 1.0.20250901 | 1.0.20251009 | ≥ 2 | 0.2.13 |
| `awg2` (2.0) | 1.0.20250901 | 1.0.20251009 | ≥ 2 | 0.2.15 |
| `awg3` (3.1) | 3.1.20260812 | 3.1.20260812 | ≥ 3 | 3.1.20260812 |

Einige Builds melden eine Platzhalterversion (beispielsweise gibt `awg --version`
bei den Tools 1.x fast immer `1.0.20210914` aus, und ein mit `make` erstelltes
Modul meldet `1.0.0`). Solche Versionen erzeugen eine Warnung, keinen Fehler;
Code `6` wird nur zurückgegeben, wenn die Version zuverlässig erkannt wird und
zu alt ist. Ist die Version tatsächlich zu alt, schlägt der Start der Unit fehl
und krot macht die Änderungen rückgängig.

## Installation

Installieren Sie krot aus Paketen oder Binärdateien auf der Release-Seite
<https://github.com/snmp161/krot/releases>, und prüfen Sie stets die Prüfsumme.

```sh
VERSION=0.7.3
BASE=https://github.com/snmp161/krot/releases/download/v$VERSION

# Paket und Liste der Prüfsummen herunterladen
wget "$BASE/krot_${VERSION}_amd64.deb" "$BASE/SHA256SUMS"
# prüfen: sollte "krot_..._amd64.deb: OK" ausgeben
sha256sum --check --ignore-missing SHA256SUMS
# installieren
sudo dpkg -i "krot_${VERSION}_amd64.deb"
```

Systeme mit RPM (AlmaLinux, Rocky, RHEL):

```sh
wget "$BASE/krot-${VERSION}-1.x86_64.rpm" "$BASE/SHA256SUMS"
sha256sum --check --ignore-missing SHA256SUMS
sudo rpm -i "krot-${VERSION}-1.x86_64.rpm"
```

Binärdatei ohne Paket: Laden Sie `krot-linux-amd64` (oder `krot-linux-arm64`)
und `SHA256SUMS` herunter, führen Sie dieselbe Prüfung aus und installieren Sie
anschließend mit `sudo install -m 0755 krot-linux-amd64 /usr/bin/krot`.

Verwenden Sie für `arm64` die Dateien `*_arm64.deb`, `*.aarch64.rpm` und `krot-linux-arm64`.

## Verwendung

```
krot <action> [flags]
```

Pro Aufruf ist genau eine Aktion zulässig; Flags unterstützen ausschließlich die
Langform `--flag value` oder `--flag=value`. Ein Flag, das nicht zur Aktion passt,
ist ein Fehler (Code `2`). Vollständige Hilfe: `krot --help`.

### Tunnel

```sh
# AmneziaWG 2.0 erstellen (Dual-Stack); Name wird automatisch gewählt: awg0, awg1, …
sudo krot --create --type awg2 --ip 10.8.0.1/24,fd00:8::1/64 --port 51820 \
          --endpoint vpn.example.com --dns 1.1.1.1,9.9.9.9

# WireGuard mit explizitem Namen und MTU
sudo krot --create --type wg --interface wg0 --ip 10.9.0.1/24 --port 51821 \
          --endpoint 203.0.113.10 --mtu 1420

# AmneziaWG 1.5 mit einem I1-Signaturpaket (krot prüft CPS-Syntax und -Länge — bis zu 2048 Zeichen)
sudo krot --create --type awg --ip 10.10.0.1/24 --port 51822 --endpoint vpn.example.com \
          --i1 '<b 0xf6ab3267fa><r 16>'

# AmneziaWG 3.1 (Header Protection, RandomTrailers, DisableCookies sind immer aktiviert)
sudo krot --create --type awg3 --ip 10.11.0.1/24 --port 51823 --endpoint vpn.example.com

sudo krot --list                         # alle Tunnel
sudo krot --list --type awg2 --json      # Filter --type/--interface/--ip/--port, mit UND verknüpft
sudo krot --down --interface awg0        # systemctl stop awg-quick@awg0
sudo krot --up --interface awg0          # systemctl start; wiederholtes --up gibt "already up" aus, Code 0
sudo krot --delete --interface awg0      # fragt nach Bestätigung, wenn Clients vorhanden sind
sudo krot --delete --interface awg0 --yes
```

### Diagnose

```sh
krot --doctor   # funktioniert ohne root: Tools, Kernelmodule, AmneziaWG-Versionen, Verzeichnisse
```

Die Ausgabe ist eine Tabelle `CHECK/STATUS/DETAIL` mit den Status `ok` / `warn` /
`fail`. Der Exit-Code ist `0`, wenn kein Eintrag den Status `fail` hat, andernfalls `6`.
Die Aktion verändert nichts und nimmt keine Sperre.

#### Diagnose von UDP-Ports

`krot` kann prüfen, welche UDP-Ports zwischen zwei Hosts tatsächlich passierbar
sind. Das ist nützlich, wenn ein Hosting-Anbieter oder dessen Anbindung bestimmte
Ports filtert, beispielsweise aufgrund von DDoS-Regeln.

Starten Sie den Listener auf dem Server:

```sh
# einzelner Port
sudo krot --udp-listener --port 51820

# Portbereich (bis zu 1024 Ports, ein Socket pro Port)
sudo krot --udp-listener --port 51820:51830
```

Der Listener beantwortet drei Datagrammtypen:
- `ping SEQ:N` → `pong SEQ:N`;
- `ECHO <text>` → `<text>`;
- `TIME` → aktuelle Unix-Zeit in Millisekunden.

Führen Sie den Checker auf dem Client aus:

```sh
# einen einzelnen Port prüfen
krot --udp-checker --address 203.0.113.1 --port 51820

# einen Bereich prüfen
krot --udp-checker --address 203.0.113.1 --port 51820:51830

# Echo-Nutzdaten vergrößern (1–1400 Bytes, Standard 64) — hilft bei der Ermittlung des optimalen MTU
krot --udp-checker --address 203.0.113.1 --port 51820 --echo-size 1400

# Anzahl der Worker festlegen und Fortschritt anzeigen
krot --udp-checker --address 203.0.113.1 --port 51820:51830 --workers 64 --verbose
```

Standardmäßig sendet der Checker 3 Pakete jedes Typs pro Port (`--count`, 1–100)
und wartet innerhalb von `--timeout` auf eine Antwort (Standard 2s). Er prüft
32 Ports parallel; mit `--workers` lassen sich 1–100 Worker festlegen.
`--verbose` schreibt den Bereich und die Ergebnisse einzelner Ports nach deren
Abschluss auf stderr; beim Listener protokolliert es Client-Adressen und
Anfragetypen. Ausgabe:

```
PORT     STATUS   PING    ECHO    TIME (min/avg/max)   DETAIL
51820    ok       3/3     3/3     12/24/45 ms          -
51821    filtered 0/3     0/3     -                    timeout
```

Status:
- `ok` — alle Ping- und Echo-Pakete haben gültige Antworten erhalten;
- `filtered` — für mindestens ein Paket wurde das Zeitlimit überschritten (der Port ist wahrscheinlich gefiltert);
- `error` — eine ungültige Antwort wurde empfangen.

Mit `--json` ist die Ausgabe ein Array von Objekten mit den Feldern `port`,
`status`, `ping_sent`, `ping_ok`, `echo_sent`, `echo_ok`, `time_sent`, `time_ok`,
`rtt_min_ms`, `rtt_avg_ms`, `rtt_max_ms`, `detail`.

Bei `Ctrl+C`/`SIGTERM` gibt der Checker die Ergebnisse vollständig geprüfter Ports
im gewählten Format aus und endet mit Code `10`; unvollendete Ports werden ausgelassen.

`--udp-checker` funktioniert ohne root. `--udp-listener` benötigt root nur, wenn
der Bereich Ports `<1024` enthält oder `/proc/net/udp` und `/proc/net/udp6` für
den aktuellen Benutzer nicht lesbar sind. Der Listener läuft bis zu `Ctrl+C`/`SIGTERM`.

Beispiel für `--list`:

```
INTERFACE  TYPE  VERSION  ADDRESS                   PORT   STATUS  CLIENTS
awg0       awg2  2.0      10.8.0.1/24,fd00:8::1/64  51820  up      12 (1 locked)
wg0        wg    -        10.9.0.1/24               51821  down    3
```

Standard-MTU: `wg` — 1420, `awg`/`awg2` — 1376, `awg3` — 1280. Die
Obfuskationsparameter (Jc, S1–S4, H1–H4, …) werden automatisch in sicheren
Bereichen erzeugt und validiert (Abschnitt 7 der Spezifikation).

### Clients

```sh
# Client hinzufügen: Adressen werden automatisch zugewiesen (erste freie /32 und /128);
# die Client-Konfiguration wird NUR EINMAL ausgegeben — der private Schlüssel des Clients wird niemals gespeichert
sudo krot --add-client --interface awg0 --id phone-sergei --allowed-ips 0.0.0.0/0,::/0

# mit PresharedKey und QR-Code im Terminal
sudo krot --add-client --interface awg0 --id laptop --allowed-ips 0.0.0.0/0 --psk --qr

# zusätzlich einen AmneziaVPN-Schlüssel ausgeben: eine Zeile vpn://… nach der Konfiguration
sudo krot --add-client --interface awg0 --id tablet --allowed-ips 0.0.0.0/0,::/0 --amnezia-key

# KeeneticOS-CLI-Befehle: Keenetic-Router als Client verbinden (nur wg, nur IPv4)
sudo krot --add-client --interface wg0 --id keenetic --allowed-ips 0.0.0.0/0 --keenetic

# bestimmte Client-Adresse
sudo krot --add-client --interface awg0 --id router --allowed-ips 10.8.0.0/24 --ip 10.8.0.50/32

# fremde Konfiguration ohne krot-Metadaten: Endpoint explizit angeben (wird nicht in die Server-Konfiguration geschrieben)
sudo krot --add-client --interface office --id guest --allowed-ips 0.0.0.0/0 --endpoint 203.0.113.10

# eigener DNS für einen einzelnen Client (wird nicht in die Server-Konfiguration geschrieben)
sudo krot --add-client --interface awg0 --id kid-phone --allowed-ips 0.0.0.0/0 --dns 94.140.14.15

sudo krot --list-client --interface awg0
sudo krot --list-client --interface awg0 --id phone-sergei --json
sudo krot --lock-client   --interface awg0 --id laptop    # wiederholt — "already locked", Code 0
sudo krot --unlock-client --interface awg0 --id laptop
sudo krot --delete-client --interface awg0 --id laptop
```

`--allowed-ips` legt `AllowedIPs` in der **Client**-Konfiguration fest (welchen
Datenverkehr der Client durch den Tunnel leitet). DNS des Clients: `--dns`, falls
angegeben; andernfalls die Metadaten `# krot:dns` des Interfaces (`--dns` bei
`--create`); andernfalls die `DNS`-Zeilen aus `[Interface]` der Server-Konfiguration.
Ist nirgends ein DNS festgelegt, enthält die Client-Konfiguration keine `DNS`-Zeile.

Client-Adressen werden aus dem Subnetz des Interfaces zugewiesen, ausgenommen
Netzwerkadresse, Broadcast-Adresse und Server-Adresse. Point-to-Point-Subnetze
`/31` und `/127` (RFC 3021, RFC 6164) werden ebenfalls unterstützt: beide Adressen
sind nutzbar, sodass ein Tunnel „Server + ein Client“ mit
`--create … --ip 10.8.0.0/31,fd00:8::/127` erstellt werden kann.

Speichern Sie die erzeugte Konfiguration sofort: Sie kann nicht erneut abgerufen
werden; Sie können den Client nur löschen und erneut hinzufügen. QR-Codes werden
mit Unicode-Halbblockzeichen ausgegeben und setzen einen dunklen
Terminal-Hintergrund voraus.

`--amnezia-key` gibt nach der Konfiguration einen AmneziaVPN-Schlüssel aus — eine
einzelne Zeile `vpn://…`, die in die AmneziaVPN-App („Insert key“) eingefügt wird,
beispielsweise nach dem Versand über einen Messenger. Sie enthält dieselbe
Client-Konfiguration: `vpn://` + base64url ohne `=` des Ergebnisses von Qt
`qCompress` (4-Byte-Länge in Big-Endian + zlib). Der Schlüssel funktioniert für
alle Typen, einschließlich `wg`. Der QR-Code aus `--qr` enthält den Text der
Konfiguration — AmneziaVPN und AmneziaWG können ihn einlesen; bei `wg` kann auch
WireGuard ihn ebenfalls einlesen.

`--keenetic` gibt nach der Konfiguration einen Block von KeeneticOS-CLI-Befehlen
aus, um einen Keenetic-Router als Tunnel-Client einzubinden. Der Block folgt der
Running-Config-Syntax von KeeneticOS und kann unverändert in die CLI des Routers
eingefügt werden (SSH/Telnet oder die Weboberfläche „Command-line interface“).
KeeneticOS unterstützt AmneziaWG-Obfuskation (ASC-Parameter): AWG 1.0 seit
KeeneticOS 4.2, AWG 1.5/2.0 seit 5.1; bei awg-Tunneln enthält der Block eine Zeile
`wireguard asc …` (CPS-Zeichenketten I1–I5 stehen in Anführungszeichen; der Import
der Konfigurationsdatei über die Weboberfläche ist zuverlässiger). AWG 3.1 wird
nicht unterstützt. Nur IPv4; weicht der MTU des Tunnels vom Standardwert 1420 ab,
wird ein Hinweis ausgegeben. Das Format beschreibt Abschnitt 6.13 der Spezifikation.

Beispiel für `--list-client`:

```
ID            PUBLIC KEY  ADDRESS                    STATUS  LAST HANDSHAKE  RX       TX
phone-sergei  aB3d…Xy9=   10.8.0.2/32,fd00:8::2/128  active  1m ago          12.4MiB  1.1GiB
laptop-old    Qw7e…Zz1=   10.8.0.3/32                locked  -               -        -
```

### Mimikry (I1–I5-Pakete)

Seit Version 1.5 kann AmneziaWG vor jedem Handshake bis zu fünf Signaturpakete
`I1`–`I5` senden: Pakete, die wie der Beginn eines anderen Protokolls aussehen
(DNS, STUN, NTP, SSDP, DTLS). krot erzeugt solche Ketten selbst mit dem Flag
`--mimic <profile>`:

```sh
# Standardprofil für alle Clients eines Interfaces
krot --create --type awg --ip 10.8.0.1/24 --port 51820 --endpoint vpn.example.com --mimic dns
# Überschreibung für einen einzelnen Client
krot --add-client --interface awg0 --id phone --allowed-ips 0.0.0.0/0 --mimic stun
```

Profile: `dns` (drei A/AAAA/HTTPS-Anfragen), `stun` (Binding Request), `ntp`,
`ssdp` (M-SEARCH), `dtls` (ClientHello). Die Kette wird bei `--add-client` erzeugt
(jeder Client erhält eine eigene) und nur in die Client-Konfiguration geschrieben:
I-Pakete werden vom Initiator des Handshakes gesendet, daher benötigt der Server
sie nicht. Der Client benötigt AmneziaWG 1.5+ — siehe Abschnitt 6.12 der Spezifikation.

Zusätzlich zu den integrierten Profilen akzeptiert krot eine fertige
CPS-Zeichenkette über `--i1` bei `--create`, beispielsweise für QUIC. Integrierte
QUIC-Profile werden nicht erzeugt, da ein einzelnes QUIC Initial 2400–2500 Zeichen
umfasst und das Limit von 2048 überschreitet (aufgrund von
[amneziawg-tools#69](https://github.com/amnezia-vpn/amneziawg-tools/issues/69)).
Fertige Zeichenketten können mit Generatoren von Drittanbietern erstellt werden
(beide MIT-lizenziert und nicht mit krot verbunden):

- [payloadGen](https://github.com/Sketchystan1/payloadGen) — eine Webseite:
  `index.html` im Browser öffnen, ein Protokoll auswählen und die erste Zeile (I1) übernehmen;
- `cpsgen.py` aus [awg-multi-script](https://github.com/pumbaX/awg-multi-script)
  (`src/py/cpsgen.py`) — Konsolenversion von payloadGen in Python 3, ohne root.
  Profile: `dns`, `ntp`, `rtp`, `stun`, `sip`, `ssdp`, `webrtc`, `quic`,
  `curl_quic`; das zweite Argument ist die Domain.

```sh
# Generator herunterladen und seinen Code vor der Ausführung prüfen
git clone --depth 1 https://github.com/pumbaX/awg-multi-script
less awg-multi-script/src/py/cpsgen.py

# nur I1 erzeugen und die Länge prüfen (höchstens 2048 Zeichen)
I1=$(python3 awg-multi-script/src/py/cpsgen.py dns example.com --only-i1)
echo ${#I1}

sudo krot --create --type awg2 --ip 10.8.0.1/24 --port 51820 --endpoint vpn.example.com --i1 "$I1"
```

Hinweise:

- `I1` wird nur bei `--create` festgelegt, in die Server-Konfiguration geschrieben
  und in alle Client-Konfigurationen kopiert. `I2`–`I5` werden von krot nicht akzeptiert.
- Die Länge darf 2048 Zeichen nicht überschreiten (siehe [Einschränkungen](#einschränkungen)),
  andernfalls wird Code `3` zurückgegeben. Die Pakete der Profile `quic` und
  `curl_quic` umfassen etwa 2400 Zeichen und passen nicht. Kürzere Profile
  funktionieren: `rtp` (~30 Zeichen), `dns` und `ntp` (~100), `stun` (~250–350),
  `ssdp` (~400), `sip` (~1200).
- `--type awg` mit `--i1` bedeutet AmneziaWG 1.5 mit eigenen Mindestversionen
  (siehe [AmneziaWG-Versionen](#amneziawg-versionen)). Die Tags `<rc N>`/`<rd N>`
  (vom Profil `sip` erzeugt) werden von amneziawg-go seit 0.2.15 unterstützt;
  für 1.5 sollten daher Profile ohne diese Tags bevorzugt werden.
- Der Client muss AmneziaWG 1.5 oder neuer unterstützen.
- Bei einem langen `I1` passt die Konfiguration möglicherweise nicht in einen
  QR-Code: `--qr` endet dann mit Code `3` (der Client wurde bereits hinzugefügt
  und die Konfiguration ausgegeben). Übertragen Sie die Konfiguration als Datei
  oder verwenden Sie `--amnezia-key`.

### JSON

`--list --json` gibt ein Array von Objekten mit `interface`, `type`, `version`,
`addresses[]`, `port`, `status`, `clients_total`, `clients_locked` aus.
`--list-client --json` gibt ein Array von Objekten mit `id`, `public_key`,
`addresses[]`, `status`, `latest_handshake` (Unix-Zeit oder `null`), `rx_bytes`,
`tx_bytes`, `endpoint` (oder `null`), `created` (oder `null` für nicht von krot verwaltete Peers) aus.
Ein leeres Ergebnis ist `[]`.

## Exit-Codes

| Code | Bedeutung |
|---|---|
| 0 | Erfolg |
| 1 | interner oder unerwarteter Fehler (einschließlich fehlgeschlagener Änderung an einem aktiven Interface — Konfiguration zurückgerollt) |
| 2 | Fehler bei der CLI-Verwendung (Flags; Löschen mit Clients ohne `--yes`, wenn stdin kein Terminal ist) |
| 3 | Fehler bei der Wertvalidierung (einschließlich eines fehlenden Endpoints, „config too large for QR“) |
| 4 | Objekt nicht gefunden (Interface, Client) |
| 5 | Konflikt: existiert bereits, Adresse oder Port belegt, keine freien Adressen |
| 6 | Umgebung: keine Root-Rechte, fehlende Tools, nicht unterstützte AmneziaWG-Version |
| 7 | die Sperre `/run/lock/krot.lock` konnte nicht innerhalb von 10 Sekunden erworben werden |
| 10 | Vorgang vom Benutzer abgebrochen |

Die Ausgabe erfolgt auf stdout; Fehler und Warnungen werden mit dem Präfix `krot:`
auf stderr ausgegeben.

## Datenspeicherung

| Typ | Konfiguration | systemd-Unit | Tool |
|---|---|---|---|
| `wg` | `/etc/wireguard/<if>.conf` | `wg-quick@<if>` | `wg` |
| `awg`, `awg2`, `awg3` | `/etc/amnezia/amneziawg/<if>.conf` | `awg-quick@<if>` | `awg` |

Metadaten werden als Kommentare gespeichert, die `wg-quick`/`awg-quick` ignorieren:

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

- Ein gesperrter Client ist ein `[Peer]`-Block, der mit dem Präfix `#~ ` auskommentiert ist.
- Die auskommentierten Hooks `PreUp`/`PostUp`/`PreDown`/`PostDown` können
  aktiviert und ausgefüllt werden (beispielsweise mit NAT-Regeln) — krot verändert
  sie nicht.
- Konfigurationen ohne krot-Metadaten (manuell erstellt) werden unterstützt:
  `--list` zeigt sie an, `--list-client` gibt ihre Peers mit der ID `-` aus
  (Operationen über `--id` gelten nicht für solche Peers), und jeder Schreibvorgang
  bewahrt alle ursprünglichen Zeilen, Kommentare und ihre Reihenfolge Byte für Byte.
- Schreibvorgänge sind atomar (temporäre Datei → `fsync` → `rename` → `fsync` des
  Verzeichnisses), mit den Berechtigungen `0600`.
- Alle verändernden Aktionen nehmen eine exklusive `flock`-Sperre auf
  `/run/lock/krot.lock` — parallele Ausführungen (beispielsweise durch Salt und
  einen Menschen) verlieren keine Änderungen.
- krot erstellt keine Backups.

## Einschränkungen

- NAT, Forwarding, Firewall und sysctl liegen außerhalb des Umfangs von krot
  (verwenden Sie die Hooks `PostUp`/`PostDown`).
- Die Installation von WireGuard/AmneziaWG und die Migration von Tunneln zwischen
  Typen werden nicht unterstützt. Die Erzeugung von Mimikry-Signaturen verwendet
  integrierte Profile (`--mimic`); QUIC-Profile und eigene Signaturen erfolgen über
  `--i1` und Generatoren von Drittanbietern, siehe [Mimikry](#mimikry-i1i5-pakete).
- `--i1` darf 2048 Zeichen nicht überschreiten: Aufgrund von
  [amneziawg-tools#69](https://github.com/amnezia-vpn/amneziawg-tools/issues/69)
  führen lange `I1`–`I5` (etwa 3600+ Zeichen) dazu, dass `awg show` hängen bleibt
  und `awg set`/`setconf` fehlschlagen.
- Private Schlüssel von Clients werden nicht gespeichert; Client-Konfigurationen
  können nicht erneut ausgestellt werden.

## Entwicklung

Sie benötigen Go aus `go.mod` (derzeit 1.27.4) und Docker für Integrationstests.

```sh
make check        # alle lokalen Checks: Format, Tests, Module, Sicherheit und Workflow-Linter
make fuzz-smoke   # zeitlich begrenztes Fuzzing aller Targets
make build        # statisches bin/krot (CGO_ENABLED=0), Version aus git describe
make build-arm64
make integration  # Integrationstests in einem privilegierten Docker-Container mit systemd
make package-preflight # DEB/RPM für beide Architekturen + Ubuntu-24.04-/AlmaLinux-9-Smoke
```

Regeln des Repositorys: [`AGENTS.md`](AGENTS.md) — Spezifikation zuerst, Tests vor
dem Code, ein Commit pro Aufgabe.

- **Unit-Tests** benötigen keine Root-Rechte und verändern das System nicht: Verzeichnisse
  werden durch temporäre ersetzt, externe Befehle (`wg`, `awg`, `systemctl`, `ip`, …)
  werden mit Aufrufaufzeichnung simuliert. Separate Tests prüfen, dass
  Client-Operationen nur `wg|awg show` und `wg|awg set <if> peer …` aufrufen.
- **Golden Files** liegen im `testdata/`-Verzeichnis der Pakete. Erzeugte Ausgaben
  (`internal/app`, `internal/qr`) werden nur absichtlich aktualisiert — mit
  `go test ./internal/app ./internal/qr -update` und einer Erklärung im Commit;
  andere Golden Files werden manuell aus der Spezifikation geschrieben.
- **Integrationstests** (`test/integration`, Build-Tag `integration`) benötigen
  root und systemd und laufen nur in separaten Netzwerk-Namespaces: Die Units der
  Test-Interfaces werden über ein `NetworkNamespacePath`-Drop-in in den Netns
  verschoben. Sie decken den vollständigen Zyklus für alle Typen, die Transparenz
  von Client-Operationen (kontinuierlicher Ping ohne Verluste, unveränderter
  ifindex und Handshake), IPv6, fremde Konfigurationen und parallele Ausführungen
  ab. Lokal über `scripts/integration-docker.sh`, in CI unter `sudo`.
- `scripts/collect-versions.sh` erstellt die realen Ausgaben der
  AmneziaWG-Versionen für Golden-Tests neu.
- **CI** (`.github/workflows/ci.yml`): lokale Qualitätschecks, Unit-Tests mit
  Shuffle, `-race` und Coverage (mindestens 90 %), begrenztes Fuzzing,
  Integrationstests, Package-Preflight und Builds für `amd64`/`arm64`. **Release** (`release.yml`, Tag
  `v*`): Binärdateien, `.deb`/`.rpm`-Pakete (nfpm, `packaging/nfpm.yaml`),
  Paket-Smoke-Tests auf allen Betriebssystemen (`scripts/smoke.sh`), GitHub Release
  mit `SHA256SUMS`.

## Lizenz

[Apache License 2.0](LICENSE). Copyright 2026 Sergey Ovchinnikov.
