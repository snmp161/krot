package app

import (
	"context"
	"encoding/json"
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/snmp161/krot/internal/cli"
	"github.com/snmp161/krot/internal/portcheck"
)

// SPEC 6.15: --udp-checker.
func (a *App) udpChecker(cmd *cli.Command) error {
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
		fmt.Fprintf(a.Stderr, "Checking UDP ports %s on %s with %d workers\n", cmd.PortRange, cmd.Address, workers)
		checker.OnResult = func(r portcheck.Result) {
			if r.Detail != "" {
				fmt.Fprintf(a.Stderr, "UDP port %d: %s (%s)\n", r.Port, r.Status, r.Detail)
				return
			}
			fmt.Fprintf(a.Stderr, "UDP port %d: %s\n", r.Port, r.Status)
		}
	}
	results, err := checker.Check(context.Background())
	if err != nil {
		return err
	}
	if cmd.JSON {
		return a.printUDPCheckerJSON(results)
	}
	return a.printUDPCheckerTable(results)
}

func (a *App) printUDPCheckerTable(results []portcheck.Result) error {
	w := tabwriter.NewWriter(a.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PORT\tSTATUS\tPING\tECHO\tTIME (min/avg/max)\tDETAIL")
	for _, r := range results {
		timeCol := portcheck.FormatRTT(r)
		if r.Status == "filtered" || r.Status == "error" {
			timeCol = "-"
		}
		fmt.Fprintf(w, "%d\t%s\t%d/%d\t%d/%d\t%s\t%s\n",
			r.Port, r.Status, r.PingOK, r.PingSent, r.EchoOK, r.EchoSent, timeCol, r.Detail)
	}
	return w.Flush()
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
