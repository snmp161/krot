// Package keenetic собирает блок команд CLI KeeneticOS для подключения роутера
// Keenetic клиентом туннеля WireGuard/AmneziaWG (SPEC 6.13).
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
	Keepalive  int    // keepalive-interval
	ASC        *ASC   // параметры обфускации (только awg*); nil — чистый WireGuard
	Note       string // заметка вызывающего кода — комментарием перед блоком
}

// ASC — параметры Advanced Security Configuration KeeneticOS (SPEC 6.13):
// значения verbatim из конфига интерфейса (H1–H4 AWG 2.0 — диапазоны "lo-hi",
// I1–I5 — CPS-строки с пробелами). Пустые поля при выводе становятся "0".
type ASC struct {
	Jc, Jmin, Jmax     string
	S1, S2, S3, S4     string
	H1, H2, H3, H4     string
	I1, I2, I3, I4, I5 string
}

// has20 сообщает, есть ли параметры AWG 1.5/2.0 (расширенная форма asc).
func (a *ASC) has20() bool {
	return a.S3 != "" || a.S4 != "" ||
		a.I1 != "" || a.I2 != "" || a.I3 != "" || a.I4 != "" || a.I5 != ""
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
	if p.ASC != nil && p.ASC.has20() {
		b.WriteString("! NB: AWG 1.5/2.0 parameters require KeeneticOS 5.1+\n")
		if p.ASC.I1 != "" || p.ASC.I2 != "" || p.ASC.I3 != "" || p.ASC.I4 != "" || p.ASC.I5 != "" {
			b.WriteString("! NB: I-values contain spaces (quoted); if the firmware rejects them, import the client config above via the web UI — KeeneticOS 5.1+ parses ASC from the file\n")
		}
	}
	if p.Note != "" {
		for _, line := range strings.Split(p.Note, "\n") {
			b.WriteString("! NB: " + line + "\n")
		}
	}
	b.WriteString("interface " + p.IfName + "\n")
	fmt.Fprintf(&b, " description %q\n", p.Name)
	fmt.Fprintf(&b, " ip address %s %s\n", p.Address, dottedMask(32))
	b.WriteString(" wireguard private-key " + p.PrivateKey + "\n")
	if p.ASC != nil {
		b.WriteString(" wireguard asc " + p.ASC.Jc + " " + p.ASC.Jmin + " " + p.ASC.Jmax +
			" " + p.ASC.S1 + " " + p.ASC.S2 +
			" " + p.ASC.H1 + " " + p.ASC.H2 + " " + p.ASC.H3 + " " + p.ASC.H4)
		if p.ASC.has20() {
			fmt.Fprintf(&b, " %s %s %s %s %s %s %s",
				zero(p.ASC.S3), zero(p.ASC.S4),
				quote(zero(p.ASC.I1)), quote(zero(p.ASC.I2)), quote(zero(p.ASC.I3)),
				quote(zero(p.ASC.I4)), quote(zero(p.ASC.I5)))
		}
		b.WriteString("\n")
	}
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

// zero подменяет пустое значение на "0" (пропуски в блоке AWG 2.0, SPEC 6.13).
func zero(s string) string {
	if s == "" {
		return "0"
	}
	return s
}

// quote оборачивает значение в кавычки (CPS-строки содержат пробелы).
func quote(s string) string {
	if s == "0" {
		return s
	}
	return `"` + s + `"`
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
