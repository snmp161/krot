package cli

// Usage возвращает текст справки (--help).
func Usage() string {
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
  --help            show this help
  --version         show version, commit and build date

Flags (--flag value or --flag=value):
  --type wg|awg|awg2|awg3   protocol type (create: required; list: filter)
  --interface NAME          interface name, up to 15 chars [a-zA-Z0-9_.=+-]
  --ip CIDR[,CIDR]          create: server addresses (10.8.0.1/24,fd00:8::1/64);
                            add-client: client addresses (/32, /128);
                            list: filter by address
  --port N                  UDP port 1..65535 (create: required; list: filter)
  --endpoint HOST           public server address for client configs
                            (create: required; add-client: per-client override)
  --mtu N                   MTU 576..9000 (default: wg 1420, awg/awg2 1376, awg3 1280)
  --dns IP[,IP]             DNS servers for client configs
                            (create: for all clients; add-client: this client only)
  --i1 CPS                  AmneziaWG I1 signature packet (awg* only, at most 2048 chars)
  --id ID                   client id [a-zA-Z0-9_.-]{1,64}, unique within an interface
  --allowed-ips CIDR[,CIDR] AllowedIPs of the client config, e.g. 0.0.0.0/0,::/0
  --psk                     generate a PresharedKey for the client
  --qr                      also print the client config as a QR code
  --amnezia-key             also print the client config as an AmneziaVPN key (vpn://...)
  --json                    JSON output instead of a table (list, list-client)
  --yes                     do not ask for confirmation (delete)

Flags per action:
  --list           [--type] [--interface] [--ip] [--port] [--json]
  --create         --type --ip --port --endpoint [--interface] [--mtu] [--dns] [--i1]
  --delete         --interface [--yes]
  --up, --down     --interface
  --add-client     --interface --id --allowed-ips [--ip] [--endpoint] [--psk] [--qr]
                   [--dns] [--amnezia-key]
  --delete-client  --interface --id
  --lock-client    --interface --id
  --unlock-client  --interface --id
  --list-client    --interface [--id] [--json]

Exit codes:
  0 success                 5 conflict (already exists, address or port in use)
  1 internal error          6 environment (not root, missing tools, AmneziaWG version)
  2 usage error             7 could not acquire the lock
  3 invalid value           10 cancelled by user
  4 not found

All actions except --help and --version require root.
`
