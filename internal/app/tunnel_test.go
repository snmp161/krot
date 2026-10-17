package app

import (
	"strings"
	"testing"

	"github.com/snmp161/krot/internal/errs"
)

// SPEC 6.3: удаление туннеля.
func TestDelete(t *testing.T) {
	t.Run("not found", func(t *testing.T) {
		e := newEnv(t)
		e.mustRun(errs.NotFound, "--delete", "--interface", "wg0")
	})
	t.Run("both dirs", func(t *testing.T) {
		e := newEnv(t)
		e.write(e.wgConf("wg0"), fixture(t, "wg0_empty.conf"))
		e.write(e.awgConf("wg0"), fixture(t, "wg0_empty.conf"))
		e.mustRun(errs.Conflict, "--delete", "--interface", "wg0")
	})
	t.Run("no clients, no prompt", func(t *testing.T) {
		e := newEnv(t)
		e.write(e.wgConf("wg0"), fixture(t, "wg0_empty.conf"))
		e.mustRun(0, "--delete", "--interface", "wg0")
		if e.exists(e.wgConf("wg0")) || !e.called("systemctl disable --now wg-quick@wg0") {
			t.Fatalf("not deleted; calls %q", e.fake.CommandLines())
		}
		if strings.Contains(e.stderr.String(), "Delete?") {
			t.Fatal("prompted without clients")
		}
	})
	t.Run("clients with --yes", func(t *testing.T) {
		e := newEnv(t)
		e.write(e.awgConf("awg0"), fixture(t, "awg0_two_clients.conf"))
		e.mustRun(0, "--delete", "--interface", "awg0", "--yes")
		if e.exists(e.awgConf("awg0")) || !e.called("systemctl disable --now awg-quick@awg0") {
			t.Fatal("not deleted")
		}
	})
	t.Run("clients, no tty", func(t *testing.T) {
		e := newEnv(t)
		e.write(e.awgConf("awg0"), fixture(t, "awg0_two_clients.conf"))
		e.mustRun(errs.Usage, "--delete", "--interface", "awg0")
		if !e.exists(e.awgConf("awg0")) || e.calledPrefix("systemctl") {
			t.Fatal("deleted without confirmation")
		}
		if !strings.Contains(e.stderr.String(), "--yes") {
			t.Fatalf("stderr = %q", e.stderr)
		}
	})
	for _, answer := range []string{"y\n", "yes\n", "Y\n", " YES \n"} {
		t.Run("tty answer "+strings.TrimSpace(answer), func(t *testing.T) {
			e := newEnv(t)
			e.tty, e.stdin = true, answer
			e.write(e.awgConf("awg0"), fixture(t, "awg0_two_clients.conf"))
			e.mustRun(0, "--delete", "--interface", "awg0")
			if !strings.Contains(e.stderr.String(), "Interface awg0 has 2 clients. Delete? [y/N]") {
				t.Fatalf("prompt = %q", e.stderr)
			}
			if e.exists(e.awgConf("awg0")) {
				t.Fatal("not deleted")
			}
		})
	}
	for _, answer := range []string{"n\n", "\n", "", "no\n", "maybe\n"} {
		t.Run("tty answer no "+strings.TrimSpace(answer), func(t *testing.T) {
			e := newEnv(t)
			e.tty, e.stdin = true, answer
			e.write(e.awgConf("awg0"), fixture(t, "awg0_two_clients.conf"))
			e.mustRun(errs.Cancelled, "--delete", "--interface", "awg0")
			if !e.exists(e.awgConf("awg0")) || e.calledPrefix("systemctl") {
				t.Fatal("deleted despite the answer")
			}
		})
	}
	t.Run("foreign config", func(t *testing.T) {
		e := newEnv(t)
		e.write(e.awgConf("office"), fixture(t, "foreign_wg.conf"))
		e.mustRun(0, "--delete", "--interface", "office", "--yes")
		if !e.called("systemctl disable --now awg-quick@office") || e.exists(e.awgConf("office")) {
			t.Fatal("foreign config not deleted")
		}
	})
	t.Run("systemctl fails", func(t *testing.T) {
		e := newEnv(t)
		e.write(e.wgConf("wg0"), fixture(t, "wg0_empty.conf"))
		e.fake.Fail("systemctl disable", 1, "Failed to disable unit")
		e.mustRun(errs.Internal, "--delete", "--interface", "wg0")
		if !e.exists(e.wgConf("wg0")) {
			t.Fatal("config removed although systemctl failed")
		}
	})
}

// SPEC 6.4: --up/--down через systemctl start|stop, идемпотентно.
func TestUpDown(t *testing.T) {
	t.Run("up", func(t *testing.T) {
		e := newEnv(t)
		e.write(e.wgConf("wg0"), fixture(t, "wg0_empty.conf"))
		e.fake.On("wg show interfaces", "wg9\n")
		e.mustRun(0, "--up", "--interface", "wg0")
		if !e.called("systemctl start wg-quick@wg0") {
			t.Fatalf("calls %q", e.fake.CommandLines())
		}
	})
	t.Run("already up", func(t *testing.T) {
		e := newEnv(t)
		e.write(e.wgConf("wg0"), fixture(t, "wg0_empty.conf"))
		e.fake.On("wg show interfaces", "wg9 wg0\n")
		e.mustRun(0, "--up", "--interface", "wg0")
		if e.calledPrefix("systemctl") || !strings.Contains(e.stdout.String(), "already up") {
			t.Fatalf("stdout %q calls %q", e.stdout, e.fake.CommandLines())
		}
	})
	t.Run("down", func(t *testing.T) {
		e := newEnv(t)
		e.write(e.awgConf("awg0"), fixture(t, "awg0_two_clients.conf"))
		e.fake.On("awg show interfaces", "awg0\n")
		e.mustRun(0, "--down", "--interface", "awg0")
		if !e.called("systemctl stop awg-quick@awg0") {
			t.Fatalf("calls %q", e.fake.CommandLines())
		}
	})
	t.Run("already down", func(t *testing.T) {
		e := newEnv(t)
		e.write(e.awgConf("awg0"), fixture(t, "awg0_two_clients.conf"))
		e.mustRun(0, "--down", "--interface", "awg0")
		if e.calledPrefix("systemctl") || !strings.Contains(e.stdout.String(), "already down") {
			t.Fatalf("stdout %q calls %q", e.stdout, e.fake.CommandLines())
		}
	})
	t.Run("not found", func(t *testing.T) {
		e := newEnv(t)
		e.mustRun(errs.NotFound, "--up", "--interface", "wg0")
		e.mustRun(errs.NotFound, "--down", "--interface", "wg0")
	})
	t.Run("systemctl fails", func(t *testing.T) {
		e := newEnv(t)
		e.write(e.wgConf("wg0"), fixture(t, "wg0_empty.conf"))
		e.fake.Fail("systemctl start", 1, "Job failed")
		e.mustRun(errs.Internal, "--up", "--interface", "wg0")
	})
	t.Run("tool missing", func(t *testing.T) {
		e := newEnv(t)
		e.write(e.awgConf("awg0"), fixture(t, "awg0_two_clients.conf"))
		e.fake.Missing = map[string]bool{"awg": true}
		e.mustRun(errs.Env, "--up", "--interface", "awg0")
		if !strings.Contains(e.stderr.String(), "amneziawg-tools") {
			t.Fatalf("stderr %q", e.stderr)
		}
	})
}
