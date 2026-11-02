package app

import (
	"strings"
	"testing"

	"github.com/snmp161/krot/internal/errs"
)

// SPEC 6.11: --doctor — самодиагностика окружения. Работает без root и без блокировки,
// ничего не меняет; код 0, если нет fail.
func TestDoctorOK(t *testing.T) {
	e := newEnv(t)
	e.mustRun(0, "--doctor")
	// Абсолютные пути временного каталога заменены на /ROOT — golden стабилен.
	golden(t, "doctor_ok.txt", strings.ReplaceAll(e.stdout.String(), e.root, "/ROOT"))
	if e.stderr.String() != "" {
		t.Fatalf("stderr: %q", e.stderr)
	}
	// Только чтение: никаких обращений к systemctl и «живым» интерфейсам.
	for _, c := range e.fake.Calls {
		if c.Name == "systemctl" || (c.Name == "ip" && c.Args[0] == "link") {
			t.Fatalf("doctor must not run %q", c.String())
		}
	}
}

// SPEC 6.11: не root — warn у проверки root, остальное не мешает коду 0.
func TestDoctorNotRoot(t *testing.T) {
	e := newEnv(t)
	e.app.Euid = func() int { return 1000 }
	e.mustRun(0, "--doctor")
	if !strings.Contains(e.stdout.String(), "root") || !strings.Contains(e.stdout.String(), "warn") {
		t.Fatalf("stdout:\n%s", e.stdout)
	}
}

// SPEC 6.11: отсутствующие утилиты и AmneziaWG — fail, код 6.
func TestDoctorFailures(t *testing.T) {
	e := newEnv(t)
	e.fake.Missing = map[string]bool{"awg": true, "awg-quick": true, "amneziawg-go": true}
	e.mustRun(errs.Env, "--doctor")
	out := e.stdout.String()
	for _, s := range []string{"awg", "fail", "amneziawg"} {
		if !strings.Contains(out, s) {
			t.Fatalf("stdout misses %q:\n%s", s, out)
		}
	}
	if !strings.Contains(e.stderr.String(), "failed") {
		t.Fatalf("stderr: %q", e.stderr)
	}
}
