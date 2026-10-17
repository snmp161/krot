package app

import (
	"errors"
	"fmt"
	"net/netip"
	"os"
	"strconv"
	"strings"

	"github.com/snmp161/krot/internal/awg"
	"github.com/snmp161/krot/internal/cli"
	"github.com/snmp161/krot/internal/conf"
	"github.com/snmp161/krot/internal/errs"
	"github.com/snmp161/krot/internal/ipalloc"
	"github.com/snmp161/krot/internal/keys"
	"github.com/snmp161/krot/internal/qr"
)

// keepalive — PersistentKeepalive клиентского конфига (SPEC 6.5).
const keepalive = 25

// SPEC 6.5: --add-client.
func (a *App) addClient(cmd *cli.Command) error {
	var clientText string
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

		clientText = renderClient(ifc, priv.String(), addrStrs, serverPub.String(), psk, endpoint, cmd.AllowedIPs)
		return nil
	})
	if err != nil {
		return err
	}

	// Конфиг и QR показываются один раз; приватный ключ клиента нигде не хранится (SPEC 5.2).
	a.printf("%s", clientText)
	if cmd.QR {
		code, err := qr.Render(clientText)
		if err != nil {
			return errs.Invalidf("config too large for QR (client %s was added, use the text config above)", cmd.ID)
		}
		a.printf("\n%s", code)
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
	if _, err := strconv.Atoi(port); err != nil {
		return "", errs.Invalidf("interface %s has no valid ListenPort in %s", t.name, t.path)
	}
	if a, err := netip.ParseAddr(host); err == nil && a.Is6() {
		host = "[" + host + "]"
	}
	return host + ":" + port, nil
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
func renderClient(ifc conf.Interface, priv string, addrs []string, serverPub, psk, endpoint string, allowed []netip.Prefix) string {
	var b strings.Builder
	b.WriteString("[Interface]\n")
	b.WriteString("PrivateKey = " + priv + "\n")
	b.WriteString("Address = " + strings.Join(addrs, ", ") + "\n")
	if dns := ifc.Meta["dns"]; dns != "" {
		b.WriteString("DNS = " + strings.Join(strings.Split(dns, ","), ", ") + "\n")
	}
	if mtu := ifc.Get("MTU"); mtu != "" {
		b.WriteString("MTU = " + mtu + "\n")
	}
	// Все AWG-параметры сервера — без изменений, в каноническом порядке.
	for _, k := range awg.Keys {
		for _, kv := range ifc.Keys {
			if strings.EqualFold(kv.Key, k) {
				b.WriteString(k + " = " + kv.Value + "\n")
				break
			}
		}
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
