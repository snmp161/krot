package cli

const usageTextGerman = `krot — verwaltet WireGuard- und AmneziaWG-Tunnel

Aufruf:
  krot <Aktion> [Flags]

Aktionen (genau eine):
  --list            listet Tunnel auf
  --create          erstellt einen Tunnel
  --delete          löscht einen Tunnel
  --up              startet eine Schnittstelle
  --down            stoppt eine Schnittstelle
  --add-client      fügt einen Client hinzu
  --delete-client   löscht einen Client
  --lock-client     sperrt einen Client
  --unlock-client   entsperrt einen Client
  --list-client     listet Clients auf
  --doctor          prüft die Umgebung (Tools, Kernelmodule, AmneziaWG-Versionen)
  --udp-listener    UDP-Listener zur Portdiagnose
  --udp-checker     UDP-Checker zur Portdiagnose
  --help            zeigt diese Hilfe
  --version         zeigt Version, Commit und Build-Datum

Flags (--flag value oder --flag=value):
  --type wg|awg|awg2|awg3   Protokolltyp (create: erforderlich; list: Filter)
  --interface NAME          Schnittstellenname, bis zu 15 Zeichen [a-zA-Z0-9_.=+-]
  --ip CIDR[,CIDR]          create: Server-Adressen (10.8.0.1/24,fd00:8::1/64);
                            add-client: Client-Adressen (/32, /128);
                            list: Filter nach Adresse
  --port N|N:M              UDP-Port oder -Bereich 1..65535, höchstens 1024 Ports
                            (create, udp-listener, udp-checker: erforderlich; list: Filter)
  --endpoint HOST           öffentliche Server-Adresse für Client-Konfigurationen
                            (create: erforderlich; add-client: clientbezogene Überschreibung)
  --address HOST            Adresse des Remote-Listeners für --udp-checker (ohne Port)
  --mtu N                   MTU 576..9000 (Standard: wg 1420, awg/awg2 1376, awg3 1280)
  --dns IP[,IP]             DNS-Server für Client-Konfigurationen
                            (create: für alle Clients; add-client: nur dieser Client)
  --i1 CPS                  AmneziaWG-I1-Signaturpaket (nur awg*, höchstens 2048 Zeichen)
  --mimic PROFILE           erzeugt eine I1-I5-Mimikry-Kette für Client-Konfigurationen:
                            dns|stun|ntp|ssdp|dtls (nur awg*; nicht mit --i1)
  --id ID                   Client-ID [a-zA-Z0-9_.-]{1,64}, eindeutig innerhalb einer Schnittstelle
  --allowed-ips CIDR[,CIDR] AllowedIPs der Client-Konfiguration, z. B. 0.0.0.0/0,::/0
  --psk                     erzeugt einen PresharedKey für den Client
  --qr                      gibt die Client-Konfiguration zusätzlich als QR-Code aus
  --amnezia-key             gibt die Client-Konfiguration als AmneziaVPN-Schlüssel aus (vpn://...)
  --keenetic                gibt zusätzlich KeeneticOS-CLI-Befehle für einen Keenetic-Router aus
                            (wg- und awg-1.0/1.5/2.0-Tunnel, nur IPv4)
  --json                    JSON-Ausgabe statt Tabelle (list, list-client, udp-listener, udp-checker)
  --yes                     fragt nicht nach Bestätigung (delete)
  --timeout DURATION        Paket-Timeout, z. B. 2s oder 500ms (Standard: 2s)
  --count N                 Pakete jedes Typs pro Port, 1..100 (Standard: 3)
  --echo-size N             Größe der Echo-Nutzdaten, 1..1400 (Standard: 64, nur udp-checker)
  --workers N               gleichzeitige Portprüfungen, 1..100 (Standard: 32, nur udp-checker)
  --verbose                 Diagnosefortschritt (udp-listener, udp-checker; stderr)

Flags pro Aktion:
  --list           [--type] [--interface] [--ip] [--port] [--json]
  --create         --type --ip --port --endpoint [--interface] [--mtu] [--dns] [--i1] [--mimic]
  --delete         --interface [--yes]
  --up, --down     --interface
  --add-client     --interface --id --allowed-ips [--ip] [--endpoint] [--psk] [--qr]
                   [--dns] [--amnezia-key] [--mimic] [--keenetic]
  --delete-client  --interface --id
  --lock-client    --interface --id
  --unlock-client  --interface --id
  --list-client    --interface [--id] [--json]
  --doctor         (keine Flags; funktioniert ohne root)
  --udp-listener   --port [--timeout] [--count] [--json] [--verbose]
  --udp-checker    --address --port [--timeout] [--count] [--echo-size] [--workers] [--json] [--verbose]

Exit-Codes:
  0 Erfolg                  5 Konflikt (existiert bereits, Adresse oder Port belegt)
  1 interner Fehler         6 Umgebung (kein root, fehlende Tools, AmneziaWG-Version)
  2 Aufruffehler            7 Sperre konnte nicht erlangt werden
  3 ungültiger Wert         10 vom Benutzer abgebrochen
  4 nicht gefunden

Root-Rechte:
  Nicht erforderlich für --help, --version, --doctor oder --udp-checker.
  --udp-listener erfordert root nur für Ports unter 1024 oder wenn
  /proc/net/udp und /proc/net/udp6 nicht lesbar sind.
  Alle anderen Aktionen erfordern root.
`
