package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/netip"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/snmp161/krot/internal/awg"
	"github.com/snmp161/krot/internal/cli"
	"github.com/snmp161/krot/internal/conf"
	"github.com/snmp161/krot/internal/errs"
)

// params возвращает AWG-параметры [Interface] с каноническими именами.
func params(ifc conf.Interface) awg.Params {
	var p awg.Params
	for _, kv := range ifc.Keys {
		if k, ok := awg.CanonicalKey(kv.Key); ok {
			p = append(p, awg.Param{Key: k, Value: kv.Value})
		}
	}
	return p
}

// tunnelInfo — строка --list (SPEC 6.1).
type tunnelInfo struct {
	Interface     string   `json:"interface"`
	Type          string   `json:"type"`
	Version       string   `json:"version"`
	Addresses     []string `json:"addresses"`
	Port          *int     `json:"port"`
	Status        string   `json:"status"`
	ClientsTotal  int      `json:"clients_total"`
	ClientsLocked int      `json:"clients_locked"`
}

func describe(t *tunnel) tunnelInfo {
	ifc, _ := t.conf.Interface()
	typ, ver := awg.Detect(t.amnezia, ifc.Meta["type"], params(ifc))
	info := tunnelInfo{
		Interface: t.name,
		Type:      string(typ),
		Version:   ver,
		Addresses: append([]string{}, ifc.List("Address")...),
		Status:    "down",
	}
	if port, err := strconv.Atoi(ifc.Get("ListenPort")); err == nil {
		info.Port = &port
	}
	for _, p := range t.conf.Peers() {
		info.ClientsTotal++
		if p.Locked {
			info.ClientsLocked++
		}
	}
	return info
}

// matches — фильтры --type, --interface, --ip, --port объединяются по И (SPEC 6.1).
func (info tunnelInfo) matches(cmd *cli.Command) bool {
	if cmd.Has("type") && info.Type != string(cmd.Type) {
		return false
	}
	if cmd.Has("interface") && info.Interface != cmd.Interface {
		return false
	}
	if cmd.Has("port") && (info.Port == nil || *info.Port != cmd.Port) {
		return false
	}
	if cmd.Has("ip") {
		hit := false
		for _, s := range info.Addresses {
			if p, err := netip.ParsePrefix(s); err == nil {
				for _, f := range cmd.IPs {
					hit = hit || f.Overlaps(p)
				}
			}
		}
		if !hit {
			return false
		}
	}
	return true
}

// SPEC 6.1: --list.
func (a *App) list(cmd *cli.Command) error {
	tunnels, err := a.scanTunnels()
	if err != nil {
		return err
	}
	var infos []tunnelInfo
	live := map[string][]string{} // утилита → живые интерфейсы; опрашиваются только при наличии конфигов
	for _, t := range tunnels {
		info := describe(t)
		if !info.matches(cmd) {
			continue
		}
		names, ok := live[t.tool()]
		if !ok {
			if err := a.need(t.tool()); err != nil {
				return err
			}
			if names, err = a.liveInterfaces(t.tool()); err != nil {
				return err
			}
			live[t.tool()] = names
		}
		if slices.Contains(names, t.name) {
			info.Status = "up"
		}
		infos = append(infos, info)
	}
	if cmd.JSON {
		return printJSON(a.Stdout, infos)
	}
	w := tabwriter.NewWriter(a.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "INTERFACE\tTYPE\tVERSION\tADDRESS\tPORT\tSTATUS\tCLIENTS")
	for _, i := range infos {
		port := "-"
		if i.Port != nil {
			port = strconv.Itoa(*i.Port)
		}
		clients := strconv.Itoa(i.ClientsTotal)
		if i.ClientsLocked > 0 {
			clients += fmt.Sprintf(" (%d locked)", i.ClientsLocked)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", i.Interface, i.Type, i.Version,
			dashIfEmpty(strings.Join(i.Addresses, ",")), port, i.Status, clients)
	}
	return w.Flush()
}

// clientInfo — строка --list-client (SPEC 6.9).
type clientInfo struct {
	ID              string   `json:"id"`
	PublicKey       string   `json:"public_key"`
	Addresses       []string `json:"addresses"`
	Status          string   `json:"status"`
	LatestHandshake *int64   `json:"latest_handshake"`
	RxBytes         int64    `json:"rx_bytes"`
	TxBytes         int64    `json:"tx_bytes"`
	Endpoint        *string  `json:"endpoint"`
	Created         *string  `json:"created"`
	live            bool     // есть данные из `show dump`
}

// peerStat — строка peer'а из `wg|awg show <if> dump`.
type peerStat struct {
	endpoint  string
	handshake int64
	rx, tx    int64
}

// parseDump разбирает `show <if> dump`: первая строка — интерфейс, далее peer'ы
// (public-key, preshared-key, endpoint, allowed-ips, latest-handshake, rx, tx, keepalive).
func parseDump(out string) map[string]peerStat {
	stats := map[string]peerStat{}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	for _, l := range lines[min(1, len(lines)):] {
		f := strings.Split(l, "\t")
		if len(f) < 8 {
			continue
		}
		hs, _ := strconv.ParseInt(f[4], 10, 64)
		rx, _ := strconv.ParseInt(f[5], 10, 64)
		tx, _ := strconv.ParseInt(f[6], 10, 64)
		stats[f[0]] = peerStat{endpoint: f[2], handshake: hs, rx: rx, tx: tx}
	}
	return stats
}

// SPEC 6.9: --list-client.
func (a *App) listClient(cmd *cli.Command) error {
	t, err := a.findTunnel(cmd.Interface)
	if err != nil {
		return err
	}
	if err := a.need(t.tool()); err != nil {
		return err
	}
	if cmd.Has("id") {
		if _, ok := t.conf.PeerByID(cmd.ID); !ok {
			return errs.NotFoundf("client %s not found on interface %s", cmd.ID, t.name)
		}
	}
	up, err := a.isUp(t)
	if err != nil {
		return err
	}
	stats := map[string]peerStat{}
	if up {
		r, err := a.Exec.Run(t.tool(), "show", t.name, "dump")
		if err != nil {
			return fmt.Errorf("cannot read the state of %s: %w", t.name, err)
		}
		stats = parseDump(string(r.Stdout))
	}
	var infos []clientInfo
	for _, p := range t.conf.Peers() {
		if cmd.Has("id") && p.ID != cmd.ID {
			continue
		}
		info := clientInfo{ID: dashIfEmpty(p.ID), PublicKey: p.PublicKey(),
			Addresses: append([]string{}, p.AllowedIPs()...), Status: "active"}
		if p.Locked {
			info.Status = "locked"
		}
		if p.Created != "" {
			created := p.Created
			info.Created = &created
		}
		if s, ok := stats[p.PublicKey()]; ok && !p.Locked {
			info.live = true
			info.RxBytes, info.TxBytes = s.rx, s.tx
			if s.handshake > 0 {
				hs := s.handshake
				info.LatestHandshake = &hs
			}
			if s.endpoint != "" && s.endpoint != "(none)" {
				ep := s.endpoint
				info.Endpoint = &ep
			}
		}
		infos = append(infos, info)
	}
	if cmd.JSON {
		return printJSON(a.Stdout, infos)
	}
	now := a.Now()
	w := tabwriter.NewWriter(a.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tPUBLIC KEY\tADDRESS\tSTATUS\tLAST HANDSHAKE\tRX\tTX")
	for _, i := range infos {
		hs, rx, tx := "-", "-", "-"
		if i.live {
			rx, tx = humanBytes(i.RxBytes), humanBytes(i.TxBytes)
			if i.LatestHandshake != nil {
				hs = humanAgo(*i.LatestHandshake, now)
			}
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", i.ID, shortKey(i.PublicKey),
			dashIfEmpty(strings.Join(i.Addresses, ",")), i.Status, hs, rx, tx)
	}
	return w.Flush()
}

// printJSON выводит массив; пустой результат — [] (не null).
func printJSON[T any](w io.Writer, v []T) error {
	if v == nil {
		v = []T{}
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "%s\n", b)
	return err
}

func dashIfEmpty(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// shortKey сокращает ключ до вида "aB3d…Xy9=".
func shortKey(k string) string {
	if len(k) <= 9 {
		return dashIfEmpty(k)
	}
	return k[:4] + "…" + k[len(k)-4:]
}

// humanBytes — размер в единицах IEC: 512B, 1.5KiB, 12.4MiB.
func humanBytes(n int64) string {
	if n < 1024 {
		return strconv.FormatInt(n, 10) + "B"
	}
	units := []string{"KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	v := float64(n) / 1024
	i := 0
	for v >= 1024 && i < len(units)-1 {
		v /= 1024
		i++
	}
	return fmt.Sprintf("%.1f%s", v, units[i])
}

// humanAgo — «сколько назад» для unix-времени; 0 — handshake не было.
func humanAgo(ts int64, now time.Time) string {
	if ts == 0 {
		return "-"
	}
	d := max(now.Unix()-ts, 0)
	switch {
	case d < 60:
		return fmt.Sprintf("%ds ago", d)
	case d < 3600:
		return fmt.Sprintf("%dm ago", d/60)
	case d < 86400:
		return fmt.Sprintf("%dh ago", d/3600)
	}
	return fmt.Sprintf("%dd ago", d/86400)
}
