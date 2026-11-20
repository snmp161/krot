package app

import (
	"strings"
	"testing"

	"github.com/snmp161/krot/internal/cli"
	"github.com/snmp161/krot/internal/conf"
	"github.com/snmp161/krot/internal/errs"
)

// SPEC 6.3 T37: после flock состав клиентов сравнивается с подтверждённым snapshot.
func TestDeleteRechecksClientsAfterLock(t *testing.T) {
	add := func(id, key string) func(*env, string) {
		return func(e *env, path string) {
			c := conf.Parse([]byte(e.read(path)))
			c.AddPeer(conf.NewPeer{ID: id, PublicKey: key, AllowedIPs: []string{"10.8.0.99/32"}})
			e.write(path, string(c.Bytes()))
		}
	}
	remove := func(id string) func(*env, string) {
		return func(e *env, path string) {
			c := conf.Parse([]byte(e.read(path)))
			if err := c.RemovePeer(id); err != nil {
				t.Fatal(err)
			}
			e.write(path, string(c.Bytes()))
		}
	}
	replace := func(e *env, path string) {
		remove("phone-sergei")(e, path)
		add("replacement", "replacement-key")(e, path)
	}

	tests := []struct {
		name         string
		fixtureName  string
		mutate       func(*env, string)
		answers      string
		tty          bool
		yes          bool
		wantCode     int
		wantPrompts  int
		wantDeleted  bool
		wantThree    bool
		wantNoTTYEnd bool
	}{
		{name: "unchanged needs one prompt", fixtureName: "awg0_two_clients.conf", answers: "y\n", tty: true, wantCode: 0, wantPrompts: 1, wantDeleted: true},
		{name: "added client needs reconfirmation", fixtureName: "awg0_two_clients.conf", mutate: add("new", "new-key"), answers: "y\ny\n", tty: true, wantCode: 0, wantPrompts: 2, wantDeleted: true, wantThree: true},
		{name: "replacement at same count needs reconfirmation", fixtureName: "awg0_two_clients.conf", mutate: replace, answers: "y\ny\n", tty: true, wantCode: 0, wantPrompts: 2, wantDeleted: true},
		{name: "client added to empty config", fixtureName: "wg0_empty.conf", mutate: add("new", "new-key"), answers: "y\n", tty: true, wantCode: 0, wantPrompts: 1, wantDeleted: true},
		{name: "all clients removed", fixtureName: "awg0_two_clients.conf", mutate: func(e *env, path string) {
			remove("phone-sergei")(e, path)
			remove("laptop-old")(e, path)
		}, answers: "y\n", tty: true, wantCode: 0, wantPrompts: 1, wantDeleted: true},
		{name: "second answer no keeps config", fixtureName: "awg0_two_clients.conf", mutate: add("new", "new-key"), answers: "y\nn\n", tty: true, wantCode: errs.Cancelled, wantPrompts: 2},
		{name: "second prompt requires tty", fixtureName: "awg0_two_clients.conf", mutate: func(e *env, path string) {
			add("new", "new-key")(e, path)
			e.tty = false
		}, answers: "y\n", tty: true, wantCode: errs.Usage, wantPrompts: 1, wantNoTTYEnd: true},
		{name: "yes skips both prompts", fixtureName: "awg0_two_clients.conf", mutate: add("new", "new-key"), yes: true, wantCode: 0, wantDeleted: true},
		{name: "foreign peers are included", fixtureName: "foreign_wg.conf", mutate: add("new", "new-key"), answers: "y\ny\n", tty: true, wantCode: 0, wantPrompts: 2, wantDeleted: true, wantThree: true},
		{name: "peer order alone is not a change", fixtureName: "awg0_two_clients.conf", mutate: func(e *env, path string) {
			remove("phone-sergei")(e, path)
			add("phone-sergei", "9MzuHlmhRvZCWO2jCp/sKB2SzR32zHPCqqxYSgfOpn4=")(e, path)
		}, answers: "y\n", tty: true, wantCode: 0, wantPrompts: 1, wantDeleted: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			path := e.awgConf("awg0")
			e.write(path, fixture(t, tt.fixtureName))
			e.tty = tt.tty
			e.app.Stdin = strings.NewReader(tt.answers)
			runLock := func(fn func() error) error {
				if tt.mutate != nil {
					tt.mutate(e, path)
				}
				return e.app.withLock(fn)
			}
			err := e.app.deleteTunnelWithLock(&cli.Command{Interface: "awg0", Yes: tt.yes}, runLock)
			if got := errs.Code(err); got != tt.wantCode {
				t.Fatalf("exit = %d, want %d; stderr=%q", got, tt.wantCode, e.stderr.String())
			}
			if got := strings.Count(e.stderr.String(), "Delete?"); got != tt.wantPrompts {
				t.Errorf("prompts = %d, want %d; stderr=%q", got, tt.wantPrompts, e.stderr.String())
			}
			if tt.wantThree && !strings.Contains(e.stderr.String(), "Interface awg0 has 3 clients") {
				t.Errorf("second prompt does not use current count: %q", e.stderr.String())
			}
			if tt.wantNoTTYEnd && (err == nil || !strings.Contains(err.Error(), "stdin is not a terminal")) {
				t.Errorf("missing non-TTY error: %v", err)
			}
			if got := !e.exists(path); got != tt.wantDeleted {
				t.Errorf("deleted = %v, want %v", got, tt.wantDeleted)
			}
			if got := e.calledPrefix("systemctl"); got != tt.wantDeleted {
				t.Errorf("systemctl called = %v, want %v; calls=%q", got, tt.wantDeleted, e.fake.CommandLines())
			}
		})
	}
}
