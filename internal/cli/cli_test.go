package cli

import (
	"net/netip"
	"slices"
	"strings"
	"testing"

	"github.com/snmp161/krot/internal/awg"
	"github.com/snmp161/krot/internal/errs"
)

var actions = []Action{
	ActList, ActCreate, ActDelete, ActUp, ActDown,
	ActAddClient, ActDeleteClient, ActLockClient, ActUnlockClient, ActListClient,
}

var params = []string{
	"type", "interface", "ip", "port", "endpoint", "mtu", "dns", "i1",
	"id", "allowed-ips", "psk", "qr", "amnezia-key", "json", "yes",
}

var boolParams = map[string]bool{"psk": true, "qr": true, "amnezia-key": true, "json": true, "yes": true}

// SPEC 4.3: матрица параметров. О — обязателен, н — необязателен, — — недопустим.
var specMatrix = map[string]map[Action]string{
	"type":        {ActList: "н", ActCreate: "О"},
	"interface":   {ActList: "н", ActCreate: "н", ActDelete: "О", ActUp: "О", ActDown: "О", ActAddClient: "О", ActDeleteClient: "О", ActLockClient: "О", ActUnlockClient: "О", ActListClient: "О"},
	"ip":          {ActList: "н", ActCreate: "О", ActAddClient: "н"},
	"port":        {ActList: "н", ActCreate: "О"},
	"endpoint":    {ActCreate: "О", ActAddClient: "н"},
	"mtu":         {ActCreate: "н"},
	"dns":         {ActCreate: "н", ActAddClient: "н"},
	"i1":          {ActCreate: "н"},
	"id":          {ActAddClient: "О", ActDeleteClient: "О", ActLockClient: "О", ActUnlockClient: "О", ActListClient: "н"},
	"allowed-ips": {ActAddClient: "О"},
	"psk":         {ActAddClient: "н"},
	"qr":          {ActAddClient: "н"},
	"amnezia-key": {ActAddClient: "н"},
	"json":        {ActList: "н", ActListClient: "н"},
	"yes":         {ActDelete: "н"},
}

// Валидное значение параметра для действия.
func validValue(p string, a Action) string {
	switch p {
	case "type":
		return "awg2"
	case "interface":
		return "wg0"
	case "ip":
		switch a {
		case ActCreate:
			return "10.8.0.1/24"
		case ActAddClient:
			return "10.8.0.2/32"
		}
		return "10.8.0.0/24"
	case "port":
		return "51820"
	case "endpoint":
		return "vpn.example.com"
	case "mtu":
		return "1376"
	case "dns":
		return "1.1.1.1"
	case "i1":
		return "<b 0xf6ab3267fa><r 16>"
	case "id":
		return "phone"
	case "allowed-ips":
		return "0.0.0.0/0,::/0"
	}
	panic(p)
}

func flagArgs(p string, a Action) []string {
	if boolParams[p] {
		return []string{"--" + p}
	}
	return []string{"--" + p, validValue(p, a)}
}

// Минимальный валидный набор: действие + все обязательные параметры.
func baseArgs(a Action, except string) []string {
	args := []string{"--" + string(a)}
	for _, p := range params {
		if p != except && specMatrix[p][a] == "О" {
			args = append(args, flagArgs(p, a)...)
		}
	}
	return args
}

func code(err error) int { return errs.Code(err) }

// SPEC 4.3: для каждого действия — обязательные, допустимые и недопустимые флаги.
func TestMatrix(t *testing.T) {
	for _, a := range actions {
		if _, err := Parse(baseArgs(a, "")); err != nil {
			t.Fatalf("%s: minimal args %q rejected: %v", a, baseArgs(a, ""), err)
		}
		for _, p := range params {
			switch specMatrix[p][a] {
			case "О":
				if got := code(parseErr(baseArgs(a, p))); got != errs.Usage {
					t.Errorf("--%s without required --%s: code %d; want 2", a, p, got)
				}
			case "н":
				args := append(baseArgs(a, ""), flagArgs(p, a)...)
				if _, err := Parse(args); err != nil {
					t.Errorf("--%s with optional --%s: %v", a, p, err)
				}
			default:
				args := append(baseArgs(a, ""), flagArgs(p, a)...)
				if got := code(parseErr(args)); got != errs.Usage {
					t.Errorf("--%s with forbidden --%s: code %d; want 2", a, p, got)
				}
			}
		}
	}
}

func parseErr(args []string) error {
	_, err := Parse(args)
	return err
}

// SPEC 4.1: ровно одно действие.
func TestActionCount(t *testing.T) {
	if got := code(parseErr(nil)); got != errs.Usage {
		t.Errorf("no args: code %d; want 2", got)
	}
	if got := code(parseErr([]string{"--interface", "wg0"})); got != errs.Usage {
		t.Errorf("no action: code %d; want 2", got)
	}
	for i, a := range actions {
		b := actions[(i+1)%len(actions)]
		args := append(baseArgs(a, ""), "--"+string(b))
		if got := code(parseErr(args)); got != errs.Usage {
			t.Errorf("--%s --%s: code %d; want 2", a, b, got)
		}
	}
}

// SPEC 4.1: только длинная форма, флаг не игнорируется молча.
func TestSyntax(t *testing.T) {
	bad := [][]string{
		{"-list"},
		{"--list", "extra"},
		{"--list", "--list"},
		{"--list=yes"},
		{"--unknown"},
		{"--list", "--unknown-flag"},
		{"--delete", "--interface"},
		{"--delete", "--interface", "--yes"},
		{"--delete-client", "--interface", "wg0", "--id", "--json"},
		{"--add-client", "--interface", "wg0", "--interface", "wg1", "--id", "a", "--allowed-ips", "0.0.0.0/0"},
		{"--", "--list"},
		{"-"},
		{"--"},
		{"--=x"},
		{"--create", "--type", "wg", "--ip", "10.8.0.1/24", "--port", "51820", "--endpoint", "h", "--i1", "<r 1>"},
	}
	for _, args := range bad {
		if got := code(parseErr(args)); got != errs.Usage {
			t.Errorf("%q: code %d; want 2", args, got)
		}
	}
}

func TestEqualsForm(t *testing.T) {
	c, err := Parse([]string{"--create", "--type=awg3", "--ip=10.8.0.1/24,fd00:8::1/64", "--port=51820",
		"--endpoint=vpn.example.com", "--mtu=1280", "--dns=1.1.1.1,9.9.9.9", "--interface=awg7", "--i1=<r 32>"})
	if err != nil {
		t.Fatal(err)
	}
	want := Command{
		Action:    ActCreate,
		Type:      awg.TypeAWG3,
		Interface: "awg7",
		IPs:       []netip.Prefix{netip.MustParsePrefix("10.8.0.1/24"), netip.MustParsePrefix("fd00:8::1/64")},
		Port:      51820,
		Endpoint:  "vpn.example.com",
		MTU:       1280,
		DNS:       []netip.Addr{netip.MustParseAddr("1.1.1.1"), netip.MustParseAddr("9.9.9.9")},
		I1:        "<r 32>",
	}
	if c.Action != want.Action || c.Type != want.Type || c.Interface != want.Interface ||
		!slices.Equal(c.IPs, want.IPs) || c.Port != want.Port || c.Endpoint != want.Endpoint ||
		c.MTU != want.MTU || !slices.Equal(c.DNS, want.DNS) || c.I1 != want.I1 {
		t.Fatalf("got %+v\nwant %+v", *c, want)
	}
	for _, f := range []string{"type", "ip", "port", "endpoint", "mtu", "dns", "interface", "i1"} {
		if !c.Has(f) {
			t.Errorf("Has(%q) = false", f)
		}
	}
	if c.Has("id") {
		t.Error("Has(id) = true")
	}
}

func TestClientFlags(t *testing.T) {
	c, err := Parse([]string{"--add-client", "--interface", "wg0", "--id", "phone-sergei",
		"--allowed-ips", "0.0.0.0/0, ::/0", "--psk", "--qr", "--amnezia-key", "--ip", "10.8.0.9/32,fd00:8::9/128", "--endpoint", "2001:db8::1",
		"--dns", "8.8.8.8,2001:4860:4860::8888"})
	if err != nil {
		t.Fatal(err)
	}
	if c.Action != ActAddClient || c.ID != "phone-sergei" || !c.PSK || !c.QR || !c.AmneziaKey || c.Endpoint != "2001:db8::1" ||
		!slices.Equal(c.DNS, []netip.Addr{netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("2001:4860:4860::8888")}) ||
		!slices.Equal(c.AllowedIPs, []netip.Prefix{netip.MustParsePrefix("0.0.0.0/0"), netip.MustParsePrefix("::/0")}) ||
		!slices.Equal(c.IPs, []netip.Prefix{netip.MustParsePrefix("10.8.0.9/32"), netip.MustParsePrefix("fd00:8::9/128")}) {
		t.Fatalf("got %+v", *c)
	}
	c, err = Parse([]string{"--list", "--ip", "10.8.0.5", "--json", "--type", "wg", "--port", "51820"})
	if err != nil {
		t.Fatal(err)
	}
	if !c.JSON || c.Type != awg.TypeWG || c.Port != 51820 ||
		!slices.Equal(c.IPs, []netip.Prefix{netip.MustParsePrefix("10.8.0.5/32")}) {
		t.Fatalf("got %+v", *c)
	}
}

// SPEC 4.1: --help и --version.
func TestHelpVersion(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"--list", "--help"}, {"--help", "--bogus", "-x"}} {
		c, err := Parse(args)
		if err != nil || !c.Help {
			t.Errorf("%q: %+v, %v", args, c, err)
		}
	}
	c, err := Parse([]string{"--version"})
	if err != nil || !c.Version {
		t.Errorf("--version: %+v, %v", c, err)
	}
	if got := code(parseErr([]string{"--version", "--list"})); got != errs.Usage {
		t.Errorf("--version --list: code %d; want 2", got)
	}
	if !strings.Contains(Usage(), "--add-client") || !strings.Contains(Usage(), "--allowed-ips") {
		t.Error("usage text is incomplete")
	}
	for _, p := range params {
		if !strings.Contains(Usage(), "  --"+p+" ") {
			t.Errorf("usage text does not describe --%s", p)
		}
	}
}

// SPEC 4.4: форматы значений → код 3.
func TestValues(t *testing.T) {
	create := func(kv ...string) []string {
		m := map[string]string{"type": "awg2", "ip": "10.8.0.1/24", "port": "51820", "endpoint": "vpn.example.com"}
		for i := 0; i < len(kv); i += 2 {
			m[kv[i]] = kv[i+1]
		}
		args := []string{"--create"}
		for _, k := range []string{"type", "ip", "port", "endpoint", "mtu", "dns", "interface", "i1"} {
			if v, ok := m[k]; ok {
				args = append(args, "--"+k, v)
			}
		}
		return args
	}
	add := func(kv ...string) []string {
		m := map[string]string{"interface": "wg0", "id": "phone", "allowed-ips": "0.0.0.0/0"}
		for i := 0; i < len(kv); i += 2 {
			m[kv[i]] = kv[i+1]
		}
		args := []string{"--add-client"}
		for _, k := range []string{"interface", "id", "allowed-ips", "ip", "endpoint", "dns"} {
			if v, ok := m[k]; ok {
				args = append(args, "--"+k, v)
			}
		}
		return args
	}
	long := strings.Repeat("a", 64)
	cases := []struct {
		args []string
		want int
	}{
		{create(), 0},
		{create("type", "foo"), 3},
		{create("type", "AWG2"), 3},
		{create("interface", "wg0"), 0},
		{create("interface", "a_b.c=d+e-f"), 0},
		{create("interface", "abcdefghijklmno"), 0},  // 15
		{create("interface", "abcdefghijklmnop"), 3}, // 16
		{create("interface", "wg 0"), 3},
		{create("interface", "wg/0"), 3},
		{create("interface", ""), 3},
		{create("interface", "."), 3},
		{create("interface", ".."), 3},
		{create("port", "1"), 0},
		{create("port", "65535"), 0},
		{create("port", "0"), 3},
		{create("port", "65536"), 3},
		{create("port", "abc"), 3},
		{create("port", "-1"), 3},
		{create("port", "+5"), 3},
		{create("port", ""), 3},
		{create("mtu", "576"), 0},
		{create("mtu", "9000"), 0},
		{create("mtu", "575"), 3},
		{create("mtu", "9001"), 3},
		{create("mtu", "1e3"), 3},
		{create("ip", "10.8.0.1/24,fd00:8::1/64"), 0},
		{create("ip", "fd00:8::1/64"), 0},
		{create("ip", "10.8.0.1"), 3},
		{create("ip", "10.8.0.0/24"), 3},             // адрес сети
		{create("ip", "10.8.0.255/24"), 3},           // broadcast
		{create("ip", "fd00:8::/64"), 3},             // адрес сети IPv6
		{create("ip", "10.8.0.1/24,10.9.0.1/24"), 3}, // два IPv4
		{create("ip", "fd00::1/64,fd01::1/64"), 3},   // два IPv6
		{create("ip", "10.8.0.1/24,"), 3},            // пустой элемент
		{create("ip", "10.8.0.1/33"), 3},
		{create("ip", "::ffff:10.8.0.1/120"), 3},
		{create("endpoint", "1.2.3.4"), 0},
		{create("endpoint", "2001:db8::1"), 0},
		{create("endpoint", "vpn-1.example.com"), 0},
		{create("endpoint", "localhost"), 0},
		{create("endpoint", "bad_host"), 3},
		{create("endpoint", "host:51820"), 3},
		{create("endpoint", "-bad.com"), 3},
		{create("endpoint", "bad-.com"), 3},
		{create("endpoint", "1.2.3.256"), 3},
		{create("endpoint", long+".com"), 3},
		{create("endpoint", "[2001:db8::1]"), 3},
		{create("endpoint", ""), 3},
		{create("dns", "1.1.1.1,9.9.9.9"), 0},
		{create("dns", "1.1.1.1, 2606:4700:4700::1111"), 0},
		{create("dns", "1.1.1"), 3},
		{create("dns", ""), 3},
		{create("dns", "1.1.1.1/32"), 3},
		{create("i1", ""), 3},
		{add(), 0},
		{add("id", "phone-sergei_2.0"), 0},
		{add("id", long), 0},
		{add("id", long+"a"), 3},
		{add("id", "a b"), 3},
		{add("id", "a/b"), 3},
		{add("id", ""), 3},
		{add("ip", "10.8.0.2/32"), 0},
		{add("ip", "10.8.0.2/32,fd00:8::2/128"), 0},
		{add("ip", "10.8.0.2/24"), 3},
		{add("ip", "fd00:8::2/64"), 3},
		{add("ip", "10.8.0.2"), 3},
		{add("ip", "10.8.0.2/32,10.8.0.3/32"), 3},
		{add("allowed-ips", "0.0.0.0/0,::/0"), 0},
		{add("allowed-ips", "10.8.0.0/24"), 0},
		{add("allowed-ips", "0.0.0.0"), 3},
		{add("allowed-ips", ""), 3},
		{add("endpoint", "bad_host"), 3},
		{add("dns", "8.8.8.8, 2001:4860:4860::8888"), 0},
		{add("dns", "8.8.8"), 3},
		{add("dns", ""), 3},
		{add("dns", "8.8.8.8/32"), 3},
		{[]string{"--list", "--ip", "10.8.0.0/24"}, 0},
		{[]string{"--list", "--ip", "10.8.0.5"}, 0},
		{[]string{"--list", "--ip", "fd00::5,10.8.0.5"}, 0},
		{[]string{"--list", "--ip", "nope"}, 3},
		{[]string{"--list", "--port", "0"}, 3},
		{[]string{"--list", "--type", "x"}, 3},
		{[]string{"--list", "--interface", "bad/name"}, 3},
		{[]string{"--list-client", "--interface", "wg0", "--id", "bad id"}, 3},
	}
	for _, c := range cases {
		if got := code(parseErr(c.args)); got != c.want {
			t.Errorf("%q: code %d; want %d (%v)", c.args, got, c.want, parseErr(c.args))
		}
	}
}

// Ошибки использования (2) проверяются раньше ошибок значений (3).
func TestUsageBeforeValues(t *testing.T) {
	args := []string{"--delete", "--interface", "bad/name", "--json"}
	if got := code(parseErr(args)); got != errs.Usage {
		t.Errorf("code %d; want 2", got)
	}
}

// SPEC 4.4, 7.1 V-05: --i1 проверяется по грамматике CPS → код 3.
func TestI1Syntax(t *testing.T) {
	base := []string{"--create", "--type", "awg", "--ip", "10.8.0.1/24", "--port", "51820", "--endpoint", "h.example"}
	for _, v := range []string{"<b 0xf6ab3267fa><r 16>", "<t><rc 10><rd 5>"} {
		if _, err := Parse(append(base, "--i1", v)); err != nil {
			t.Errorf("--i1 %q: %v", v, err)
		}
	}
	for _, v := range []string{"<x>", "<b 0xabc>", "<r 1001>", "text", "<b 0x00"} {
		if got := code(parseErr(append(base, "--i1", v))); got != errs.Invalid {
			t.Errorf("--i1 %q: code %d; want 3", v, got)
		}
	}
}
