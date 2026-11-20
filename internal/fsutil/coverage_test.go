package fsutil

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// SPEC 5.4 T36: ошибки создания временного файла и открытия каталога возвращаются.
func TestAtomicWriteFilesystemErrors(t *testing.T) {
	root := t.TempDir()
	if err := WriteFileAtomic(filepath.Join(root, "missing", "wg0.conf"), []byte("data")); err == nil {
		t.Fatal("WriteFileAtomic succeeded in a missing directory")
	}
	if err := syncDir(filepath.Join(root, "missing")); err == nil {
		t.Fatal("syncDir succeeded for a missing directory")
	}
}

// SPEC 5.5 T36: невозможность создать каталог или открыть lock — обычная ошибка.
func TestLockFilesystemErrors(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "file")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := AcquireLock(filepath.Join(file, "krot.lock"), time.Millisecond); err == nil {
		t.Fatal("AcquireLock succeeded below a regular file")
	}
	if _, err := AcquireLock(root, time.Millisecond); err == nil {
		t.Fatal("AcquireLock opened a directory as a lock file")
	}
}
