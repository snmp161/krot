// Package app реализует действия krot поверх разбора CLI, конфигов и исполнителя команд (SPEC 6).
package app

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/snmp161/krot/internal/cli"
	"github.com/snmp161/krot/internal/errs"
	"github.com/snmp161/krot/internal/execx"
	"github.com/snmp161/krot/internal/fsutil"
)

// Paths — пути в системе; в тестах подменяются временными каталогами.
type Paths struct {
	WGDir     string // конфиги wg
	AWGDir    string // конфиги awg*
	LockFile  string // глобальная блокировка (SPEC 5.5)
	ProcNet   string // /proc/net — занятые UDP-порты
	SysModule string // /sys/module — загруженные модули ядра
}

// DefaultPaths — пути на реальной системе (SPEC 3, 5.5).
func DefaultPaths() Paths {
	return Paths{
		WGDir:     "/etc/wireguard",
		AWGDir:    "/etc/amnezia/amneziawg",
		LockFile:  "/run/lock/krot.lock",
		ProcNet:   "/proc/net",
		SysModule: "/sys/module",
	}
}

// App — всё окружение krot. Поля подменяются в тестах.
type App struct {
	Stdout io.Writer
	Stderr io.Writer
	Stdin  io.Reader
	IsTTY  func() bool // stdin — терминал
	Exec   execx.Runner
	Rand   io.Reader // crypto/rand в рабочем коде
	Now    func() time.Time
	Euid   func() int
	Paths  Paths

	LockTimeout time.Duration

	Version, Commit, Date string
}

// New создаёт App для реальной системы.
func New(version, commit, date string) *App {
	return &App{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Stdin:  os.Stdin,
		IsTTY: func() bool {
			st, err := os.Stdin.Stat()
			return err == nil && st.Mode()&os.ModeCharDevice != 0
		},
		Exec:        execx.OS{},
		Rand:        rand.Reader,
		Now:         time.Now,
		Euid:        os.Geteuid,
		Paths:       DefaultPaths(),
		LockTimeout: 10 * time.Second,
		Version:     version,
		Commit:      commit,
		Date:        date,
	}
}

// Run выполняет krot с аргументами и возвращает код выхода (SPEC 4.5).
func (a *App) Run(args []string) int {
	err := a.run(args)
	if err != nil {
		fmt.Fprintf(a.Stderr, "krot: %v\n", err)
	}
	return errs.Code(err)
}

func (a *App) run(args []string) error {
	cmd, err := cli.Parse(args)
	if err != nil {
		return err
	}
	switch {
	case cmd.Help:
		fmt.Fprint(a.Stdout, cli.Usage())
		return nil
	case cmd.Version:
		fmt.Fprintf(a.Stdout, "krot %s (commit %s, built %s)\n", a.Version, a.Commit, a.Date)
		return nil
	}
	if a.Euid() != 0 {
		return errs.Envf("root privileges are required: run krot as root")
	}
	switch cmd.Action {
	case cli.ActCreate:
		return a.create(cmd)
	case cli.ActDelete:
		return a.deleteTunnel(cmd)
	case cli.ActUp:
		return a.upDown(cmd, true)
	case cli.ActDown:
		return a.upDown(cmd, false)
	case cli.ActAddClient:
		return a.addClient(cmd)
	case cli.ActDeleteClient:
		return a.deleteClient(cmd)
	case cli.ActLockClient:
		return a.lockClient(cmd)
	case cli.ActUnlockClient:
		return a.unlockClient(cmd)
	case cli.ActList:
		return a.list(cmd)
	case cli.ActListClient:
		return a.listClient(cmd)
	}
	return errs.New(errs.Internal, "action --%s is not implemented", cmd.Action)
}

func (a *App) warnf(format string, args ...any) {
	fmt.Fprintf(a.Stderr, "krot: warning: "+format+"\n", args...)
}

func (a *App) printf(format string, args ...any) {
	fmt.Fprintf(a.Stdout, format, args...)
}

// withLock выполняет изменяющее действие под глобальной блокировкой (SPEC 5.5).
func (a *App) withLock(fn func() error) error {
	l, err := fsutil.AcquireLock(a.Paths.LockFile, a.LockTimeout)
	if err != nil {
		if errors.Is(err, fsutil.ErrLockTimeout) {
			return errs.Lockf("could not acquire %s within %s: another krot is running", a.Paths.LockFile, a.LockTimeout)
		}
		return errs.Lockf("could not acquire %s: %v", a.Paths.LockFile, err)
	}
	defer l.Release()
	return fn()
}
