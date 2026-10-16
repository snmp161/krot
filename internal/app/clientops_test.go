package app

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/sovchinnikov/krot/internal/conf"
	"github.com/sovchinnikov/krot/internal/errs"
	"github.com/sovchinnikov/krot/internal/execx"
)

const lockTime = "2026-10-17T03:00:00Z"

// expected применяет операцию пакета conf (покрыт golden-тестами, SPEC 8.2) к исходному конфигу.
func expected(t *testing.T, orig string, op func(c *conf.Config) error) string {
	t.Helper()
	c := conf.Parse([]byte(orig))
	if err := op(c); err != nil {
		t.Fatal(err)
	}
	return string(c.Bytes())
}

func peerKey(t *testing.T, cfg, id string) string {
	t.Helper()
	p, ok := conf.Parse([]byte(cfg)).PeerByID(id)
	if !ok {
		t.Fatalf("no peer %s", id)
	}
	return p.PublicKey()
}

// SPEC 6.6: --delete-client.
func TestDeleteClient(t *testing.T) {
	t.Run("active, interface up", func(t *testing.T) {
		e := newEnv(t)
		path := setupAWG0(t, e, true)
		orig := e.read(path)
		e.mustRun(0, "--delete-client", "--interface", "awg0", "--id", "phone-sergei")
		if got := e.read(path); got != expected(t, orig, func(c *conf.Config) error { return c.RemovePeer("phone-sergei") }) {
			t.Fatalf("config:\n%s", got)
		}
		if !e.called("awg set awg0 peer " + peerKey(t, orig, "phone-sergei") + " remove") {
			t.Fatalf("calls %q", e.fake.CommandLines())
		}
		assertOnlyPeerOps(t, e, "awg", "awg0")
	})
	t.Run("active, interface down", func(t *testing.T) {
		e := newEnv(t)
		path := setupAWG0(t, e, false)
		e.mustRun(0, "--delete-client", "--interface", "awg0", "--id", "phone-sergei")
		if e.calledPrefix("awg set") {
			t.Fatal("live call on a down interface")
		}
		if _, ok := conf.Parse([]byte(e.read(path))).PeerByID("phone-sergei"); ok {
			t.Fatal("peer not removed")
		}
		assertOnlyPeerOps(t, e, "awg", "awg0")
	})
	t.Run("locked, interface up", func(t *testing.T) {
		e := newEnv(t)
		path := setupAWG0(t, e, true)
		orig := e.read(path)
		e.mustRun(0, "--delete-client", "--interface", "awg0", "--id", "laptop-old")
		if got := e.read(path); got != expected(t, orig, func(c *conf.Config) error { return c.RemovePeer("laptop-old") }) {
			t.Fatalf("config:\n%s", got)
		}
		if e.calledPrefix("awg set") {
			t.Fatal("locked peer is not in the interface: no live call expected")
		}
		assertOnlyPeerOps(t, e, "awg", "awg0")
	})
	t.Run("not found", func(t *testing.T) {
		e := newEnv(t)
		e.mustRun(errs.NotFound, "--delete-client", "--interface", "awg0", "--id", "x")
		setupAWG0(t, e, true)
		e.mustRun(errs.NotFound, "--delete-client", "--interface", "awg0", "--id", "nobody")
		// Peer'ы без krot:id (показываются с id "-") по --id недоступны (SPEC 5.3).
		e.write(e.wgConf("office"), fixture(t, "foreign_wg.conf"))
		e.mustRun(errs.NotFound, "--delete-client", "--interface", "office", "--id", "-")
	})
	t.Run("live failure rolls back", func(t *testing.T) {
		e := newEnv(t)
		path := setupAWG0(t, e, true)
		orig := e.read(path)
		e.fake.Fail("awg set", 1, "Unable to modify interface")
		e.mustRun(errs.Internal, "--delete-client", "--interface", "awg0", "--id", "phone-sergei")
		if e.read(path) != orig {
			t.Fatal("config not rolled back")
		}
	})
}

// SPEC 6.7: --lock-client.
func TestLockClient(t *testing.T) {
	t.Run("interface up", func(t *testing.T) {
		e := newEnv(t)
		path := setupAWG0(t, e, true)
		orig := e.read(path)
		e.mustRun(0, "--lock-client", "--interface", "awg0", "--id", "phone-sergei")
		if got := e.read(path); got != expected(t, orig, func(c *conf.Config) error { return c.LockPeer("phone-sergei", lockTime) }) {
			t.Fatalf("config:\n%s", got)
		}
		if !e.called("awg set awg0 peer " + peerKey(t, orig, "phone-sergei") + " remove") {
			t.Fatalf("calls %q", e.fake.CommandLines())
		}
		assertOnlyPeerOps(t, e, "awg", "awg0")
	})
	t.Run("interface down", func(t *testing.T) {
		e := newEnv(t)
		path := setupAWG0(t, e, false)
		e.mustRun(0, "--lock-client", "--interface", "awg0", "--id", "phone-sergei")
		if p, _ := conf.Parse([]byte(e.read(path))).PeerByID("phone-sergei"); !p.Locked || p.LockedAt != lockTime {
			t.Fatalf("peer = %+v", p)
		}
		if e.calledPrefix("awg set") {
			t.Fatal("live call on a down interface")
		}
		assertOnlyPeerOps(t, e, "awg", "awg0")
	})
	t.Run("already locked", func(t *testing.T) {
		e := newEnv(t)
		path := setupAWG0(t, e, true)
		orig := e.read(path)
		e.mustRun(0, "--lock-client", "--interface", "awg0", "--id", "laptop-old")
		if e.read(path) != orig || e.calledPrefix("awg set") || !strings.Contains(e.stdout.String(), "already locked") {
			t.Fatalf("stdout %q calls %q", e.stdout, e.fake.CommandLines())
		}
		assertOnlyPeerOps(t, e, "awg", "awg0")
	})
	t.Run("not found", func(t *testing.T) {
		e := newEnv(t)
		setupAWG0(t, e, true)
		e.mustRun(errs.NotFound, "--lock-client", "--interface", "awg0", "--id", "nobody")
	})
	t.Run("live failure rolls back", func(t *testing.T) {
		e := newEnv(t)
		path := setupAWG0(t, e, true)
		orig := e.read(path)
		e.fake.Fail("awg set", 1, "Unable to modify interface")
		e.mustRun(errs.Internal, "--lock-client", "--interface", "awg0", "--id", "phone-sergei")
		if e.read(path) != orig {
			t.Fatal("config not rolled back")
		}
	})
	t.Run("wg tunnel", func(t *testing.T) {
		e := newEnv(t)
		path := e.wgConf("wg0")
		e.write(path, fixture(t, "wg0_empty.conf"))
		e.mustRun(0, "--add-client", "--interface", "wg0", "--id", "a", "--allowed-ips", "0.0.0.0/0")
		e.fake.On("wg show interfaces", "wg0\n")
		e.fake.Calls = nil
		e.mustRun(0, "--lock-client", "--interface", "wg0", "--id", "a")
		if !e.called("wg set wg0 peer " + peerKey(t, e.read(path), "a") + " remove") {
			t.Fatalf("calls %q", e.fake.CommandLines())
		}
		assertOnlyPeerOps(t, e, "wg", "wg0")
	})
}

// SPEC 6.8: --unlock-client.
func TestUnlockClient(t *testing.T) {
	t.Run("interface up", func(t *testing.T) {
		e := newEnv(t)
		path := setupAWG0(t, e, true)
		orig := e.read(path)
		e.mustRun(0, "--unlock-client", "--interface", "awg0", "--id", "laptop-old")
		if got := e.read(path); got != expected(t, orig, func(c *conf.Config) error { return c.UnlockPeer("laptop-old") }) {
			t.Fatalf("config:\n%s", got)
		}
		if !e.called("awg set awg0 peer " + peerKey(t, orig, "laptop-old") + " allowed-ips 10.8.0.3/32") {
			t.Fatalf("calls %q", e.fake.CommandLines())
		}
		assertOnlyPeerOps(t, e, "awg", "awg0")
	})
	t.Run("with PresharedKey", func(t *testing.T) {
		e := newEnv(t)
		path := setupAWG0(t, e, false)
		e.mustRun(0, "--lock-client", "--interface", "awg0", "--id", "phone-sergei")
		e.fake.On("awg show interfaces", "awg0\n")
		e.fake.Calls = nil
		cfg := conf.Parse([]byte(e.read(path)))
		p, _ := cfg.PeerByID("phone-sergei")
		var pskFile string
		e.fake.Handler = func(c execx.Call) *execx.Response {
			if i := slices.Index(c.Args, "preshared-key"); i >= 0 {
				pskFile = c.Args[i+1]
				st, err := os.Stat(pskFile)
				b, _ := os.ReadFile(pskFile)
				if err != nil || st.Mode().Perm() != 0o600 || strings.TrimSpace(string(b)) != p.PresharedKey() {
					t.Errorf("psk file: %v %v %q", st, err, b)
				}
			}
			return nil
		}
		e.mustRun(0, "--unlock-client", "--interface", "awg0", "--id", "phone-sergei")
		want := "awg set awg0 peer " + p.PublicKey() + " allowed-ips 10.8.0.2/32,fd00:8::2/128 preshared-key " + pskFile
		if pskFile == "" || !e.called(want) || e.exists(pskFile) {
			t.Fatalf("want %q (file removed), calls %q", want, e.fake.CommandLines())
		}
		if e.read(path) != fixture(t, "awg0_two_clients.conf") {
			t.Fatal("lock+unlock must restore the original config")
		}
		assertOnlyPeerOps(t, e, "awg", "awg0")
	})
	t.Run("already unlocked", func(t *testing.T) {
		e := newEnv(t)
		path := setupAWG0(t, e, true)
		orig := e.read(path)
		e.mustRun(0, "--unlock-client", "--interface", "awg0", "--id", "phone-sergei")
		if e.read(path) != orig || e.calledPrefix("awg set") || !strings.Contains(e.stdout.String(), "already unlocked") {
			t.Fatalf("stdout %q calls %q", e.stdout, e.fake.CommandLines())
		}
		assertOnlyPeerOps(t, e, "awg", "awg0")
	})
	t.Run("live failure rolls back", func(t *testing.T) {
		e := newEnv(t)
		path := setupAWG0(t, e, true)
		orig := e.read(path)
		e.fake.Fail("awg set", 1, "Unable to modify interface")
		e.mustRun(errs.Internal, "--unlock-client", "--interface", "awg0", "--id", "laptop-old")
		if e.read(path) != orig {
			t.Fatal("config not rolled back")
		}
	})
	t.Run("tool missing", func(t *testing.T) {
		e := newEnv(t)
		setupAWG0(t, e, true)
		e.fake.Missing = map[string]bool{"awg": true}
		e.mustRun(errs.Env, "--unlock-client", "--interface", "awg0", "--id", "laptop-old")
	})
}
