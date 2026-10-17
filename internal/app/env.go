package app

import (
	"os"
	"path/filepath"

	"github.com/snmp161/krot/internal/awg"
	"github.com/snmp161/krot/internal/errs"
)

// Пакеты, из которых ставятся внешние программы (SPEC 2.2).
var packages = map[string]string{
	"wg":           "wireguard-tools",
	"wg-quick":     "wireguard-tools",
	"awg":          "amneziawg-tools",
	"awg-quick":    "amneziawg-tools",
	"ip":           "iproute2",
	"systemctl":    "systemd",
	"amneziawg-go": "amneziawg-go",
}

// need проверяет наличие программ в PATH (SPEC 2.2, код 6).
func (a *App) need(progs ...string) error {
	for _, p := range progs {
		if _, err := a.Exec.LookPath(p); err != nil {
			return errs.Envf("%s not found in PATH: install package %s", p, packages[p])
		}
	}
	return nil
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// checkWGModule — модуль ядра WireGuard загружен или установлен (SPEC 2.2).
func (a *App) checkWGModule() error {
	if exists(filepath.Join(a.Paths.SysModule, "wireguard")) {
		return nil
	}
	if _, err := a.Exec.Run("modinfo", "-F", "version", "wireguard"); err == nil {
		return nil
	}
	return errs.Envf("WireGuard kernel module (wireguard) not found")
}

// checkAmnezia проверяет, что установленная AmneziaWG поддерживает тип (SPEC 3.1).
func (a *App) checkAmnezia(t awg.Type, i1 bool) error {
	var in awg.Installed
	if r, err := a.Exec.Run("awg", "--version"); err == nil {
		in.Tools = string(r.Stdout)
	}
	modDir := filepath.Join(a.Paths.SysModule, "amneziawg")
	switch {
	case exists(modDir):
		in.ModuleLoaded, in.ModulePresent = true, true
		if b, err := os.ReadFile(filepath.Join(modDir, "version")); err == nil {
			in.Module = string(b)
		}
		if _, err := a.Exec.LookPath("genl"); err == nil {
			if r, err := a.Exec.Run("genl", "ctrl", "get", "name", "amneziawg"); err == nil {
				in.Genl = string(r.Stdout)
			}
		}
	default:
		if r, err := a.Exec.Run("modinfo", "-F", "version", "amneziawg"); err == nil {
			in.ModulePresent, in.Module = true, string(r.Stdout)
		} else if _, err := a.Exec.LookPath("amneziawg-go"); err == nil {
			in.GoPresent = true
			if r, err := a.Exec.Run("amneziawg-go", "--version"); err == nil {
				in.Go = string(r.Stdout)
			}
		}
	}
	warnings, err := awg.CheckInstalled(awg.RequirementFor(t, i1), in)
	for _, w := range warnings {
		a.warnf("%s", w)
	}
	if err != nil {
		return errs.Wrap(errs.Env, err)
	}
	return nil
}
