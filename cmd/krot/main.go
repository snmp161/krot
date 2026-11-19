// Command krot manages WireGuard and AmneziaWG tunnels on Linux servers.
package main

import (
	"os"

	"github.com/snmp161/krot/internal/app"
	"github.com/snmp161/krot/internal/i18n"
)

// Заполняются при сборке через -ldflags "-X main.version=... -X main.commit=... -X main.date=...".
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	i18n.Configure(os.Getenv)
	os.Exit(app.New(version, commit, date).Run(os.Args[1:]))
}
