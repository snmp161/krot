package app

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/snmp161/krot/internal/awg"
)

// SPEC 2, 5 T36: production App имеет полный набор системных зависимостей по умолчанию.
func TestDefaultAppConfiguration(t *testing.T) {
	a := New("v1", "commit", "date")
	if a.Stdout != os.Stdout || a.Stderr != os.Stderr || a.Stdin != os.Stdin {
		t.Error("standard streams are not wired")
	}
	if a.Exec == nil || a.Rand == nil || a.IsTTY == nil || a.Now == nil || a.Euid == nil {
		t.Fatal("system dependencies are not wired")
	}
	if a.Now().IsZero() || a.LockTimeout.String() != "10s" {
		t.Error("clock or lock timeout is not initialized")
	}
	if a.Version != "v1" || a.Commit != "commit" || a.Date != "date" {
		t.Error("build information is not preserved")
	}
	paths := DefaultPaths()
	if paths.WGDir != "/etc/wireguard" || paths.AWGDir != "/etc/amnezia/amneziawg" ||
		paths.LockFile != "/run/lock/krot.lock" || paths.ProcNet != "/proc/net" || paths.SysModule != "/sys/module" {
		t.Fatalf("unexpected default paths: %+v", paths)
	}
}

// SPEC 6.11 T36: модуль WireGuard различает loaded, installed и ошибки окружения.
func TestDoctorWGModuleBranches(t *testing.T) {
	collect := func(e *env) []docLine {
		t.Helper()
		var lines []docLine
		e.app.doctorWGModule(func(check, status, detail string) {
			lines = append(lines, docLine{check, status, detail})
		})
		return lines
	}

	t.Run("stat error", func(t *testing.T) {
		e := newEnv(t)
		file := filepath.Join(e.root, "not-dir")
		e.write(file, "x")
		e.app.Paths.SysModule = file
		if lines := collect(e); len(lines) != 1 || lines[0].status != "fail" {
			t.Fatalf("lines: %+v", lines)
		}
	})
	t.Run("modinfo missing", func(t *testing.T) {
		e := newEnv(t)
		if err := os.Remove(filepath.Join(e.app.Paths.SysModule, "wireguard")); err != nil {
			t.Fatal(err)
		}
		e.fake.Missing = map[string]bool{"modinfo": true}
		lines := collect(e)
		if lines[0].status != "fail" || !strings.Contains(lines[0].detail, "kmod") {
			t.Fatalf("lines: %+v", lines)
		}
	})
	t.Run("module absent", func(t *testing.T) {
		e := newEnv(t)
		if err := os.Remove(filepath.Join(e.app.Paths.SysModule, "wireguard")); err != nil {
			t.Fatal(err)
		}
		e.fake.Fail("modinfo -F version wireguard", 1, "not found")
		if lines := collect(e); lines[0].status != "fail" || lines[0].detail != "not found" {
			t.Fatalf("lines: %+v", lines)
		}
	})
	t.Run("module installed", func(t *testing.T) {
		e := newEnv(t)
		if err := os.Remove(filepath.Join(e.app.Paths.SysModule, "wireguard")); err != nil {
			t.Fatal(err)
		}
		e.fake.On("modinfo -F version wireguard", "1.2.3\n")
		lines := collect(e)
		if lines[0].status != "ok" || !strings.Contains(lines[0].detail, "1.2.3") {
			t.Fatalf("lines: %+v", lines)
		}
	})
}

// SPEC 6.11 T36: doctor показывает источник AmneziaWG и ошибки каталогов.
func TestDoctorSourcesAndDirectories(t *testing.T) {
	sourceTests := []struct {
		name string
		in   awg.Installed
		want string
	}{
		{"loaded", awg.Installed{ModuleLoaded: true, Module: "3.1.20260812\n", Genl: "Version: 0x3"}, "netlink family 3"},
		{"loaded unknown", awg.Installed{ModuleLoaded: true, Module: "unknown", Genl: "bad"}, "kernel module loaded"},
		{"installed", awg.Installed{ModulePresent: true, Module: "3.1.20260812"}, "kernel module installed 3.1.20260812"},
		{"installed unknown", awg.Installed{ModulePresent: true, Module: "unknown"}, "kernel module installed"},
		{"go", awg.Installed{GoPresent: true, Go: goV31}, "amneziawg-go 3.1.20260828"},
		{"go unknown", awg.Installed{GoPresent: true, Go: "unknown"}, "amneziawg-go"},
		{"none", awg.Installed{}, "none found"},
	}
	for _, tt := range sourceTests {
		t.Run(tt.name, func(t *testing.T) {
			if got := amneziaSource(tt.in); !strings.Contains(got, tt.want) {
				t.Fatalf("amneziaSource() = %q, want substring %q", got, tt.want)
			}
		})
	}

	e := newEnv(t)
	if err := os.Remove(e.app.Paths.WGDir); err != nil {
		t.Fatal(err)
	}
	e.write(e.app.Paths.WGDir, "x")
	var lines []docLine
	e.app.doctorDirs(func(check, status, detail string) {
		lines = append(lines, docLine{check, status, detail})
	})
	if len(lines) != 3 || lines[0].status != "fail" || !strings.Contains(lines[0].detail, "not a directory") {
		t.Fatalf("directory lines: %+v", lines)
	}
	root := t.TempDir()
	missing := filepath.Join(root, "one", "two")
	if got := nearestExisting(missing); got != root {
		t.Fatalf("nearestExisting() = %q, want %q", got, root)
	}
	if !accessWOK(missing) {
		t.Error("writable ancestor was not accepted")
	}
}

// SPEC 6.11 T36: нечитаемая версия tools не скрывает путь к awg.
func TestDoctorToolDetailFallback(t *testing.T) {
	e := newEnv(t)
	e.fake.On("awg --version", "unrecognized")
	if got := awgToolDetail(e.app, "/usr/bin/awg"); got != "/usr/bin/awg" {
		t.Fatalf("invalid version detail = %q", got)
	}
	e.fake.Fail("awg --version", 1, "failed")
	if got := awgToolDetail(e.app, "/usr/bin/awg"); got != "/usr/bin/awg" {
		t.Fatalf("failed version detail = %q", got)
	}
}

// SPEC 6.9 T36: форматирование ключей и ошибки writer не теряются.
func TestOutputErrorAndShortKeyBranches(t *testing.T) {
	if got := shortKey(""); got != "-" {
		t.Fatalf("empty key = %q", got)
	}
	if got := shortKey("short"); got != "short" {
		t.Fatalf("short key = %q", got)
	}
	var out bytes.Buffer
	if err := printJSON(&out, []int(nil)); err != nil || out.String() != "[]\n" {
		t.Fatalf("nil JSON = %q, %v", out.String(), err)
	}
	if err := printJSON(errorWriter{}, []int{1}); err == nil {
		t.Fatal("writer error was ignored")
	}
}

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }
