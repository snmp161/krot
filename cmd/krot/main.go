// Command krot manages WireGuard and AmneziaWG tunnels on Linux servers.
package main

import (
	"fmt"
	"os"
)

// Заполняются при сборке через -ldflags "-X main.version=... -X main.commit=... -X main.date=...".
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	fmt.Fprintf(os.Stderr, "krot %s (%s, %s): not implemented yet\n", version, commit, date)
	os.Exit(1)
}
