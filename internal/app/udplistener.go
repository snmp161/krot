package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/snmp161/krot/internal/cli"
	"github.com/snmp161/krot/internal/errs"
	"github.com/snmp161/krot/internal/i18n"
	"github.com/snmp161/krot/internal/portcheck"
)

// listenerStart — стартовая строка `--udp-listener --json` (SPEC 6.14).
type listenerStart struct {
	PortStart int    `json:"port_start"`
	PortEnd   int    `json:"port_end"`
	Status    string `json:"status"`
}

// SPEC 6.14: --udp-listener работает до SIGINT/SIGTERM.
func (a *App) udpListener(cmd *cli.Command) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return a.runUDPListener(ctx, cmd)
}

func (a *App) runUDPListener(ctx context.Context, cmd *cli.Command) error {
	ports := cmd.PortRange.Ports()
	if err := a.checkUDPListenerRoot(ports); err != nil {
		return err
	}
	if err := a.checkUDPPortsFree(ports); err != nil {
		return err
	}

	listener := portcheck.NewListener(ports)
	if cmd.Verbose {
		listener.OnRequest = func(client *net.UDPAddr, port int, request string) {
			fmt.Fprint(a.Stderr, i18n.Sprintf("UDP request from %s to port %d: %s\n", client, port, request))
		}
	}
	if cmd.JSON {
		b, err := json.Marshal(listenerStart{PortStart: cmd.PortRange.Start, PortEnd: cmd.PortRange.End, Status: "listening"})
		if err != nil {
			return err
		}
		fmt.Fprintf(a.Stdout, "%s\n", b)
	} else {
		a.printf("Listening on UDP %s (Ctrl+C to stop)\n", cmd.PortRange.String())
	}

	// После отмены ctx Listen возвращает ctx.Err() — это штатное завершение.
	if err := listener.Listen(ctx); err != nil && ctx.Err() == nil {
		return errs.Wrap(errs.Internal, err)
	}
	return nil
}

// checkUDPListenerRoot проверяет, нужен ли root для listener (SPEC 2.4, 6.14):
// root требуется для привилегированных портов (<1024) или если /proc/net/udp{,6}
// недоступны текущему пользователю для чтения.
func (a *App) checkUDPListenerRoot(ports []int) error {
	if a.Euid() == 0 {
		return nil
	}
	for _, p := range ports {
		if p < 1024 {
			return errs.Envf("root privileges are required: UDP port %d is privileged (<1024)", p)
		}
	}
	for _, f := range []string{"udp", "udp6"} {
		path := filepath.Join(a.Paths.ProcNet, f)
		file, err := os.Open(path) // #nosec G304 -- path is /proc/net/udp{,6} from fixed Paths.ProcNet
		if err != nil {
			return errs.Envf("root privileges are required: cannot read %s", path)
		}
		_ = file.Close() // #nosec G104 -- readability check only, close error is not relevant
	}
	return nil
}

// checkUDPPortsFree проверяет, что порты не заняты другим конфигом krot
// и не слушаются в системе (SPEC 6.14).
func (a *App) checkUDPPortsFree(ports []int) error {
	tunnels, err := a.scanTunnels()
	if err != nil {
		return err
	}
	for _, port := range ports {
		for _, t := range tunnels {
			if ifc, ok := t.conf.Interface(); ok && ifc.Get("ListenPort") == strconv.Itoa(port) {
				return errs.Conflictf("UDP port %d is already used by interface %s (%s)", port, t.name, t.path)
			}
		}
		for _, f := range []string{"udp", "udp6"} {
			if listened(filepath.Join(a.Paths.ProcNet, f), port) {
				return errs.Conflictf("UDP port %d is already in use in the system", port)
			}
		}
	}
	return nil
}
