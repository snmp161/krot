//go:build integration

// Интеграционные тесты krot (SPEC 8.3). Требуют root и systemd. Работают только внутри
// отдельных network namespace: veth создаётся сразу в них, unit'ы wg-quick@/awg-quick@
// тестовых интерфейсов попадают в netns через drop-in NetworkNamespacePath. Сеть хоста
// (корневой netns) не затрагивается. Локальный запуск — scripts/integration-docker.sh.
package integration

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	nsSrv    = "krot-it-srv"
	nsCli    = "krot-it-cli"
	srvAddr  = "192.0.2.1" // TEST-NET-1, только внутри netns
	cliAddr  = "192.0.2.2"
	prefix   = "krtit" // имена серверных интерфейсов
	cliIface = "krtc0" // имя клиентского интерфейса (отличается: UAPI-сокеты amneziawg общие)
	wgDir    = "/etc/wireguard"
	awgDir   = "/etc/amnezia/amneziawg"
	unitDir  = "/etc/systemd/system"
)

var krotBin string

func TestMain(m *testing.M) {
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "integration tests require root (use scripts/integration-docker.sh)")
		os.Exit(1)
	}
	if _, err := os.Stat("/run/systemd/system"); err != nil {
		fmt.Fprintln(os.Stderr, "integration tests require a running systemd")
		os.Exit(1)
	}
	if err := refuseLeftovers(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	dir, err := os.MkdirTemp("", "krot-it-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	krotBin = os.Getenv("KROT_BIN")
	if krotBin == "" {
		krotBin = filepath.Join(dir, "krot")
		out, err := exec.Command("go", "build", "-o", krotBin, "../../cmd/krot").CombinedOutput()
		if err != nil {
			fmt.Fprintf(os.Stderr, "build krot: %v\n%s", err, out)
			os.Exit(1)
		}
	}
	if err := setupNetns(); err != nil {
		fmt.Fprintln(os.Stderr, "setup netns:", err)
		teardownNetns()
		os.Exit(1)
	}
	code := m.Run()
	teardownNetns()
	os.RemoveAll(dir)
	os.Exit(code)
}

// refuseLeftovers — тесты не трогают чужое: если тестовые объекты уже есть, они не запускаются.
func refuseLeftovers() error {
	for _, d := range []string{wgDir, awgDir} {
		if m, _ := filepath.Glob(filepath.Join(d, prefix+"*.conf")); len(m) > 0 {
			return fmt.Errorf("leftovers from a previous run: %v; remove them first", m)
		}
	}
	for _, ns := range []string{nsSrv, nsCli} {
		if _, err := os.Stat("/run/netns/" + ns); err == nil {
			return fmt.Errorf("network namespace %s already exists; remove it first (ip netns del %s)", ns, ns)
		}
	}
	return nil
}

func run(name string, args ...string) (string, error) {
	var out bytes.Buffer
	cmd := exec.Command(name, args...)
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	if err != nil {
		return out.String(), fmt.Errorf("%s %s: %v: %s", name, strings.Join(args, " "), err, out.String())
	}
	return out.String(), nil
}

func setupNetns() error {
	steps := [][]string{
		{"ip", "netns", "add", nsSrv},
		{"ip", "netns", "add", nsCli},
		// veth создаётся сразу внутри namespace'ов — корневой netns не меняется.
		{"ip", "link", "add", "veth-it-srv", "netns", nsSrv, "type", "veth", "peer", "name", "veth-it-cli", "netns", nsCli},
		// IPv6 внутри тестовых netns (sysctl net.* — свои у каждого netns).
		{"ip", "netns", "exec", nsSrv, "sysctl", "-q", "-w", "net.ipv6.conf.all.disable_ipv6=0", "net.ipv6.conf.default.disable_ipv6=0"},
		{"ip", "netns", "exec", nsCli, "sysctl", "-q", "-w", "net.ipv6.conf.all.disable_ipv6=0", "net.ipv6.conf.default.disable_ipv6=0"},
		{"ip", "-n", nsSrv, "link", "set", "lo", "up"},
		{"ip", "-n", nsCli, "link", "set", "lo", "up"},
		{"ip", "-n", nsSrv, "addr", "add", srvAddr + "/24", "dev", "veth-it-srv"},
		{"ip", "-n", nsCli, "addr", "add", cliAddr + "/24", "dev", "veth-it-cli"},
		{"ip", "-n", nsSrv, "link", "set", "veth-it-srv", "up"},
		{"ip", "-n", nsCli, "link", "set", "veth-it-cli", "up"},
	}
	for _, s := range steps {
		if _, err := run(s[0], s[1:]...); err != nil {
			return err
		}
	}
	_, err := run("ip", "netns", "exec", nsCli, "ping", "-c", "1", "-W", "2", srvAddr)
	return err
}

func teardownNetns() {
	run("ip", "netns", "del", nsCli)
	run("ip", "netns", "del", nsSrv)
}

// krot запускает krot в серверном netns.
func krot(args ...string) (stdout, stderr string, code int) {
	var o, e bytes.Buffer
	cmd := exec.Command("ip", append([]string{"netns", "exec", nsSrv, krotBin}, args...)...)
	cmd.Stdout, cmd.Stderr = &o, &e
	err := cmd.Run()
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		code = ee.ExitCode()
	} else if err != nil {
		code = -1
	}
	return o.String(), e.String(), code
}

func mustKrot(t *testing.T, args ...string) string {
	t.Helper()
	out, errOut, code := krot(args...)
	if code != 0 {
		t.Fatalf("krot %s: exit %d\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), code, out, errOut)
	}
	return out
}

func mustRun(t *testing.T, name string, args ...string) string {
	t.Helper()
	out, err := run(name, args...)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// tunnel — серверный туннель krot в netns krot-it-srv.
type tunnel struct {
	typ, name, tool, quick, unit, conf string
	subnet                             string // "10.66.N"
	port                               int
}

func newTunnel(t *testing.T, typ string, n int) *tunnel {
	t.Helper()
	tu := &tunnel{typ: typ, name: fmt.Sprintf("%s%d", prefix, n), subnet: fmt.Sprintf("10.66.%d", n), port: 51900 + n}
	if typ == "wg" {
		tu.tool, tu.quick, tu.conf = "wg", "wg-quick", filepath.Join(wgDir, tu.name+".conf")
	} else {
		tu.tool, tu.quick, tu.conf = "awg", "awg-quick", filepath.Join(awgDir, tu.name+".conf")
	}
	tu.unit = tu.quick + "@" + tu.name
	addDropIn(t, tu.unit)
	t.Cleanup(func() {
		run("systemctl", "disable", "--now", tu.unit)
		os.Remove(tu.conf)
	})
	return tu
}

// addDropIn помещает unit в серверный netns. Drop-in лежит в /etc: если netns пропадёт
// (например, после перезагрузки), unit не запустится вместо того, чтобы тронуть сеть хоста.
func addDropIn(t *testing.T, unit string) {
	t.Helper()
	dir := filepath.Join(unitDir, unit+".service.d")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	conf := "[Service]\nNetworkNamespacePath=/run/netns/" + nsSrv + "\n"
	if err := os.WriteFile(filepath.Join(dir, "krot-it-netns.conf"), []byte(conf), 0o644); err != nil {
		t.Fatal(err)
	}
	mustRun(t, "systemctl", "daemon-reload")
	t.Cleanup(func() {
		os.RemoveAll(dir)
		run("systemctl", "daemon-reload")
	})
}

func (tu *tunnel) create(t *testing.T, ip string, extra ...string) {
	t.Helper()
	args := []string{"--create", "--type", tu.typ, "--interface", tu.name, "--ip", ip,
		"--port", strconv.Itoa(tu.port), "--endpoint", srvAddr}
	mustKrot(t, append(args, extra...)...)
}

func (tu *tunnel) serverIP() string { return tu.subnet + ".1" }

func (tu *tunnel) addClient(t *testing.T, id string, extra ...string) string {
	t.Helper()
	args := []string{"--add-client", "--interface", tu.name, "--id", id, "--allowed-ips", tu.subnet + ".0/24"}
	return mustKrot(t, append(args, extra...)...)
}

// clientUp поднимает клиентский интерфейс из выданного конфига в клиентском netns.
func (tu *tunnel) clientUp(t *testing.T, cfg string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, cliIface+".conf")
	if err := os.WriteFile(path, []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	mustRun(t, "ip", "netns", "exec", nsCli, tu.quick, "up", path)
	t.Cleanup(func() { run("ip", "netns", "exec", nsCli, tu.quick, "down", path) })
}

func ping(args ...string) bool {
	_, err := run("ip", append([]string{"netns", "exec", nsCli, "ping", "-n"}, args...)...)
	return err == nil
}

func pingOnce(addr string) bool { return ping("-c", "1", "-W", "1", addr) }

// waitPing ждёт, пока ping через туннель пройдёт (handshake может занять несколько секунд).
func waitPing(t *testing.T, addr string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for !pingOnce(addr) {
		if time.Now().After(deadline) {
			t.Fatalf("no ping to %s within %s", addr, timeout)
		}
	}
}

func expectNoPing(t *testing.T, addr string) {
	t.Helper()
	if ping("-c", "3", "-i", "0.3", "-W", "1", addr) {
		t.Fatalf("ping to %s must fail", addr)
	}
}

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// srvShow — `wg|awg show <if> <what>` в серверном netns.
func (tu *tunnel) srvShow(t *testing.T, what string) string {
	t.Helper()
	return mustRun(t, "ip", "netns", "exec", nsSrv, tu.tool, "show", tu.name, what)
}

var ifindexRe = regexp.MustCompile(`^(\d+):`)

func (tu *tunnel) ifindex(t *testing.T) int {
	t.Helper()
	out := mustRun(t, "ip", "-n", nsSrv, "-o", "link", "show", "dev", tu.name)
	m := ifindexRe.FindStringSubmatch(out)
	if m == nil {
		t.Fatalf("cannot parse ifindex from %q", out)
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

// handshake — latest-handshake peer'а с данным ключом.
func (tu *tunnel) handshake(t *testing.T, pub string) int64 {
	t.Helper()
	for _, l := range strings.Split(tu.srvShow(t, "latest-handshakes"), "\n") {
		f := strings.Fields(l)
		if len(f) == 2 && f[0] == pub {
			n, _ := strconv.ParseInt(f[1], 10, 64)
			return n
		}
	}
	t.Fatalf("peer %s not in the live interface", pub)
	return 0
}

func (tu *tunnel) livePeers(t *testing.T) []string {
	t.Helper()
	return strings.Fields(tu.srvShow(t, "peers"))
}

type listItem struct {
	Interface     string `json:"interface"`
	Type          string `json:"type"`
	Status        string `json:"status"`
	ClientsTotal  int    `json:"clients_total"`
	ClientsLocked int    `json:"clients_locked"`
}

func (tu *tunnel) list(t *testing.T) listItem {
	t.Helper()
	var items []listItem
	out := mustKrot(t, "--list", "--json", "--interface", tu.name)
	if err := json.Unmarshal([]byte(out), &items); err != nil || len(items) != 1 {
		t.Fatalf("--list: %v %s", err, out)
	}
	return items[0]
}

type clientItem struct {
	ID        string   `json:"id"`
	PublicKey string   `json:"public_key"`
	Addresses []string `json:"addresses"`
	Status    string   `json:"status"`
}

func (tu *tunnel) clients(t *testing.T) []clientItem {
	t.Helper()
	var items []clientItem
	out := mustKrot(t, "--list-client", "--interface", tu.name, "--json")
	if err := json.Unmarshal([]byte(out), &items); err != nil {
		t.Fatalf("--list-client: %v %s", err, out)
	}
	return items
}

func (tu *tunnel) clientKey(t *testing.T, id string) string {
	t.Helper()
	for _, c := range tu.clients(t) {
		if c.ID == id {
			return c.PublicKey
		}
	}
	t.Fatalf("client %s not listed", id)
	return ""
}
