package fsutil

import (
	"os"
	"path/filepath"
	"testing"
)

// SPEC 5.4: конфиг-ссылка сохраняется, как в wg-quick: временный файл и rename —
// в каталоге целевого файла, заменяется целевой файл, а не ссылка.
func TestWriteFileAtomicKeepsSymlink(t *testing.T) {
	linkDir, targetDir := t.TempDir(), t.TempDir()
	target := filepath.Join(targetDir, "managed-wg0.conf")
	if err := os.WriteFile(target, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(linkDir, "wg0.conf")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomic(link, []byte("new\n")); err != nil {
		t.Fatal(err)
	}
	st, err := os.Lstat(link)
	if err != nil || st.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("symlink replaced: mode %v, err %v", st.Mode(), err)
	}
	if got, _ := os.Readlink(link); got != target {
		t.Fatalf("symlink points to %q; want %q", got, target)
	}
	b, _ := os.ReadFile(target)
	if string(b) != "new\n" {
		t.Fatalf("target content = %q", b)
	}
	if st, _ := os.Stat(target); st.Mode().Perm() != 0o600 {
		t.Fatalf("target mode = %v; want 0600", st.Mode().Perm())
	}
	if names := dirNames(t, linkDir); len(names) != 1 {
		t.Fatalf("leftovers next to the link: %v", names)
	}
	if names := dirNames(t, targetDir); len(names) != 1 || names[0] != "managed-wg0.conf" {
		t.Fatalf("leftovers next to the target: %v", names)
	}
}

// SPEC 5.4: относительная ссылка разрешается от каталога ссылки.
func TestWriteFileAtomicRelativeSymlink(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "managed"), 0o700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, "managed", "wg0.conf")
	if err := os.WriteFile(target, []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "wg0.conf")
	if err := os.Symlink("managed/wg0.conf", link); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomic(link, []byte("new\n")); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.Readlink(link); got != "managed/wg0.conf" {
		t.Fatalf("symlink changed to %q", got)
	}
	if b, _ := os.ReadFile(target); string(b) != "new\n" {
		t.Fatalf("target content = %q", b)
	}
}

// SPEC 5.4: битая ссылка не заменяется обычным файлом.
func TestWriteFileAtomicDanglingSymlink(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, "wg0.conf")
	missing := filepath.Join(dir, "missing", "wg0.conf")
	if err := os.Symlink(missing, link); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomic(link, []byte("new\n")); err == nil {
		t.Fatal("writing through a dangling symlink must fail")
	}
	if got, err := os.Readlink(link); err != nil || got != missing {
		t.Fatalf("dangling symlink changed: %q, %v", got, err)
	}
	if names := dirNames(t, dir); len(names) != 1 {
		t.Fatalf("leftovers: %v", names)
	}
}
