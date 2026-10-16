package app

import (
	"bufio"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sovchinnikov/krot/internal/awg"
	"github.com/sovchinnikov/krot/internal/cli"
	"github.com/sovchinnikov/krot/internal/conf"
	"github.com/sovchinnikov/krot/internal/errs"
	"github.com/sovchinnikov/krot/internal/fsutil"
	"github.com/sovchinnikov/krot/internal/keys"
)

// SPEC 6.2: --create.
func (a *App) create(cmd *cli.Command) error {
	typ := cmd.Type
	amnezia := typ.IsAmnezia()
	// 1. Окружение: утилиты, модуль, версия AmneziaWG.
	if err := a.need("systemctl", "ip"); err != nil {
		return err
	}
	if amnezia {
		if err := a.need("awg", "awg-quick"); err != nil {
			return err
		}
		if err := a.checkAmnezia(typ, cmd.I1 != ""); err != nil {
			return err
		}
	} else {
		if err := a.need("wg", "wg-quick"); err != nil {
			return err
		}
		if err := a.checkWGModule(); err != nil {
			return err
		}
	}
	mtu := cmd.MTU
	if mtu == 0 {
		mtu = awg.DefaultMTU(typ)
	}

	return a.withLock(func() error {
		// 2–3. Имя и конфликты.
		name, err := a.pickName(cmd, amnezia)
		if err != nil {
			return err
		}
		if err := a.checkPort(cmd.Port); err != nil {
			return err
		}
		// 4. Ключ сервера и параметры обфускации.
		priv, err := keys.NewPrivate(a.Rand)
		if err != nil {
			return err
		}
		params, err := awg.Generate(typ, mtu, cmd.I1, a.Rand)
		if err != nil {
			return err
		}
		if err := awg.Validate(typ, params, mtu); err != nil {
			return errs.Wrap(errs.Invalid, err)
		}
		meta := []conf.KV{{Key: "type", Value: string(typ)}, {Key: "endpoint", Value: cmd.Endpoint}}
		if len(cmd.DNS) > 0 {
			meta = append(meta, conf.KV{Key: "dns", Value: joinAddrs(cmd.DNS, ",")})
		}
		var kvs []conf.KV
		for _, p := range params {
			kvs = append(kvs, conf.KV{Key: p.Key, Value: p.Value})
		}
		addrs := prefixStrings(cmd.IPs)
		data := conf.RenderServer(conf.Server{
			Meta:       meta,
			PrivateKey: priv.String(),
			Addresses:  addrs,
			ListenPort: cmd.Port,
			MTU:        mtu,
			Params:     kvs,
		})

		// 5. Атомарная запись.
		dir := a.Paths.WGDir
		if amnezia {
			dir = a.Paths.AWGDir
		}
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
		path := filepath.Join(dir, name+".conf")
		if err := fsutil.WriteFileAtomic(path, data); err != nil {
			return err
		}

		// 6. Запуск; при ошибке — возврат системы в исходное состояние.
		unit := unitName(amnezia, name)
		if _, err := a.Exec.Run("systemctl", "enable", "--now", unit); err != nil {
			a.Exec.Run("systemctl", "disable", "--now", unit)
			os.Remove(path)
			return fmt.Errorf("failed to start %s, changes rolled back: %w", unit, err)
		}

		// 7. Сводка.
		a.printf("Interface %s created\n", name)
		a.printf("  type:     %s\n", typ)
		a.printf("  address:  %s\n", strings.Join(addrs, ", "))
		a.printf("  port:     %d\n", cmd.Port)
		a.printf("  endpoint: %s\n", cmd.Endpoint)
		return nil
	})
}

// pickName возвращает заданное имя, проверив конфликты, или первое свободное wgN/awgN (SPEC 6.2 п.2–3).
func (a *App) pickName(cmd *cli.Command, amnezia bool) (string, error) {
	if cmd.Has("interface") {
		name := cmd.Interface
		if a.configExists(name) {
			return "", errs.Conflictf("interface %s already exists", name)
		}
		if a.linkExists(name) {
			return "", errs.Conflictf("network interface %s already exists in the system", name)
		}
		return name, nil
	}
	prefix := "wg"
	if amnezia {
		prefix = "awg"
	}
	for i := 0; i < 1000; i++ {
		name := prefix + strconv.Itoa(i)
		if !a.configExists(name) && !a.linkExists(name) {
			return name, nil
		}
	}
	return "", errs.Conflictf("no free interface name %sN", prefix)
}

func (a *App) linkExists(name string) bool {
	_, err := a.Exec.Run("ip", "link", "show", "dev", name)
	return err == nil
}

// checkPort — порт не занят другим конфигом и не слушается в системе (SPEC 6.2 п.3).
func (a *App) checkPort(port int) error {
	tunnels, err := a.scanTunnels()
	if err != nil {
		return err
	}
	for _, t := range tunnels {
		if ifc, ok := t.conf.Interface(); ok && ifc.Get("ListenPort") == strconv.Itoa(port) {
			return errs.Conflictf("UDP port %d is already used by interface %s (%s)", port, t.name, t.path)
		}
	}
	for _, f := range []string{"udp", "udp6"} {
		if listened(filepath.Join(a.Paths.ProcNet, f), port) {
			return errs.Conflictf("UDP port %d is already in use in the system", port)
		}
	}
	return nil
}

// listened разбирает /proc/net/udp{,6}: local_address вида ADDR:PORT (hex).
func listened(path string, port int) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	s.Scan() // заголовок
	for s.Scan() {
		fields := strings.Fields(s.Text())
		if len(fields) < 2 {
			continue
		}
		i := strings.LastIndexByte(fields[1], ':')
		if i < 0 {
			continue
		}
		if p, err := strconv.ParseUint(fields[1][i+1:], 16, 16); err == nil && int(p) == port {
			return true
		}
	}
	return false
}

func prefixStrings(ps []netip.Prefix) []string {
	out := make([]string, len(ps))
	for i, p := range ps {
		out[i] = p.String()
	}
	return out
}

func joinAddrs(as []netip.Addr, sep string) string {
	out := make([]string, len(as))
	for i, a := range as {
		out[i] = a.String()
	}
	return strings.Join(out, sep)
}
