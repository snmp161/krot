package app

import (
	"strings"
	"testing"

	"github.com/snmp161/krot/internal/conf"
)

// SPEC 6.13: --add-client --keenetic печатает блок команд CLI KeeneticOS после конфига.
func TestAddClientKeenetic(t *testing.T) {
	e := newEnv(t)
	e.mustRun(0, createArgs("wg")...)
	e.mustRun(0, "--add-client", "--interface", "wg0", "--id", "a", "--allowed-ips", "0.0.0.0/0", "--psk", "--keenetic")
	out := e.stdout.String()

	for _, want := range []string{
		"interface Wireguard0",
		`description "krot wg0: a"`,
		"ip address 10.8.0.2 255.255.255.255",
		"wireguard private-key ",
		"wireguard peer ",
		"!krot wg0",
		"endpoint vpn.example.com:51820",
		"preshared-key ",
		"keepalive-interval 25",
		"allow-ips 0.0.0.0 0.0.0.0",
		"\n up",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("keenetic output lacks %q:\n%s", want, out)
		}
	}
	// Конфиг идёт первым, блок Keenetic после (SPEC 6.5 п.7).
	if strings.Index(out, "interface Wireguard0") < strings.Index(out, "[Peer]") {
		t.Fatalf("client config must precede the keenetic block:\n%s", out)
	}
	// MTU по умолчанию 1420 — без примечания.
	if strings.Contains(out, "NB:") {
		t.Fatalf("MTU 1420 must not trigger a note:\n%s", out)
	}
	// Приватный ключ из блока совпадает с конфигом.
	if !samePrivateKey(out) {
		t.Fatalf("keenetic private key differs from the client config:\n%s", out)
	}
}

// samePrivateKey сравнивает PrivateKey конфига и wireguard private-key блока.
func samePrivateKey(out string) bool {
	cfgKey := betweenStr(out, "PrivateKey = ", "\n")
	kgKey := betweenStr(out, "wireguard private-key ", "\n")
	return cfgKey != "" && cfgKey == kgKey
}

func betweenStr(s, a, b string) string {
	i := strings.Index(s, a)
	if i < 0 {
		return ""
	}
	s = s[i+len(a):]
	j := strings.Index(s, b)
	if j < 0 {
		return ""
	}
	return s[:j]
}

// SPEC 6.13: awg (1.0) — блок с девятипараметровым wireguard asc.
func TestAddClientKeeneticAWG(t *testing.T) {
	e := newEnv(t)
	e.mustRun(0, createArgs("awg")...)
	e.mustRun(0, "--add-client", "--interface", "awg0", "--id", "a", "--allowed-ips", "0.0.0.0/0", "--keenetic")
	out := e.stdout.String()
	if !strings.Contains(out, "wireguard asc ") {
		t.Fatalf("asc line missing:\n%s", out)
	}
	asc := betweenStr(out, "wireguard asc ", "\n")
	if n := len(strings.Fields(asc)); n != 9 {
		t.Fatalf("AWG 1.0 asc: %d params; want 9 (%q)", n, asc)
	}
	if strings.Contains(out, "KeeneticOS 5.1") {
		t.Fatalf("AWG 1.0 must not require KeeneticOS 5.1:\n%s", out)
	}
	// Значения совпадают с параметрами интерфейса.
	c := conf.Parse([]byte(e.read(e.awgConf("awg0"))))
	ifc, _ := c.Interface()
	for _, k := range []string{"Jc", "Jmin", "Jmax", "S1", "S2", "H1", "H2", "H3", "H4"} {
		if !strings.Contains(asc, ifc.Get(k)) {
			t.Fatalf("asc lacks %s=%q:\n%s", k, ifc.Get(k), asc)
		}
	}
}

// SPEC 6.13: awg2 — расширенная форма asc (S3/S4, 16 значений) и требование 5.1+.
func TestAddClientKeeneticAWG2(t *testing.T) {
	e := newEnv(t)
	e.mustRun(0, createArgs("awg2")...)
	e.mustRun(0, "--add-client", "--interface", "awg0", "--id", "a", "--allowed-ips", "0.0.0.0/0", "--keenetic")
	out := e.stdout.String()
	asc := betweenStr(out, "wireguard asc ", "\n")
	if n := len(strings.Fields(asc)); n != 16 {
		t.Fatalf("AWG 2.0 asc: %d params; want 16 (%q)", n, asc)
	}
	if !strings.Contains(out, "KeeneticOS 5.1+") {
		t.Fatalf("AWG 2.0 must note KeeneticOS 5.1+:\n%s", out)
	}
}

// SPEC 6.13: awg3 (3.1) — KeeneticOS защиту заголовка не поддерживает: код 3.
func TestAddClientKeeneticAWG3(t *testing.T) {
	e := newEnv(t)
	e.mustRun(0, createArgs("awg3")...)
	if got := e.run("--add-client", "--interface", "awg0", "--id", "a", "--allowed-ips", "0.0.0.0/0", "--keenetic"); got != 3 {
		t.Fatalf("awg3 --keenetic: exit %d; want 3", got)
	}
	if strings.Contains(e.read(e.awgConf("awg0")), "[Peer]") {
		t.Fatal("failed --keenetic must not change the config")
	}
}

// SPEC 6.13: только IPv4 — IPv6 в --allowed-ip's отклоняется до изменений.
func TestAddClientKeeneticIPv6(t *testing.T) {
	e := newEnv(t)
	e.mustRun(0, createArgs("wg")...)
	if got := e.run("--add-client", "--interface", "wg0", "--id", "a", "--allowed-ips", "0.0.0.0/0,::/0", "--keenetic"); got != 3 {
		t.Fatalf("IPv6 allowed-ips --keenetic: exit %d; want 3", got)
	}
	if strings.Contains(e.read(e.wgConf("wg0")), "[Peer]") {
		t.Fatal("failed --keenetic must not change the config")
	}
}
