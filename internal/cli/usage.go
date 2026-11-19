package cli

import "github.com/snmp161/krot/internal/i18n"

// Usage возвращает текст справки (--help).
func Usage() string {
	if i18n.Current() == i18n.Russian {
		return usageTextRussian
	}
	return usageText
}

const usageText = `krot — manage WireGuard and AmneziaWG tunnels

Usage:
  krot <action> [flags]

Actions (exactly one):
  --list            list tunnels
  --create          create a tunnel
  --delete          delete a tunnel
  --up              start an interface
  --down            stop an interface
  --add-client      add a client
  --delete-client   delete a client
  --lock-client     lock a client
  --unlock-client   unlock a client
  --list-client     list clients
  --doctor          check the environment (tools, kernel modules, AmneziaWG versions)
  --udp-listener    UDP listener for port diagnostics
  --udp-checker     UDP checker for port diagnostics
  --help            show this help
  --version         show version, commit and build date

Flags (--flag value or --flag=value):
  --type wg|awg|awg2|awg3   protocol type (create: required; list: filter)
  --interface NAME          interface name, up to 15 chars [a-zA-Z0-9_.=+-]
  --ip CIDR[,CIDR]          create: server addresses (10.8.0.1/24,fd00:8::1/64);
                            add-client: client addresses (/32, /128);
                            list: filter by address
  --port N|N:M              UDP port or range 1..65535, at most 1024 ports
                            (create, udp-listener, udp-checker: required; list: filter)
  --endpoint HOST           public server address for client configs
                            (create: required; add-client: per-client override)
  --address HOST            remote listener address for --udp-checker (without port)
  --mtu N                   MTU 576..9000 (default: wg 1420, awg/awg2 1376, awg3 1280)
  --dns IP[,IP]             DNS servers for client configs
                            (create: for all clients; add-client: this client only)
  --i1 CPS                  AmneziaWG I1 signature packet (awg* only, at most 2048 chars)
  --mimic PROFILE           generate an I1-I5 mimicry chain for client configs:
                            dns|stun|ntp|ssdp|dtls (awg* only; not with --i1)
  --id ID                   client id [a-zA-Z0-9_.-]{1,64}, unique within an interface
  --allowed-ips CIDR[,CIDR] AllowedIPs of the client config, e.g. 0.0.0.0/0,::/0
  --psk                     generate a PresharedKey for the client
  --qr                      also print the client config as a QR code
  --amnezia-key             also print the client config as an AmneziaVPN key (vpn://...)
  --keenetic                also print KeeneticOS CLI commands for a Keenetic router
                            (wg and awg 1.0/1.5/2.0 tunnels, IPv4 only)
  --json                    JSON output instead of a table (list, list-client, udp-listener, udp-checker)
  --yes                     do not ask for confirmation (delete)
  --timeout DURATION        packet timeout, e.g. 2s or 500ms (default 2s)
  --count N                 packets of each type per port, 1..100 (default 3)
  --echo-size N             echo payload size, 1..1400 (default 64, udp-checker only)
  --workers N               concurrent port checks, 1..100 (default 32, udp-checker only)
  --verbose                 diagnostic progress (udp-listener, udp-checker; stderr)

Flags per action:
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
  --doctor         (no flags; works without root)
  --udp-listener   --port [--timeout] [--count] [--json] [--verbose]
  --udp-checker    --address --port [--timeout] [--count] [--echo-size] [--workers] [--json] [--verbose]

Exit codes:
  0 success                 5 conflict (already exists, address or port in use)
  1 internal error          6 environment (not root, missing tools, AmneziaWG version)
  2 usage error             7 could not acquire the lock
  3 invalid value           10 canceled by user
  4 not found

Root privileges:
  Not required for --help, --version, --doctor, or --udp-checker.
  --udp-listener requires root only for ports below 1024 or when
  /proc/net/udp and /proc/net/udp6 are not readable.
  All other actions require root.
`
