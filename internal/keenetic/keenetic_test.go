package keenetic

import (
	"net/netip"
	"strings"
	"testing"
)

func testParams() Params {
	return Params{
		Name:       "krot wg0: phone",
		IfName:     "Wireguard0",
		Iface:      "wg0",
		PrivateKey: "cFppM0F4V2s5QzF6RjJ4TDh5NlcxYjBzTWpjNmdhaVE9",
		Address:    netip.MustParseAddr("10.8.0.2"),
		MTU:        1420,
		PublicKey:  "sG9yU2VydmVyUHVibGljS2V5MTIzNDU2Nzg5MEFCQ0RFRg==",
		Endpoint:   "vpn.example.com:51820",
		PSK:        "cGstc2VjcmV0LWtleQ==",
		AllowedIPs: []netip.Prefix{netip.MustParsePrefix("0.0.0.0/0")},
		Keepalive:  25,
	}
}

// SPEC 6.13: полный блок команд по синтаксису running-config KeeneticOS.
func TestCommandsGolden(t *testing.T) {
	got, err := Commands(testParams())
	if err != nil {
		t.Fatal(err)
	}
	want := `interface Wireguard0
 description "krot wg0: phone"
 ip address 10.8.0.2 255.255.255.255
 wireguard private-key cFppM0F4V2s5QzF6RjJ4TDh5NlcxYjBzTWpjNmdhaVE9
 wireguard peer sG9yU2VydmVyUHVibGljS2V5MTIzNDU2Nzg5MEFCQ0RFRg== !krot wg0
 endpoint vpn.example.com:51820
 preshared-key cGstc2VjcmV0LWtleQ==
 keepalive-interval 25
 allow-ips 0.0.0.0 0.0.0.0
 !
 up`
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

// SPEC 6.13: без PSK строки preshared-key нет.
func TestCommandsNoPSK(t *testing.T) {
	p := testParams()
	p.PSK = ""
	got, err := Commands(p)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "preshared-key") {
		t.Fatalf("no PSK expected:\n%s", got)
	}
	if !strings.Contains(got, "endpoint vpn.example.com:51820") {
		t.Fatalf("endpoint missing:\n%s", got)
	}
}

// SPEC 6.13: allow-ips построчно, маска точками, порядок сохраняется.
func TestCommandsAllowIPs(t *testing.T) {
	p := testParams()
	p.PSK = ""
	p.AllowedIPs = []netip.Prefix{
		netip.MustParsePrefix("10.8.0.0/24"),
		netip.MustParsePrefix("192.168.0.0/16"),
		netip.MustParsePrefix("172.16.5.5/32"),
	}
	got, err := Commands(p)
	if err != nil {
		t.Fatal(err)
	}
	lines := []string{
		"allow-ips 10.8.0.0 255.255.255.0",
		"allow-ips 192.168.0.0 255.255.0.0",
		"allow-ips 172.16.5.5 255.255.255.255",
	}
	at := 0
	for _, line := range strings.Split(got, "\n") {
		if line == lines[at] {
			at++
			if at == len(lines) {
				break
			}
		}
	}
	if at != len(lines) {
		t.Fatalf("allow-ips lines missing or out of order:\n%s", got)
	}
}

// SPEC 6.13: MTU не по умолчанию (1420) — комментарий перед блоком.
func TestCommandsMTUNote(t *testing.T) {
	p := testParams()
	p.MTU = 1380
	got, err := Commands(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "! NB: ") || !strings.Contains(got, "1380") {
		t.Fatalf("MTU note expected:\n%s", got)
	}
	if !strings.Contains(got, "interface Wireguard0") {
		t.Fatalf("block missing:\n%s", got)
	}
	for _, mtu := range []int{1420, 0} {
		p.MTU = mtu
		got, err := Commands(p)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(got, "NB:") {
			t.Fatalf("MTU %d must not trigger a note:\n%s", mtu, got)
		}
	}
}

// SPEC 6.13: только IPv4 — IPv6 в адресе или AllowedIPs отклоняется.
func TestCommandsIPv6Rejected(t *testing.T) {
	p := testParams()
	p.Address = netip.MustParseAddr("fd00::2")
	if _, err := Commands(p); err == nil {
		t.Fatal("IPv6 address: want error")
	}
	p = testParams()
	p.AllowedIPs = []netip.Prefix{netip.MustParsePrefix("::/0")}
	if _, err := Commands(p); err == nil {
		t.Fatal("IPv6 allowed-ips: want error")
	}
}

// SPEC 6.13: имя интерфейса и описание попадают в блок.
func TestCommandsNames(t *testing.T) {
	p := testParams()
	p.IfName = "Wireguard7"
	p.Name = "krot wg1: laptop"
	p.Iface = "wg1"
	got, err := Commands(p)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"interface Wireguard7", `description "krot wg1: laptop"`, "!krot wg1"} {
		if !strings.Contains(got, want) {
			t.Fatalf("%q missing:\n%s", want, got)
		}
	}
}
