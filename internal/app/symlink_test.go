package app

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// linkConfig переносит конфиг в «управляемый» каталог (как у Salt/Ansible) и
// оставляет на его месте символическую ссылку; возвращает путь целевого файла.
func (e *env) linkConfig(path string) string {
	e.t.Helper()
	target := filepath.Join(e.root, "managed", filepath.Base(path))
	e.mkdir(filepath.Dir(target))
	if err := os.Rename(path, target); err != nil {
		e.t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		e.t.Fatal(err)
	}
	return target
}

// mustBeLink проверяет, что path по-прежнему ссылка на target.
func (e *env) mustBeLink(path, target string) {
	e.t.Helper()
	st, err := os.Lstat(path)
	if err != nil || st.Mode()&os.ModeSymlink == 0 {
		e.t.Fatalf("%s is no longer a symlink (err %v)", path, err)
	}
	if got, _ := os.Readlink(path); got != target {
		e.t.Fatalf("%s points to %q; want %q", path, got, target)
	}
}

// SPEC 5.4, 6.1: --list показывает туннель, чей конфиг — ссылка на обычный файл.
func TestListSymlinkedConfig(t *testing.T) {
	e := newEnv(t)
	e.mustRun(0, createArgs("wg")...)
	e.linkConfig(e.wgConf("wg0"))
	e.mustRun(0, "--list")
	if !strings.Contains(e.stdout.String(), "wg0") || !strings.Contains(e.stdout.String(), "51820") {
		t.Fatalf("symlinked config is missing from --list:\n%s", e.stdout)
	}
	e.mustRun(0, "--list", "--json")
	if !strings.Contains(e.stdout.String(), `"interface": "wg0"`) {
		t.Fatalf("symlinked config is missing from --list --json:\n%s", e.stdout)
	}
}

// SPEC 5.4, 6.2 п.3: порт конфига-ссылки занят — конфликт, код 5.
func TestCreatePortConflictWithSymlinkedConfig(t *testing.T) {
	e := newEnv(t)
	e.mustRun(0, createArgs("wg")...)
	e.linkConfig(e.wgConf("wg0"))
	e.mustRun(5, "--create", "--type", "awg2", "--ip", "10.9.0.1/24", "--port", "51820", "--endpoint", "vpn.example.com")
}

// SPEC 5.4, 6.5–6.8: изменения пишутся в целевой файл, ссылка сохраняется.
func TestClientOpsThroughSymlinkKeepLink(t *testing.T) {
	e := newEnv(t)
	e.mustRun(0, createArgs("wg")...)
	link := e.wgConf("wg0")
	target := e.linkConfig(link)
	steps := []struct {
		args []string
		want string
	}{
		{[]string{"--add-client", "--interface", "wg0", "--id", "a", "--allowed-ips", "0.0.0.0/0"}, "# krot:id=a"},
		{[]string{"--lock-client", "--interface", "wg0", "--id", "a"}, "#~ [Peer]"},
		{[]string{"--unlock-client", "--interface", "wg0", "--id", "a"}, "\n[Peer]"},
	}
	for _, s := range steps {
		e.mustRun(0, s.args...)
		e.mustBeLink(link, target)
		if !strings.Contains(e.read(target), s.want) {
			t.Fatalf("%s: target file misses %q:\n%s", s.args[0], s.want, e.read(target))
		}
	}
	e.mustRun(0, "--delete-client", "--interface", "wg0", "--id", "a")
	e.mustBeLink(link, target)
	if strings.Contains(e.read(target), "[Peer]") {
		t.Fatalf("client was not deleted from the target file:\n%s", e.read(target))
	}
	if st, _ := os.Stat(target); st.Mode().Perm() != 0o600 {
		t.Fatalf("target mode = %v; want 0600", st.Mode().Perm())
	}
}

// SPEC 5.4, 6.3: --delete удаляет только ссылку, целевой файл не изменяется.
func TestDeleteSymlinkedConfigRemovesOnlyLink(t *testing.T) {
	e := newEnv(t)
	e.mustRun(0, createArgs("wg")...)
	link := e.wgConf("wg0")
	target := e.linkConfig(link)
	before := e.read(target)
	e.mustRun(0, "--delete", "--interface", "wg0", "--yes")
	if _, err := os.Lstat(link); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("symlink still exists: %v", err)
	}
	if got := e.read(target); got != before {
		t.Fatalf("target file changed:\n%s", got)
	}
	if !e.called("systemctl disable --now wg-quick@wg0") {
		t.Fatalf("unit not disabled: %q", e.fake.CommandLines())
	}
}

// SPEC 5.4: битая ссылка — предупреждение и пропуск при обходе каталогов, код 4
// для действий по --interface, конфликт 5 для --create с тем же именем.
func TestDanglingSymlinkConfig(t *testing.T) {
	e := newEnv(t)
	link := e.wgConf("wg0")
	missing := filepath.Join(e.root, "managed", "wg0.conf")
	if err := os.Symlink(missing, link); err != nil {
		t.Fatal(err)
	}
	e.mustRun(0, "--list")
	if strings.Contains(e.stdout.String(), "wg0") {
		t.Fatalf("dangling symlink must be skipped:\n%s", e.stdout)
	}
	if !strings.Contains(e.stderr.String(), "warning") || !strings.Contains(e.stderr.String(), link) {
		t.Fatalf("no warning about the dangling symlink: %q", e.stderr)
	}
	e.mustRun(4, "--add-client", "--interface", "wg0", "--id", "a", "--allowed-ips", "0.0.0.0/0")
	e.mustRun(5, append(createArgs("wg"), "--interface", "wg0")...)
	e.mustRun(0, createArgs("wg")...)
	if !strings.Contains(e.stdout.String(), "wg1") {
		t.Fatalf("automatic name must skip the dangling wg0:\n%s", e.stdout)
	}
	if got, err := os.Readlink(link); err != nil || got != missing {
		t.Fatalf("dangling symlink changed: %q, %v", got, err)
	}
}
