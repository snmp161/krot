// Package fsutil — атомарная запись конфигов (SPEC 5.4) и межпроцессная блокировка (SPEC 5.5).
package fsutil

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// testHook позволяет тестам внедрить ошибку на шаге записи; в рабочем коде nil.
var testHook func(step string) error

func step(name string) error {
	if testHook != nil {
		return testHook(name)
	}
	return nil
}

// WriteFileAtomic записывает файл атомарно: временный файл в том же каталоге (0600,
// root:root при запуске от root) → fsync → rename поверх оригинала → fsync каталога.
// При ошибке до rename оригинал не меняется, временный файл удаляется.
func WriteFileAtomic(path string, data []byte) (err error) {
	dir, base := filepath.Split(path)
	if dir == "" {
		dir = "."
	}
	if err := step("create"); err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, "."+base+".tmp-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	closed := false
	defer func() {
		if err != nil {
			if !closed {
				f.Close()
			}
			os.Remove(tmp)
		}
	}()

	if err = step("chmod"); err != nil {
		return err
	}
	if err = f.Chmod(0o600); err != nil {
		return err
	}
	if err = step("chown"); err != nil {
		return err
	}
	if os.Geteuid() == 0 {
		if err = f.Chown(0, 0); err != nil {
			return err
		}
	}
	if err = step("write"); err != nil {
		return err
	}
	if _, err = f.Write(data); err != nil {
		return err
	}
	if err = step("sync"); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = step("close"); err != nil {
		return err
	}
	closed = true
	if err = f.Close(); err != nil {
		return err
	}
	if err = step("rename"); err != nil {
		return err
	}
	if err = os.Rename(tmp, path); err != nil {
		return err
	}
	// Файл уже заменён целиком; ошибка ниже не должна удалять его.
	return syncDir(dir)
}

func syncDir(dir string) error {
	if err := step("syncdir"); err != nil {
		return err
	}
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

// ErrLockTimeout — не удалось получить блокировку за отведённое время (код 7).
var ErrLockTimeout = errors.New("timed out waiting for the lock")

// Lock — удерживаемая эксклюзивная flock-блокировка.
type Lock struct {
	f *os.File
}

// AcquireLock берёт эксклюзивный flock на файл, ожидая до timeout.
func AcquireLock(path string, timeout time.Duration) (*Lock, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(timeout)
	for {
		err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		switch {
		case err == nil:
			return &Lock{f: f}, nil
		case errors.Is(err, syscall.EINTR):
			continue
		case !errors.Is(err, syscall.EWOULDBLOCK):
			f.Close()
			return nil, err
		}
		if !time.Now().Before(deadline) {
			f.Close()
			return nil, ErrLockTimeout
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// Release снимает блокировку.
func (l *Lock) Release() error {
	err := syscall.Flock(int(l.f.Fd()), syscall.LOCK_UN)
	if cerr := l.f.Close(); err == nil {
		err = cerr
	}
	return err
}
