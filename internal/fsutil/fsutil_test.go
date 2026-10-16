package fsutil

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func dirNames(t *testing.T, dir string) []string {
	t.Helper()
	es, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, e := range es {
		out = append(out, e.Name())
	}
	return out
}

// SPEC 5.4, 5.1: атомарная запись, права 0600.
func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wg0.conf")
	if err := WriteFileAtomic(path, []byte("one\n")); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomic(path, []byte("two\n")); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	if string(b) != "two\n" {
		t.Fatalf("content = %q", b)
	}
	st, _ := os.Stat(path)
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v; want 0600", st.Mode().Perm())
	}
	if names := dirNames(t, dir); len(names) != 1 || names[0] != "wg0.conf" {
		t.Fatalf("leftovers: %v", names)
	}
}

// SPEC 5.4, 8.2: при ошибке посередине оригинальный файл не изменён, временный файл удалён.
func TestWriteFileAtomicFailure(t *testing.T) {
	for _, step := range []string{"create", "chmod", "chown", "write", "sync", "close", "rename"} {
		t.Run(step, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "awg0.conf")
			if err := os.WriteFile(path, []byte("original\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			boom := errors.New("injected failure")
			testHook = func(s string) error {
				if s == step {
					return boom
				}
				return nil
			}
			defer func() { testHook = nil }()

			if err := WriteFileAtomic(path, []byte("new content\n")); !errors.Is(err, boom) {
				t.Fatalf("err = %v; want injected failure", err)
			}
			b, _ := os.ReadFile(path)
			if string(b) != "original\n" {
				t.Fatalf("original changed: %q", b)
			}
			if names := dirNames(t, dir); len(names) != 1 {
				t.Fatalf("leftovers: %v", names)
			}
		})
	}
}

// Ошибка fsync каталога после rename возвращается, файл уже целиком новый.
func TestWriteFileAtomicSyncDirFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wg0.conf")
	boom := errors.New("injected failure")
	testHook = func(s string) error {
		if s == "syncdir" {
			return boom
		}
		return nil
	}
	defer func() { testHook = nil }()
	if err := WriteFileAtomic(path, []byte("new\n")); !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
	if b, _ := os.ReadFile(path); string(b) != "new\n" {
		t.Fatalf("content = %q", b)
	}
}

// Временный файл — в том же каталоге, скрытый и без суффикса .conf (не попадает в --list).
func TestTempName(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "wg0.conf")
	var seen []string
	testHook = func(s string) error {
		if s == "write" {
			seen = dirNames(t, dir)
		}
		return nil
	}
	defer func() { testHook = nil }()
	if err := WriteFileAtomic(path, []byte("x\n")); err != nil {
		t.Fatal(err)
	}
	if len(seen) != 1 || !strings.HasPrefix(seen[0], ".wg0.conf.tmp-") || strings.HasSuffix(seen[0], ".conf") {
		t.Fatalf("temp files = %v", seen)
	}
}

// SPEC 5.5: эксклюзивный flock с ожиданием, по истечении — ErrLockTimeout (код 7).
func TestLock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "krot.lock")
	l1, err := AcquireLock(path, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	if _, err := AcquireLock(path, 200*time.Millisecond); !errors.Is(err, ErrLockTimeout) {
		t.Fatalf("second lock: err = %v; want ErrLockTimeout", err)
	}
	if d := time.Since(start); d < 200*time.Millisecond {
		t.Fatalf("gave up too early: %v", d)
	}

	got := make(chan error, 1)
	go func() {
		l2, err := AcquireLock(path, 5*time.Second)
		if err == nil {
			err = l2.Release()
		}
		got <- err
	}()
	time.Sleep(100 * time.Millisecond)
	if err := l1.Release(); err != nil {
		t.Fatal(err)
	}
	if err := <-got; err != nil {
		t.Fatalf("waiting lock: %v", err)
	}
}
