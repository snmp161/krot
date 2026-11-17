package app

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/snmp161/krot/internal/cli"
	"github.com/snmp161/krot/internal/errs"
	"github.com/snmp161/krot/internal/portcheck"
)

// SPEC 6.14: --udp-listener.
func (a *App) udpListener(cmd *cli.Command) error {
	ports := cmd.PortRange.Ports()
	if err := a.checkUDPPortsFree(ports); err != nil {
		return err
	}

	listener := portcheck.NewListener(ports)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	if cmd.JSON {
		a.printf("{\"ports\":%s,\"status\":\"listening\"}\n", cmd.PortRange.String())
	} else {
		a.printf("Listening on UDP %s (Ctrl+C to stop)\n", cmd.PortRange.String())
	}

	if err := listener.Listen(ctx); err != nil && err != context.Canceled {
		return errs.Wrap(errs.Internal, err)
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
