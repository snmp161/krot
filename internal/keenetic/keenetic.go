// Package keenetic собирает блок команд CLI KeeneticOS для подключения роутера
// Keenetic клиентом туннеля WireGuard (SPEC 6.13).
package keenetic

import (
	"fmt"
	"net/netip"
	"strings"
)

// DefaultMTU — MTU WireGuard по умолчанию на KeeneticOS (SPEC 6.13).
const DefaultMTU = 1420

// Params — данные клиентского конфига для сборки блока команд.
type Params struct {
	Name       string // описание подключения: "krot <инфтерфейс>: <id>"
	IfName     string // имя интерфейса на Keenetic (например "Wireguard0")
	Iface      string // имя интерфейса krot — для комментария peer'а
	PrivateKey string // приватный ключ клиента
	Address    netip.Addr
	MTU        int // MTU туннеля; 0 или 1420 — без примечания
	PublicKey  string
	Endpoint   string // host:port
	PSK        string // "" — без preshared-key
	AllowedIPs []netip.Prefix
	Keepalive  int // keepalive-interval
}

// Commands собирает блок команд CLI KeeneticOS по синтаксису running-config
// (SPEC 6.13). Только IPv4: IPv6 в адресе или AllowedIPs — ошибка.
func Commands(p Params) (string, error) {
	if !p.Address.Is4() {
		return "", fmt.Errorf("client address %s is not an IPv4 address", p.Address)
	}
	var b strings.Builder
	if p.MTU != 0 && p.MTU != DefaultMTU {
		fmt.Fprintf(&b, "! NB: server MTU is %d; set it manually in the connection settings (Keenetic web UI)\n", p.MTU)
	}
	b.WriteString("interface " + p.IfName + "\n")
	fmt.Fprintf(&b, " description %q\n", p.Name)
	fmt.Fprintf(&b, " ip address %s %s\n", p.Address, dottedMask(32))
	b.WriteString(" wireguard private-key " + p.PrivateKey + "\n")
	b.WriteString(" wireguard peer " + p.PublicKey + " !krot " + p.Iface + "\n")
	b.WriteString(" endpoint " + p.Endpoint + "\n")
	if p.PSK != "" {
		b.WriteString(" preshared-key " + p.PSK + "\n")
	}
	fmt.Fprintf(&b, " keepalive-interval %d\n", p.Keepalive)
	for _, cidr := range p.AllowedIPs {
		if !cidr.Addr().Is4() {
			return "", fmt.Errorf("allowed-ips %s is not an IPv4 prefix", cidr)
		}
		fmt.Fprintf(&b, " allow-ips %s %s\n", cidr.Masked().Addr(), dottedMask(cidr.Bits()))
	}
	b.WriteString(" !\n")
	b.WriteString(" up")
	return b.String(), nil
}

// dottedMask — маска из длины префикса точками ("0.0.0.0", "255.255.255.0").
func dottedMask(bits int) string {
	var m [4]uint8
	for i := 0; i < 4; i++ {
		if b := bits - i*8; b >= 8 {
			m[i] = 0xFF
		} else if b > 0 {
			m[i] = 0xFF << (8 - b)
		}
	}
	return fmt.Sprintf("%d.%d.%d.%d", m[0], m[1], m[2], m[3])
}
