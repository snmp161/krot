package app

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"text/tabwriter"

	"github.com/snmp161/krot/internal/awg"
	"github.com/snmp161/krot/internal/errs"
	"github.com/snmp161/krot/internal/i18n"
)

// SPEC 6.11: --doctor — самодиагностика окружения. Только читает: без root, без блокировки.
func (a *App) doctor() error {
	var lines []docLine
	fails := 0
	add := func(check, status, detail string) {
		lines = append(lines, docLine{i18n.Text(check), status, detail})
		if status == "fail" {
			fails++
		}
	}

	if a.Euid() == 0 {
		add("root", "ok", i18n.Text("running as root"))
	} else {
		add("root", "warn", i18n.Text("not root: tunnel/client actions require root (code 6); --udp-checker does not, and --udp-listener needs it only for privileged ports or unreadable /proc/net/udp{,6}"))
	}

	for _, p := range []string{"ip", "systemctl", "wg", "wg-quick", "awg", "awg-quick"} {
		if path, err := a.Exec.LookPath(p); err != nil {
			add(p, "fail", i18n.Sprintf("not found in PATH: install package %s", packages[p]))
		} else if p == "awg" {
			add(p, "ok", awgToolDetail(a, path))
		} else {
			add(p, "ok", path)
		}
	}

	a.doctorWGModule(add)
	a.doctorAmnezia(add)
	a.doctorDirs(add)

	w := tabwriter.NewWriter(a.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, i18n.Text("CHECK\tSTATUS\tDETAIL"))
	for _, l := range lines {
		fmt.Fprintf(w, "%s\t%s\t%s\n", l.check, l.status, l.detail)
	}
	if err := w.Flush(); err != nil {
		return err
	}
	if fails > 0 {
		return errs.New(errs.Env, "%d check(s) failed (see the table above)", fails)
	}
	return nil
}

type docLine struct {
	check, status, detail string
}

// awgToolDetail — путь к awg и версия инструментов, если она разбирается.
func awgToolDetail(a *App, path string) string {
	if r, err := a.Exec.Run("awg", "--version"); err == nil {
		if v, err := awg.ParseToolsOutput(string(r.Stdout)); err == nil {
			return i18n.Sprintf("%s (amneziawg-tools %s)", path, v)
		}
	}
	return path
}

// doctorWGModule — модуль ядра WireGuard (SPEC 2.2).
func (a *App) doctorWGModule(add func(string, string, string)) {
	switch st, err := os.Stat(filepath.Join(a.Paths.SysModule, "wireguard")); {
	case err == nil && st.IsDir():
		add("wireguard kernel module", "ok", i18n.Text("loaded"))
	case err != nil && !errors.Is(err, fs.ErrNotExist):
		add("wireguard kernel module", "fail", err.Error())
	default:
		if _, err := a.Exec.LookPath("modinfo"); err != nil {
			add("wireguard kernel module", "fail", i18n.Text("modinfo not found in PATH: install package kmod to check the module"))
			return
		}
		r, err := a.Exec.Run("modinfo", "-F", "version", "wireguard")
		if err != nil {
			add("wireguard kernel module", "fail", i18n.Text("not found"))
			return
		}
		add("wireguard kernel module", "ok", i18n.Sprintf("installed %s", strings.TrimSpace(string(r.Stdout))))
	}
}

// doctorAmnezia — поддерживаемые поколения AmneziaWG по правилам 3.1.
func (a *App) doctorAmnezia(add func(string, string, string)) {
	in := a.amneziaInstalled()
	src := amneziaSource(in)
	for _, t := range []awg.Type{awg.TypeAWG, awg.TypeAWG2, awg.TypeAWG3} {
		for _, i1 := range []bool{false, true} {
			if i1 && t != awg.TypeAWG {
				continue // 1.5 — только awg с --i1
			}
			req := awg.RequirementFor(t, i1)
			warnings, err := awg.CheckInstalled(req, in)
			check := i18n.Sprintf("amneziawg %s (%s)", req.Gen, i18n.Text(req.Label))
			switch {
			case err != nil:
				add(check, "fail", src+": "+err.Error())
			case len(warnings) > 0:
				add(check, "warn", warnings[0])
			default:
				add(check, "ok", src)
			}
		}
	}
}

// amneziaSource описывает найденный источник AmneziaWG: модуль ядра или amneziawg-go.
func amneziaSource(in awg.Installed) string {
	switch {
	case in.ModuleLoaded:
		s := i18n.Text("kernel module loaded")
		if v, err := awg.ParseModuleOutput(in.Module); err == nil {
			s += " " + v.String()
		}
		if in.Genl != "" {
			if n, err := awg.ParseGenlOutput(in.Genl); err == nil {
				s += i18n.Sprintf(" (netlink family %d)", n)
			}
		}
		return s
	case in.ModulePresent:
		if v, err := awg.ParseModuleOutput(in.Module); err == nil {
			return i18n.Sprintf("kernel module installed %s", v.String())
		}
		return i18n.Text("kernel module installed")
	case in.GoPresent:
		if v, err := awg.ParseGoOutput(in.Go); err == nil {
			return "amneziawg-go " + v.String()
		}
		return "amneziawg-go"
	}
	return i18n.Text("none found")
}

// doctorDirs — каталоги конфигов и каталог файла блокировки (SPEC 5).
func (a *App) doctorDirs(add func(string, string, string)) {
	for _, d := range []struct{ check, path string }{
		{"config dir (wg)", a.Paths.WGDir},
		{"config dir (awg)", a.Paths.AWGDir},
		{"lock file dir", filepath.Dir(a.Paths.LockFile)},
	} {
		switch st, err := os.Stat(d.path); {
		case err == nil && st.IsDir():
			add(d.check, "ok", d.path)
		case err == nil:
			add(d.check, "fail", i18n.Sprintf("%s exists and is not a directory", d.path))
		case !errors.Is(err, fs.ErrNotExist):
			add(d.check, "fail", err.Error())
		case accessWOK(filepath.Dir(d.path)):
			add(d.check, "ok", i18n.Sprintf("will be created: %s", d.path))
		default:
			add(d.check, "fail", i18n.Sprintf("%s is not writable", nearestExisting(filepath.Dir(d.path))))
		}
	}
}

// wOK — флаг проверки права на запись для access(2) (syscall не экспортирует константу).
const wOK = 0x2

// accessWOK проверяет право на запись, поднимаясь к ближайшему существующему предку.
func accessWOK(path string) bool {
	for {
		if _, err := os.Stat(path); err == nil {
			return syscall.Access(path, wOK) == nil
		}
		parent := filepath.Dir(path)
		if parent == path {
			return false
		}
		path = parent
	}
}

func nearestExisting(path string) string {
	for {
		if _, err := os.Stat(path); err == nil {
			return path
		}
		parent := filepath.Dir(path)
		if parent == path {
			return path
		}
		path = parent
	}
}
