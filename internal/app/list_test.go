package app

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/sovchinnikov/krot/internal/conf"
	"github.com/sovchinnikov/krot/internal/errs"
)

// stripMeta убирает строки "# krot:" — получается «чужой» конфиг.
func stripMeta(s string) string {
	var out []string
	for _, l := range strings.SplitAfter(s, "\n") {
		if !strings.HasPrefix(l, "# krot:") {
			out = append(out, l)
		}
	}
	return strings.Join(out, "")
}

func setPortAddr(s, port, addr string) string {
	c := conf.Parse([]byte(s))
	ifc, _ := c.Interface()
	s = strings.Replace(s, "ListenPort = "+ifc.Get("ListenPort"), "ListenPort = "+port, 1)
	return strings.Replace(s, "Address = "+strings.Join(ifc.List("Address"), ", "), "Address = "+addr, 1)
}

const awg30 = `[Interface]
PrivateKey = yAnz5TF+lXXJte14tji3zlMNq+hd2rYUIgJBgB3fBmk=
Address = 10.14.0.1/24
ListenPort = 51825
Jc = 5
Jmin = 64
Jmax = 400
S1 = 20
S2 = 20
S3 = 20
S4 = 20
H1 = 1
H2 = 2
H3 = 3
H4 = 4
HeaderProtectionKey = MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=
RekeyAfterTime = 100-130
`

// listEnv — набор туннелей всех видов (SPEC 3.2, 5.3, 6.1).
func listEnv(t *testing.T) *env {
	e := newEnv(t)
	e.write(e.awgConf("awg0"), fixture(t, "awg0_two_clients.conf"))
	e.write(e.wgConf("wg0"), fixture(t, "wg0_empty.conf"))
	e.write(e.wgConf("office"), setPortAddr(fixture(t, "foreign_wg.conf"), "51822", "10.11.0.1/24, fd00:11::1/64"))
	e.write(e.awgConf("compat0"), setPortAddr(stripMeta(fixture(t, "wg0_empty.conf")), "51823", "10.12.0.1/24"))
	e.write(e.awgConf("awg9"), setPortAddr(fixture(t, "create_awg3.golden"), "51824", "10.13.0.1/24"))
	e.write(e.awgConf("awg30"), awg30)
	e.write(e.awgConf("legacy15"), setPortAddr(stripMeta(fixture(t, "create_awg_i1.golden")), "51826", "10.15.0.1/24"))
	e.write(e.awgConf("legacy10"), setPortAddr(stripMeta(fixture(t, "create_awg.golden")), "51827", "10.16.0.1/24"))
	// Не конфиги: временный файл атомарной записи и файл с недопустимым именем.
	e.write(filepath.Join(e.app.Paths.AWGDir, ".awg0.conf.tmp-123"), "garbage")
	e.write(e.wgConf("name-longer-than-15"), "garbage")
	e.fake.On("awg show interfaces", "awg0 awg9 krtc0\n")
	e.fake.On("wg show interfaces", "office\n")
	return e
}

type listItem struct {
	Interface     string   `json:"interface"`
	Type          string   `json:"type"`
	Version       string   `json:"version"`
	Addresses     []string `json:"addresses"`
	Port          *int     `json:"port"`
	Status        string   `json:"status"`
	ClientsTotal  int      `json:"clients_total"`
	ClientsLocked int      `json:"clients_locked"`
}

func listJSON(t *testing.T, e *env, args ...string) []listItem {
	t.Helper()
	e.mustRun(0, append([]string{"--list", "--json"}, args...)...)
	var items []listItem
	if err := json.Unmarshal(e.stdout.Bytes(), &items); err != nil {
		t.Fatalf("bad JSON: %v\n%s", err, e.stdout)
	}
	if items == nil {
		t.Fatalf("JSON must be an array, got %s", e.stdout)
	}
	return items
}

func names(items []listItem) []string {
	var out []string
	for _, it := range items {
		out = append(out, it.Interface)
	}
	return out
}

// SPEC 6.1, 3.2: таблица и JSON, сортировка по имени, версия, статус, клиенты.
func TestList(t *testing.T) {
	e := listEnv(t)
	e.mustRun(0, "--list")
	golden(t, "list.golden", e.stdout.String())
	if !strings.HasPrefix(e.stdout.String(), "INTERFACE") {
		t.Fatalf("table header missing:\n%s", e.stdout)
	}

	items := listJSON(t, e)
	golden(t, "list.json.golden", e.stdout.String())
	want := []struct {
		name, typ, ver, status string
		total, locked          int
	}{
		{"awg0", "awg2", "2.0", "up", 2, 1},
		{"awg30", "awg3", "3.0", "down", 0, 0},
		{"awg9", "awg3", "3.1", "up", 0, 0},
		{"compat0", "awg", "compat", "down", 0, 0},
		{"legacy10", "awg", "1.0", "down", 0, 0},
		{"legacy15", "awg", "1.5", "down", 0, 0},
		{"office", "wg", "-", "up", 2, 0},
		{"wg0", "wg", "-", "down", 0, 0},
	}
	if len(items) != len(want) {
		t.Fatalf("got %q", names(items))
	}
	for i, w := range want {
		it := items[i]
		if it.Interface != w.name || it.Type != w.typ || it.Version != w.ver || it.Status != w.status ||
			it.ClientsTotal != w.total || it.ClientsLocked != w.locked {
			t.Errorf("item %d = %+v; want %+v", i, it, w)
		}
	}
	if a := items[0]; !slices.Equal(a.Addresses, []string{"10.8.0.1/24", "fd00:8::1/64"}) || a.Port == nil || *a.Port != 51820 {
		t.Errorf("awg0 = %+v", a)
	}
	if !strings.Contains(fmt.Sprint(e.stdout.String()), `"addresses": [`) {
		t.Error("addresses must be an array")
	}
}

// SPEC 6.1: фильтры --type, --interface, --ip, --port объединяются по И.
func TestListFilters(t *testing.T) {
	e := listEnv(t)
	cases := []struct {
		args []string
		want []string
	}{
		{[]string{"--type", "awg2"}, []string{"awg0"}},
		{[]string{"--type", "awg3"}, []string{"awg30", "awg9"}},
		{[]string{"--type", "awg"}, []string{"compat0", "legacy10", "legacy15"}},
		{[]string{"--type", "wg"}, []string{"office", "wg0"}},
		{[]string{"--interface", "wg0"}, []string{"wg0"}},
		{[]string{"--interface", "nope"}, nil},
		{[]string{"--ip", "10.8.0.77"}, []string{"awg0"}},
		{[]string{"--ip", "fd00:11::5"}, []string{"office"}},
		{[]string{"--ip", "10.0.0.0/8,fd00:8::/64"}, []string{"awg0", "awg30", "awg9", "compat0", "legacy10", "legacy15", "office", "wg0"}},
		{[]string{"--ip", "192.0.2.1"}, nil},
		{[]string{"--port", "51821"}, []string{"wg0"}},
		{[]string{"--type", "wg", "--port", "51822"}, []string{"office"}},
		{[]string{"--type", "wg", "--port", "51820"}, nil},
		{[]string{"--type", "awg3", "--ip", "10.13.0.1", "--interface", "awg9", "--port", "51824"}, []string{"awg9"}},
	}
	for _, c := range cases {
		if got := names(listJSON(t, e, c.args...)); !slices.Equal(got, c.want) {
			t.Errorf("%q: got %q; want %q", c.args, got, c.want)
		}
	}
	// Пустой результат: только заголовок / [] и код 0.
	e.mustRun(0, "--list", "--interface", "nope")
	if strings.Count(e.stdout.String(), "\n") != 1 || !strings.HasPrefix(e.stdout.String(), "INTERFACE") {
		t.Fatalf("empty table:\n%q", e.stdout)
	}
	e.mustRun(0, "--list", "--json", "--interface", "nope")
	if e.stdout.String() != "[]\n" {
		t.Fatalf("empty JSON: %q", e.stdout)
	}
}

// Нет каталогов и конфигов — пустой список; утилиты нужны только для существующих конфигов.
func TestListEmptyAndTools(t *testing.T) {
	e := newEnv(t)
	e.fake.Missing = map[string]bool{"awg": true}
	e.mustRun(0, "--list", "--json")
	if e.stdout.String() != "[]\n" {
		t.Fatalf("got %q", e.stdout)
	}
	e.write(e.awgConf("awg0"), fixture(t, "awg0_two_clients.conf"))
	e.mustRun(errs.Env, "--list")
	if !strings.Contains(e.stderr.String(), "amneziawg-tools") {
		t.Fatalf("stderr %q", e.stderr)
	}
	if e.exists(e.app.Paths.LockFile) {
		t.Fatal("--list must not take the lock")
	}
}

func dumpFor(t *testing.T, e *env) string {
	cfg := e.read(e.awgConf("awg0"))
	phone := peerKey(t, cfg, "phone-sergei")
	laptop := peerKey(t, cfg, "laptop-old")
	now := fixedNow.Unix()
	return strings.Join([]string{
		"8PmO8wRo4aJLy+bo7GiQC3n1mSPkbnDAH/sWWs85NW8=\tlbQxp9EyU0L4qKJh2Xb9xX8hX2y9b8sY2b2b2b2b2bY=\t51820\t6\t64\t512\t43\t117\t31\t12\t194847201-195371488\t905128442-905652729\t1503002211-1503526498\t2001009901-2001534188\t(null)\t(null)\t(null)\t(null)\t(null)\toff",
		phone + "\t(hidden)\t203.0.113.50:41000\t10.8.0.2/32,fd00:8::2/128\t" + fmt.Sprint(now-75) + "\t13002342\t1181116006\toff",
		// заблокированный peer в интерфейсе отсутствует; чужой peer из живого интерфейса игнорируется
		laptop[:10] + "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=\t(none)\t(none)\t10.8.0.99/32\t0\t0\t0\toff",
		"",
	}, "\n")
}

type clientItem struct {
	ID              string   `json:"id"`
	PublicKey       string   `json:"public_key"`
	Addresses       []string `json:"addresses"`
	Status          string   `json:"status"`
	LatestHandshake *int64   `json:"latest_handshake"`
	RxBytes         int64    `json:"rx_bytes"`
	TxBytes         int64    `json:"tx_bytes"`
	Endpoint        *string  `json:"endpoint"`
	Created         *string  `json:"created"`
}

// SPEC 6.9: --list-client — данные конфига плюс `awg show <if> dump`.
func TestListClient(t *testing.T) {
	e := newEnv(t)
	setupAWG0(t, e, true)
	e.fake.On("awg show awg0 dump", dumpFor(t, e))

	e.mustRun(0, "--list-client", "--interface", "awg0")
	golden(t, "list_client.golden", e.stdout.String())
	out := e.stdout.String()
	for _, s := range []string{"ID", "PUBLIC KEY", "LAST HANDSHAKE", "phone-sergei", "1m ago", "12.4MiB", "1.1GiB", "laptop-old", "locked", "active"} {
		if !strings.Contains(out, s) {
			t.Errorf("table misses %q:\n%s", s, out)
		}
	}
	phone := peerKey(t, e.read(e.awgConf("awg0")), "phone-sergei")
	if !strings.Contains(out, phone[:4]+"…"+phone[len(phone)-4:]) {
		t.Errorf("abbreviated key missing:\n%s", out)
	}
	assertOnlyPeerOps(t, e, "awg", "awg0")

	e.mustRun(0, "--list-client", "--interface", "awg0", "--json")
	golden(t, "list_client.json.golden", e.stdout.String())
	var items []clientItem
	if err := json.Unmarshal(e.stdout.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("items = %+v", items)
	}
	p, l := items[0], items[1]
	if p.ID != "phone-sergei" || p.PublicKey != phone || p.Status != "active" || p.LatestHandshake == nil ||
		*p.LatestHandshake != fixedNow.Unix()-75 || p.RxBytes != 13002342 || p.TxBytes != 1181116006 ||
		p.Endpoint == nil || *p.Endpoint != "203.0.113.50:41000" || p.Created == nil || *p.Created != "2026-10-17T01:30:00Z" ||
		!slices.Equal(p.Addresses, []string{"10.8.0.2/32", "fd00:8::2/128"}) {
		t.Errorf("phone = %+v", p)
	}
	if l.ID != "laptop-old" || l.Status != "locked" || l.LatestHandshake != nil || l.Endpoint != nil || l.RxBytes != 0 {
		t.Errorf("laptop = %+v", l)
	}

	// Фильтр --id; несуществующий клиент — код 4.
	e.mustRun(0, "--list-client", "--interface", "awg0", "--id", "laptop-old", "--json")
	if err := json.Unmarshal(e.stdout.Bytes(), &items); err != nil || len(items) != 1 || items[0].ID != "laptop-old" {
		t.Fatalf("filtered = %+v, %v", items, err)
	}
	e.mustRun(errs.NotFound, "--list-client", "--interface", "awg0", "--id", "nobody")
	e.mustRun(errs.NotFound, "--list-client", "--interface", "nope")
}

// Интерфейс опущен: данных о сессиях нет.
func TestListClientDown(t *testing.T) {
	e := newEnv(t)
	setupAWG0(t, e, false)
	e.mustRun(0, "--list-client", "--interface", "awg0")
	golden(t, "list_client_down.golden", e.stdout.String())
	if e.calledPrefix("awg show awg0 dump") {
		t.Fatal("dump of a down interface")
	}
	e.mustRun(0, "--list-client", "--interface", "awg0", "--json")
	var items []clientItem
	json.Unmarshal(e.stdout.Bytes(), &items)
	if len(items) != 2 || items[0].LatestHandshake != nil || items[0].Endpoint != nil || items[0].RxBytes != 0 {
		t.Fatalf("items = %+v", items)
	}
}

// SPEC 5.3: peer'ы без krot:id показываются с id "-".
func TestListClientForeign(t *testing.T) {
	e := newEnv(t)
	e.write(e.wgConf("office"), fixture(t, "foreign_wg.conf"))
	e.mustRun(0, "--list-client", "--interface", "office", "--json")
	var items []clientItem
	if err := json.Unmarshal(e.stdout.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].ID != "-" || items[1].ID != "-" || items[0].Created != nil ||
		!slices.Equal(items[1].Addresses, []string{"10.9.0.3/32", "192.168.50.0/24"}) {
		t.Fatalf("items = %+v", items)
	}
	e.mustRun(0, "--list-client", "--interface", "office")
	if !strings.Contains(e.stdout.String(), "\n-  ") {
		t.Fatalf("table:\n%s", e.stdout)
	}
}

func TestHumanFormats(t *testing.T) {
	bytesCases := map[int64]string{0: "0B", 512: "512B", 1023: "1023B", 1024: "1.0KiB", 1536: "1.5KiB",
		13002342: "12.4MiB", 1181116006: "1.1GiB", 5 << 40: "5.0TiB"}
	for n, want := range bytesCases {
		if got := humanBytes(n); got != want {
			t.Errorf("humanBytes(%d) = %q; want %q", n, got, want)
		}
	}
	now := fixedNow
	agoCases := map[int64]string{0: "-", now.Unix() - 5: "5s ago", now.Unix() - 75: "1m ago",
		now.Unix() - 7200: "2h ago", now.Unix() - 3*86400: "3d ago", now.Unix() + 10: "0s ago"}
	for ts, want := range agoCases {
		if got := humanAgo(ts, now); got != want {
			t.Errorf("humanAgo(%d) = %q; want %q", ts, got, want)
		}
	}
}
