package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/snmp161/krot/internal/cli"
	"github.com/snmp161/krot/internal/errs"
	"github.com/snmp161/krot/internal/i18n"
	"github.com/snmp161/krot/internal/portcheck"
)

// SPEC 6.15: --udp-checker.
func (a *App) udpChecker(cmd *cli.Command) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	return a.runUDPChecker(ctx, cmd)
}

func (a *App) runUDPChecker(ctx context.Context, cmd *cli.Command) error {
	timeout := cmd.Timeout
	if timeout == 0 {
		timeout = 2 * time.Second
	}
	count := cmd.Count
	if count == 0 {
		count = 3
	}
	echoSize := cmd.EchoSize
	if echoSize == 0 {
		echoSize = 64
	}
	workers := cmd.Workers
	if workers == 0 {
		workers = portcheck.DefaultWorkers
	}

	checker := &portcheck.Checker{
		Address:  cmd.Address,
		Ports:    cmd.PortRange.Ports(),
		Timeout:  timeout,
		Count:    count,
		EchoSize: echoSize,
		Workers:  workers,
		Rand:     a.Rand,
	}
	if cmd.Verbose {
		fmt.Fprint(a.Stderr, i18n.Sprintf("Checking UDP ports %s on %s with %d workers\n", cmd.PortRange, cmd.Address, workers))
		checker.OnResult = func(r portcheck.Result) {
			if r.Detail != "" {
				fmt.Fprint(a.Stderr, i18n.Sprintf("UDP port %d: %s (%s)\n", r.Port, r.Status, localizedUDPDetail(r.Detail)))
				return
			}
			fmt.Fprint(a.Stderr, i18n.Sprintf("UDP port %d: %s\n", r.Port, r.Status))
		}
	}
	results, checkErr := checker.Check(ctx)
	if checkErr != nil && !errors.Is(checkErr, context.Canceled) {
		return checkErr
	}
	if cmd.JSON {
		if err := a.printUDPCheckerJSON(results); err != nil {
			return err
		}
	} else {
		if err := a.printUDPCheckerTable(results); err != nil {
			return err
		}
	}
	if checkErr != nil {
		return errs.Cancelledf("UDP check cancelled; partial results for %d of %d ports", len(results), len(cmd.PortRange.Ports()))
	}
	return nil
}

func (a *App) printUDPCheckerTable(results []portcheck.Result) error {
	w := tabwriter.NewWriter(a.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, i18n.Text("PORT\tSTATUS\tPING\tECHO\tTIME (min/avg/max)\tDETAIL"))
	for _, r := range results {
		timeCol := portcheck.FormatRTT(r)
		if r.Status == "filtered" || r.Status == "error" {
			timeCol = "-"
		}
		fmt.Fprintf(w, "%d\t%s\t%d/%d\t%d/%d\t%s\t%s\n",
			r.Port, r.Status, r.PingOK, r.PingSent, r.EchoOK, r.EchoSent, timeCol, localizedUDPDetail(r.Detail))
	}
	return w.Flush()
}

func localizedUDPDetail(detail string) string {
	if detail == "timeout" {
		return i18n.Text(detail)
	}
	return detail
}

func (a *App) printUDPCheckerJSON(results []portcheck.Result) error {
	out := make([]map[string]any, len(results))
	for i, r := range results {
		out[i] = map[string]any{
			"port":       r.Port,
			"status":     r.Status,
			"ping_sent":  r.PingSent,
			"ping_ok":    r.PingOK,
			"echo_sent":  r.EchoSent,
			"echo_ok":    r.EchoOK,
			"time_sent":  r.TimeSent,
			"time_ok":    r.TimeOK,
			"rtt_min_ms": r.RTTMin.Milliseconds(),
			"rtt_avg_ms": r.RTTAvg.Milliseconds(),
			"rtt_max_ms": r.RTTMax.Milliseconds(),
			"detail":     r.Detail,
		}
	}
	enc := json.NewEncoder(a.Stdout)
	return enc.Encode(out)
}
