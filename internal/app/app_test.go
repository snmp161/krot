package app

import (
	"bytes"
	"encoding/binary"
	"flag"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sovchinnikov/krot/internal/errs"
	"github.com/sovchinnikov/krot/internal/execx"
	"github.com/sovchinnikov/krot/internal/fsutil"
)

var update = flag.Bool("update", false, "update golden files")

const (
	toolsV31 = "amneziawg-tools v3.1.20260812 - https://amnezia.org\n"
	goV31    = "amneziawg-go v3.1.20260828\n\nUserspace AmneziaWG daemon for linux-amd64.\nInformation available at https://amnezia.org\n"
	udpHdr   = "  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt   uid  timeout inode ref pointer drops\n"
)

var fixedNow = time.Date(2026, 10, 17, 3, 0, 0, 0, time.UTC)

// env — изолированное окружение: каталоги во временной папке, фейковый исполнитель,
// детерминированный rand и фиксированное время. Система не затрагивается.
type env struct {
	t      *testing.T
	app    *App
	fake   *execx.Fake
	stdout *bytes.Buffer
	stderr *bytes.Buffer
	stdin  string
	tty    bool
	root   string
}

func seededRand(seed uint64) *rand.ChaCha8 {
	var s [32]byte
	binary.LittleEndian.PutUint64(s[:], seed)
	return rand.NewChaCha8(s)
}

func newEnv(t *testing.T) *env {
	t.Helper()
	root := t.TempDir()
	e := &env{t: t, root: root, stdout: &bytes.Buffer{}, stderr: &bytes.Buffer{}, fake: &execx.Fake{}}
	paths := Paths{
		WGDir:     filepath.Join(root, "etc/wireguard"),
		AWGDir:    filepath.Join(root, "etc/amnezia/amneziawg"),
		LockFile:  filepath.Join(root, "run/lock/krot.lock"),
		ProcNet:   filepath.Join(root, "proc/net"),
		SysModule: filepath.Join(root, "sys/module"),
	}
	e.mkdir(paths.WGDir)
	e.mkdir(filepath.Join(paths.SysModule, "wireguard"))
	e.write(filepath.Join(paths.ProcNet, "udp"), udpHdr)
	e.write(filepath.Join(paths.ProcNet, "udp6"), udpHdr)

	// По умолчанию: имён интерфейсов в системе нет, модуля amneziawg нет, есть amneziawg-go 3.1.
	e.fake.Fail("ip link show", 1, `Device "x" does not exist.`)
	e.fake.On("awg --version", toolsV31)
	e.fake.Fail("modinfo -F version amneziawg", 1, "modinfo: ERROR: Module amneziawg not found.")
	e.fake.On("amneziawg-go --version", goV31)

	e.app = &App{
		Stdout:      e.stdout,
		Stderr:      e.stderr,
		Stdin:       strings.NewReader(""),
		IsTTY:       func() bool { return e.tty },
		Exec:        e.fake,
		Rand:        seededRand(1),
		Now:         func() time.Time { return fixedNow },
		Euid:        func() int { return 0 },
		Paths:       paths,
		LockTimeout: 200 * time.Millisecond,
		Version:     "v1.2.3",
		Commit:      "abc1234",
		Date:        "2026-10-17T00:00:00Z",
	}
	return e
}

func (e *env) mkdir(p string) {
	e.t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		e.t.Fatal(err)
	}
}

func (e *env) write(p, content string) {
	e.t.Helper()
	e.mkdir(filepath.Dir(p))
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		e.t.Fatal(err)
	}
}

func (e *env) read(p string) string {
	e.t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		e.t.Fatal(err)
	}
	return string(b)
}

func (e *env) exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func (e *env) wgConf(name string) string  { return filepath.Join(e.app.Paths.WGDir, name+".conf") }
func (e *env) awgConf(name string) string { return filepath.Join(e.app.Paths.AWGDir, name+".conf") }

func fixture(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// run запускает krot с аргументами; stdin и tty берутся из env.
func (e *env) run(args ...string) int {
	e.t.Helper()
	e.stdout.Reset()
	e.stderr.Reset()
	e.app.Stdin = strings.NewReader(e.stdin)
	return e.app.Run(args)
}

func (e *env) mustRun(want int, args ...string) {
	e.t.Helper()
	if got := e.run(args...); got != want {
		e.t.Fatalf("krot %s: exit %d; want %d\nstdout:\n%s\nstderr:\n%s\ncalls: %q",
			strings.Join(args, " "), got, want, e.stdout, e.stderr, e.fake.CommandLines())
	}
}

func (e *env) called(line string) bool {
	for _, c := range e.fake.CommandLines() {
		if c == line {
			return true
		}
	}
	return false
}

func (e *env) calledPrefix(prefix string) bool {
	for _, c := range e.fake.CommandLines() {
		if c == prefix || strings.HasPrefix(c, prefix+" ") {
			return true
		}
	}
	return false
}

// golden сравнивает с testdata/<name>; с -update перезаписывает файл.
func golden(t *testing.T, name, got string) {
	t.Helper()
	p := filepath.Join("testdata", name)
	if *update {
		if err := os.WriteFile(p, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("golden %s: %v (run go test -update after a deliberate change)", name, err)
	}
	if got != string(want) {
		t.Fatalf("golden %s mismatch\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
	}
}

// SPEC 4.1: --help и --version работают без root.
func TestHelpVersionWithoutRoot(t *testing.T) {
	e := newEnv(t)
	e.app.Euid = func() int { return 1000 }
	e.mustRun(0, "--help")
	if !strings.Contains(e.stdout.String(), "--add-client") {
		t.Fatalf("help output:\n%s", e.stdout)
	}
	e.mustRun(0, "--version")
	if got := e.stdout.String(); got != "krot v1.2.3 (commit abc1234, built 2026-10-17T00:00:00Z)\n" {
		t.Fatalf("version output: %q", got)
	}
	if len(e.fake.Calls) != 0 {
		t.Fatalf("unexpected calls: %q", e.fake.CommandLines())
	}
}

// SPEC 2.4: все действия, кроме --help и --version, требуют root (код 6).
func TestRequiresRoot(t *testing.T) {
	e := newEnv(t)
	e.app.Euid = func() int { return 1000 }
	for _, args := range [][]string{
		{"--list"},
		{"--create", "--type", "wg", "--ip", "10.9.0.1/24", "--port", "51821", "--endpoint", "203.0.113.1"},
		{"--up", "--interface", "wg0"},
		{"--list-client", "--interface", "wg0"},
	} {
		e.mustRun(errs.Env, args...)
		if !strings.Contains(e.stderr.String(), "root") {
			t.Errorf("%q: stderr = %q", args, e.stderr)
		}
	}
	if len(e.fake.Calls) != 0 {
		t.Fatalf("unexpected calls: %q", e.fake.CommandLines())
	}
	// Ошибки использования важнее проверки root.
	e.mustRun(errs.Usage, "--up")
}

// SPEC 5.5: не удалось получить блокировку — код 7, ничего не изменено.
func TestLockBusy(t *testing.T) {
	e := newEnv(t)
	l, err := fsutil.AcquireLock(e.app.Paths.LockFile, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Release()
	e.mustRun(errs.Lock, "--create", "--type", "wg", "--ip", "10.9.0.1/24", "--port", "51821", "--endpoint", "203.0.113.1")
	if e.exists(e.wgConf("wg0")) || e.calledPrefix("systemctl") {
		t.Fatal("changes made without the lock")
	}
}
