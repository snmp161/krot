package app

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/snmp161/krot/internal/cli"
	"github.com/snmp161/krot/internal/conf"
	"github.com/snmp161/krot/internal/errs"
	"github.com/snmp161/krot/internal/fsutil"
)

// tunnel — конфиг интерфейса на диске.
type tunnel struct {
	name    string
	path    string
	amnezia bool // конфиг в каталоге AmneziaWG
	conf    *conf.Config
}

// tool — утилита управления: wg или awg (SPEC 3).
func (t *tunnel) tool() string {
	if t.amnezia {
		return "awg"
	}
	return "wg"
}

// unit — systemd unit интерфейса (SPEC 3).
func (t *tunnel) unit() string { return unitName(t.amnezia, t.name) }

func unitName(amnezia bool, name string) string {
	if amnezia {
		return "awg-quick@" + name
	}
	return "wg-quick@" + name
}

type confDir struct {
	path    string
	amnezia bool
}

func (a *App) confDirs() []confDir {
	return []confDir{{a.Paths.WGDir, false}, {a.Paths.AWGDir, true}}
}

func readTunnel(path, name string, amnezia bool) (*tunnel, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return &tunnel{name: name, path: path, amnezia: amnezia, conf: conf.Parse(b)}, nil
}

// findTunnel ищет конфиг интерфейса в обоих каталогах: нет — код 4, в обоих — код 5.
func (a *App) findTunnel(name string) (*tunnel, error) {
	var found []*tunnel
	for _, d := range a.confDirs() {
		t, err := readTunnel(filepath.Join(d.path, name+".conf"), name, d.amnezia)
		switch {
		case err == nil:
			found = append(found, t)
		case !errors.Is(err, fs.ErrNotExist):
			return nil, err
		}
	}
	switch len(found) {
	case 0:
		return nil, errs.NotFoundf("interface %s not found (no %s.conf in %s or %s)",
			name, name, a.Paths.WGDir, a.Paths.AWGDir)
	case 1:
		return found[0], nil
	}
	return nil, errs.Conflictf("interface %s is configured in both %s and %s", name, found[0].path, found[1].path)
}

// configExists — есть ли конфиг с таким именем в любом из каталогов.
func (a *App) configExists(name string) bool {
	for _, d := range a.confDirs() {
		if exists(filepath.Join(d.path, name+".conf")) {
			return true
		}
	}
	return false
}

// scanTunnels читает все конфиги обоих каталогов (только файлы вида <interface>.conf),
// отсортированные по имени. Отсутствующий каталог считается пустым.
func (a *App) scanTunnels() ([]*tunnel, error) {
	var out []*tunnel
	for _, d := range a.confDirs() {
		entries, err := os.ReadDir(d.path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			name, ok := strings.CutSuffix(e.Name(), ".conf")
			if !ok || !e.Type().IsRegular() || !cli.ValidInterfaceName(name) {
				continue
			}
			t, err := readTunnel(filepath.Join(d.path, e.Name()), name, d.amnezia)
			if err != nil {
				return nil, err
			}
			out = append(out, t)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out, nil
}

// liveInterfaces — интерфейсы, существующие в системе, по `wg|awg show interfaces`.
func (a *App) liveInterfaces(tool string) ([]string, error) {
	r, err := a.Exec.Run(tool, "show", "interfaces")
	if err != nil {
		return nil, fmt.Errorf("cannot list live interfaces: %w", err)
	}
	return strings.Fields(string(r.Stdout)), nil
}

// isUp — интерфейс существует в системе (SPEC 6.1 STATUS).
func (a *App) isUp(t *tunnel) (bool, error) {
	live, err := a.liveInterfaces(t.tool())
	if err != nil {
		return false, err
	}
	return slices.Contains(live, t.name), nil
}

// save атомарно записывает конфиг туннеля (SPEC 5.4).
func (a *App) save(t *tunnel) error {
	if err := fsutil.WriteFileAtomic(t.path, t.conf.Bytes()); err != nil {
		return fmt.Errorf("write %s: %w", t.path, err)
	}
	return nil
}

// SPEC 6.3: --delete.
func (a *App) deleteTunnel(cmd *cli.Command) error {
	if err := a.need("systemctl"); err != nil {
		return err
	}
	t, err := a.findTunnel(cmd.Interface)
	if err != nil {
		return err
	}
	// Подтверждение спрашивается до блокировки, чтобы не держать её, пока человек думает.
	if n := len(t.conf.Peers()); n > 0 && !cmd.Yes {
		if !a.IsTTY() {
			return errs.Usagef("interface %s has %d clients: stdin is not a terminal, use --yes to delete without confirmation", t.name, n)
		}
		fmt.Fprintf(a.Stderr, "Interface %s has %d clients. Delete? [y/N] ", t.name, n)
		line, _ := bufio.NewReader(a.Stdin).ReadString('\n')
		if ans := strings.ToLower(strings.TrimSpace(line)); ans != "y" && ans != "yes" {
			return errs.Cancelledf("deletion of interface %s cancelled", t.name)
		}
	}
	return a.withLock(func() error {
		t, err := a.findTunnel(cmd.Interface)
		if err != nil {
			return err
		}
		if _, err := a.Exec.Run("systemctl", "disable", "--now", t.unit()); err != nil {
			return fmt.Errorf("failed to stop %s, config kept: %w", t.unit(), err)
		}
		if err := os.Remove(t.path); err != nil {
			return err
		}
		a.printf("Interface %s deleted\n", t.name)
		return nil
	})
}

// SPEC 6.4: --up / --down.
func (a *App) upDown(cmd *cli.Command, up bool) error {
	return a.withLock(func() error {
		t, err := a.findTunnel(cmd.Interface)
		if err != nil {
			return err
		}
		if err := a.need(t.tool(), "systemctl"); err != nil {
			return err
		}
		running, err := a.isUp(t)
		if err != nil {
			return err
		}
		verb, state := "stop", "down"
		if up {
			verb, state = "start", "up"
		}
		if running == up {
			a.printf("Interface %s is already %s\n", t.name, state)
			return nil
		}
		if _, err := a.Exec.Run("systemctl", verb, t.unit()); err != nil {
			return fmt.Errorf("systemctl %s %s failed: %w", verb, t.unit(), err)
		}
		a.printf("Interface %s is %s\n", t.name, state)
		return nil
	})
}
