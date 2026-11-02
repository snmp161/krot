package app

import (
	"errors"
	"fmt"
	"net/netip"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/snmp161/krot/internal/amnezia"
	"github.com/snmp161/krot/internal/awg"
	"github.com/snmp161/krot/internal/cli"
	"github.com/snmp161/krot/internal/conf"
	"github.com/snmp161/krot/internal/errs"
	"github.com/snmp161/krot/internal/ipalloc"
	"github.com/snmp161/krot/internal/keenetic"
	"github.com/snmp161/krot/internal/keys"
	"github.com/snmp161/krot/internal/qr"
)

// keepalive — PersistentKeepalive клиентского конфига (SPEC 6.5).
const keepalive = 25

// iKeys — I-параметры AmneziaWG: при сгенерированной цепочке мимикрии копирование
// значений сервера не нужно (SPEC 6.12).
var iKeys = []string{"I1", "I2", "I3", "I4", "I5"}

// SPEC 6.5: --add-client.
func (a *App) addClient(cmd *cli.Command) error {
	var clientText string
	var kgBlock string
	err := a.withLock(func() error {
		t, err := a.findTunnel(cmd.Interface)
		if err != nil {
			return err
		}
		if err := a.need(t.tool()); err != nil {
			return err
		}
		ifc, _ := t.conf.Interface()
		if _, ok := t.conf.PeerByID(cmd.ID); ok {
			return errs.Conflictf("client %s already exists on interface %s", cmd.ID, t.name)
		}

		// Всё, что нужно клиентскому конфигу, проверяется до изменений.
		endpoint, err := clientEndpoint(t, ifc, cmd.Endpoint)
		if err != nil {
			return err
		}
		serverPriv, err := keys.ParseKey(ifc.Get("PrivateKey"))
		if err != nil {
			return fmt.Errorf("interface %s: invalid PrivateKey in %s", t.name, t.path)
		}
		serverPub, err := serverPriv.Public()
		if err != nil {
			return err
		}
		addrs, err := clientAddresses(t, ifc, cmd.IPs)
		if err != nil {
			return err
		}

		priv, err := keys.NewPrivate(a.Rand)
		if err != nil {
			return err
		}
		pub, err := priv.Public()
		if err != nil {
			return err
		}
		var psk string
		if cmd.PSK {
			k, err := keys.NewPreshared(a.Rand)
			if err != nil {
				return err
			}
			psk = k.String()
		}

		// Цепочка мимикрии I1–Ik: --mimic, иначе профиль интерфейса (SPEC 6.12).
		// Генерируется до изменений: серверу I-параметры не нужны, цепочка уходит
		// только в клиентский конфиг.
		profile := cmd.Mimic
		if profile == "" {
			profile = ifc.Meta["mimic"]
		}
		var chain []string
		if profile != "" {
			chain, err = awg.GenerateMimicChain(profile, a.Rand, a.Now())
			if err != nil {
				return errs.Invalidf("interface %s: %v", t.name, err)
			}
			// Сгенерированное обязано проходить валидацию (SPEC 7.6).
			total := 0
			for _, cps := range chain {
				if err := awg.ValidateCPS(cps); err != nil {
					return errs.Wrap(errs.Invalid, err)
				}
				total += len(cps)
			}
			if total > awg.MaxITotal {
				return errs.Invalidf("mimic chain is %d characters, at most %d (SPEC 7.1 V-06)", total, awg.MaxITotal)
			}
		}

		// Блок команд CLI Keenetic: всё проверяется до изменений (SPEC 6.13).
		if cmd.Keenetic {
			var asc *keenetic.ASC
			if t.amnezia {
				// KeeneticOS поддерживает ASC AmneziaWG 1.0/1.5/2.0 (4.2+/5.1+),
				// но не защиту заголовка 3.1 (SPEC 6.13).
				if ifc.Get("HeaderProtectionKey") != "" || ifc.Get("RandomTrailers") != "" || ifc.Get("DisableCookies") != "" {
					return errs.Invalidf("--keenetic: KeeneticOS does not support AmneziaWG 3.1 (header protection) of interface %s", t.name)
				}
				asc = &keenetic.ASC{
					Jc: ifc.Get("Jc"), Jmin: ifc.Get("Jmin"), Jmax: ifc.Get("Jmax"),
					S1: ifc.Get("S1"), S2: ifc.Get("S2"), S3: ifc.Get("S3"), S4: ifc.Get("S4"),
					H1: ifc.Get("H1"), H2: ifc.Get("H2"), H3: ifc.Get("H3"), H4: ifc.Get("H4"),
					I1: ifc.Get("I1"), I2: ifc.Get("I2"), I3: ifc.Get("I3"), I4: ifc.Get("I4"), I5: ifc.Get("I5"),
				}
			}
			var v4 netip.Addr
			for _, p := range addrs {
				if p.Addr().Is4() {
					v4 = p.Addr()
					break
				}
			}
			if !v4.IsValid() {
				return errs.Invalidf("--keenetic requires an IPv4 client address: %s has none", t.name)
			}
			var allowed4 []netip.Prefix
			for _, p := range cmd.AllowedIPs {
				if !p.Addr().Is4() {
					return errs.Invalidf("--keenetic supports IPv4 only: %s is IPv6", p)
				}
				allowed4 = append(allowed4, p)
			}
			if strings.HasPrefix(endpoint, "[") {
				return errs.Invalidf("--keenetic supports IPv4 endpoints only: %s is IPv6", endpoint)
			}
			note := ""
			if chain != nil {
				// Цепочка мимикрии (6.12) живёт только в клиентском конфиге —
				// в asc блоке I1–I5 будут 0.
				note = "mimicry chain I1-I5 is only in the client config above; import it via the Keenetic web UI or set the I-values manually"
			}
			mtu, _ := strconv.Atoi(ifc.Get("MTU"))
			block, kerr := keenetic.Commands(keenetic.Params{
				Name:       "krot " + t.name + ": " + cmd.ID,
				IfName:     "Wireguard0",
				Iface:      t.name,
				PrivateKey: priv.String(),
				Address:    v4,
				MTU:        mtu,
				PublicKey:  serverPub.String(),
				Endpoint:   endpoint,
				PSK:        psk,
				AllowedIPs: allowed4,
				Keepalive:  keepalive,
				ASC:        asc,
				Note:       note,
			})
			if kerr != nil {
				return errs.Invalidf("invalid --keenetic: %v", kerr)
			}
			kgBlock = block
		}

		// Запись peer'а в конфиг.
		orig := t.conf.Bytes()
		addrStrs := prefixStrings(addrs)
		t.conf.AddPeer(conf.NewPeer{
			ID:           cmd.ID,
			Created:      a.Now().UTC().Format("2006-01-02T15:04:05Z"),
			PublicKey:    pub.String(),
			PresharedKey: psk,
			AllowedIPs:   addrStrs,
		})
		if err := a.save(t); err != nil {
			return err
		}
		// Живое применение; при ошибке — откат конфига.
		if err := a.applyPeer(t, pub.String(), addrStrs, psk); err != nil {
			return a.rollback(t, orig, err)
		}

		clientText = renderClient(ifc, priv.String(), addrStrs, clientDNS(ifc, cmd.DNS), serverPub.String(), psk, endpoint, cmd.AllowedIPs, chain)
		return nil
	})
	if err != nil {
		return err
	}

	// Конфиг, ключ и QR показываются один раз; приватный ключ клиента нигде не хранится (SPEC 5.2).
	a.printf("%s", clientText)
	if cmd.AmneziaKey {
		a.printf("\n%s\n", amnezia.Key(clientText))
	}
	if cmd.QR {
		code, err := qr.Render(clientText)
		if err != nil {
			return errs.Invalidf("config too large for QR (client %s was added, use the text config above)", cmd.ID)
		}
		a.printf("\n%s", code)
	}
	if kgBlock != "" {
		a.printf("\n%s\n", kgBlock)
	}
	return nil
}

// applyPeer добавляет peer в поднятый интерфейс точечно: `wg|awg set <if> peer ...` (SPEC 6.5, 6.10).
// PSK передаётся через временный файл 0600, удаляемый сразу после вызова.
func (a *App) applyPeer(t *tunnel, pub string, allowed []string, psk string) error {
	up, err := a.isUp(t)
	if err != nil || !up {
		return err
	}
	args := []string{"set", t.name, "peer", pub, "allowed-ips", strings.Join(allowed, ",")}
	if psk != "" {
		f, err := os.CreateTemp("", "krot-psk-*")
		if err != nil {
			return err
		}
		defer os.Remove(f.Name())
		_, werr := f.WriteString(psk + "\n")
		if cerr := f.Close(); werr == nil {
			werr = cerr
		}
		if werr != nil {
			return werr
		}
		args = append(args, "preshared-key", f.Name())
	}
	_, err = a.Exec.Run(t.tool(), args...)
	return err
}

// rollback возвращает конфиг к исходному состоянию после неудачной живой операции (код 1).
func (a *App) rollback(t *tunnel, orig []byte, cause error) error {
	t.conf = conf.Parse(orig)
	if err := a.save(t); err != nil {
		return fmt.Errorf("%v; rollback of %s also failed: %v", cause, t.path, err)
	}
	return fmt.Errorf("failed to apply the change to the running interface %s, config rolled back: %w", t.name, cause)
}

// clientEndpoint — host:port для клиентского конфига: --endpoint или # krot:endpoint (SPEC 6.5).
func clientEndpoint(t *tunnel, ifc conf.Interface, override string) (string, error) {
	host := override
	if host == "" {
		host = ifc.Meta["endpoint"]
	}
	if host == "" {
		return "", errs.Invalidf("interface %s has no endpoint: pass --endpoint or add the line '# krot:endpoint=<host>' before [Interface] in %s", t.name, t.path)
	}
	port := ifc.Get("ListenPort")
	if p, err := strconv.Atoi(port); err != nil || p < 1 || p > 65535 {
		return "", errs.Invalidf("interface %s has no valid ListenPort in %s", t.name, t.path)
	}
	if a, err := netip.ParseAddr(host); err == nil && a.Is6() {
		host = "[" + host + "]"
	}
	return host + ":" + port, nil
}

// clientDNS — DNS клиентского конфига: --dns, иначе # krot:dns, иначе все строки DNS из [Interface]
// сервера; "" — DNS нет (SPEC 6.5).
func clientDNS(ifc conf.Interface, override []netip.Addr) string {
	if len(override) > 0 {
		return joinAddrs(override, ", ")
	}
	if dns := ifc.Meta["dns"]; dns != "" {
		var out []string
		for _, s := range strings.Split(dns, ",") {
			if s = strings.TrimSpace(s); s != "" {
				out = append(out, s)
			}
		}
		return strings.Join(out, ", ")
	}
	return strings.Join(ifc.List("DNS"), ", ")
}

// serverPrefixes — адреса интерфейса (Address).
func serverPrefixes(ifc conf.Interface) []netip.Prefix {
	var out []netip.Prefix
	for _, s := range ifc.List("Address") {
		if p, err := netip.ParsePrefix(s); err == nil {
			out = append(out, p)
		}
	}
	return out
}

// usedPrefixes — занятые адреса: адреса сервера и AllowedIPs всех peer'ов (включая заблокированных и чужих).
func usedPrefixes(c *conf.Config, servers []netip.Prefix) []netip.Prefix {
	var out []netip.Prefix
	for _, s := range servers {
		out = append(out, netip.PrefixFrom(s.Addr(), s.Addr().BitLen()))
	}
	for _, p := range c.Peers() {
		for _, s := range p.AllowedIPs() {
			if pr, err := netip.ParsePrefix(s); err == nil {
				out = append(out, pr)
			} else if a, err := netip.ParseAddr(s); err == nil {
				out = append(out, netip.PrefixFrom(a, a.BitLen()))
			}
		}
	}
	return out
}

// clientAddresses — адреса клиента: заданные вручную (проверка, код 5) или первые свободные (SPEC 6.5 п.2).
func clientAddresses(t *tunnel, ifc conf.Interface, manual []netip.Prefix) ([]netip.Prefix, error) {
	servers := serverPrefixes(ifc)
	if len(servers) == 0 {
		return nil, errs.Invalidf("interface %s has no Address in %s", t.name, t.path)
	}
	used := usedPrefixes(t.conf, servers)
	if len(manual) > 0 {
		for _, p := range manual {
			if err := ipalloc.Check(p.Addr(), servers, used); err != nil {
				return nil, errs.Conflictf("address %s cannot be used on %s: %v", p, t.name, err)
			}
		}
		return manual, nil
	}
	var out []netip.Prefix
	for _, is4 := range []bool{true, false} {
		for _, s := range servers {
			if s.Addr().Is4() != is4 {
				continue
			}
			addr, err := ipalloc.Allocate(s, used)
			if errors.Is(err, ipalloc.ErrExhausted) {
				return nil, errs.Conflictf("no free addresses left in %s on interface %s", s.Masked(), t.name)
			}
			if err != nil {
				return nil, err
			}
			out = append(out, netip.PrefixFrom(addr, addr.BitLen()))
			break // одна сеть на семейство
		}
	}
	return out, nil
}

// renderClient формирует клиентский конфиг (SPEC 6.5, 7.5).
// mimic — сгенерированная цепочка I1–Ik (SPEC 6.12): не nil — заменяет
// I-параметры, скопированные с сервера; nil — копирование по 7.5.
func renderClient(ifc conf.Interface, priv string, addrs []string, dns, serverPub, psk, endpoint string, allowed []netip.Prefix, mimic []string) string {
	var b strings.Builder
	b.WriteString("[Interface]\n")
	b.WriteString("PrivateKey = " + priv + "\n")
	b.WriteString("Address = " + strings.Join(addrs, ", ") + "\n")
	if dns != "" {
		b.WriteString("DNS = " + dns + "\n")
	}
	if mtu := ifc.Get("MTU"); mtu != "" {
		b.WriteString("MTU = " + mtu + "\n")
	}
	// Все AWG-параметры сервера — без изменений, в каноническом порядке;
	// I-параметры пропускаются, если есть сгенерированная цепочка.
	for _, k := range awg.Keys {
		if mimic != nil && slices.Contains(iKeys, k) {
			continue
		}
		for _, kv := range ifc.Keys {
			if strings.EqualFold(kv.Key, k) {
				b.WriteString(k + " = " + kv.Value + "\n")
				break
			}
		}
	}
	for i, cps := range mimic {
		b.WriteString(fmt.Sprintf("I%d = %s\n", i+1, cps))
	}
	b.WriteString("\n[Peer]\n")
	b.WriteString("PublicKey = " + serverPub + "\n")
	if psk != "" {
		b.WriteString("PresharedKey = " + psk + "\n")
	}
	b.WriteString("Endpoint = " + endpoint + "\n")
	b.WriteString("AllowedIPs = " + strings.Join(prefixStrings(allowed), ", ") + "\n")
	b.WriteString("PersistentKeepalive = " + strconv.Itoa(keepalive) + "\n")
	return b.String()
}
