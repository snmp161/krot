package app

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/snmp161/krot/internal/errs"
	"github.com/snmp161/krot/internal/portcheck"
)

// freeUDPPort возвращает свободный UDP-порт на 127.0.0.1.
func freeUDPPort(t *testing.T) int {
	t.Helper()
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).Port
}

// SPEC 2.4: --udp-listener и --udp-checker требуют root.
func TestUDPRequiresRoot(t *testing.T) {
	e := newEnv(t)
	e.app.Euid = func() int { return 1000 }
	e.mustRun(errs.Env, "--udp-listener", "--port", "51820")
	if !strings.Contains(e.stderr.String(), "root") {
		t.Errorf("udp-listener stderr = %q", e.stderr)
	}
	e.stderr.Reset()
	e.mustRun(errs.Env, "--udp-checker", "--address", "203.0.113.1", "--port", "51820")
	if !strings.Contains(e.stderr.String(), "root") {
		t.Errorf("udp-checker stderr = %q", e.stderr)
	}
}

// SPEC 6.14: --udp-listener не стартует, если порт занят (код 5).
func TestUDPListenerPortConflict(t *testing.T) {
	port := freeUDPPort(t)
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	e := newEnv(t)
	done := make(chan struct{})
	go func() {
		defer close(done)
		e.run("--udp-listener", "--port", itoa(port))
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("listener did not exit")
	}
	if got := errs.Code(e.app.Run([]string{"--udp-listener", "--port", itoa(port)})); got != errs.Conflict {
		t.Fatalf("exit code %d; want %d (conflict)", got, errs.Conflict)
	}
}

// SPEC 6.15: --udp-checker выводит таблицу результатов.
func TestUDPCheckerOutput(t *testing.T) {
	port := freeUDPPort(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := portcheck.NewListener([]int{port})
	go l.Listen(ctx)
	time.Sleep(50 * time.Millisecond)

	e := newEnv(t)
	e.mustRun(0, "--udp-checker", "--address", "127.0.0.1", "--port", itoa(port), "--timeout", "500ms")
	out := e.stdout.String()
	for _, s := range []string{itoa(port), "ok", "3/3"} {
		if !strings.Contains(out, s) {
			t.Fatalf("output misses %q:\n%s", s, out)
		}
	}
}

// SPEC 6.15: --udp-checker с --json.
func TestUDPCheckerJSON(t *testing.T) {
	port := freeUDPPort(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := portcheck.NewListener([]int{port})
	go l.Listen(ctx)
	time.Sleep(50 * time.Millisecond)

	e := newEnv(t)
	e.mustRun(0, "--udp-checker", "--address", "127.0.0.1", "--port", itoa(port), "--timeout", "500ms", "--json")
	out := e.stdout.String()
	if !strings.Contains(out, "\"port\"") || !strings.Contains(out, "\"status\"") {
		t.Fatalf("json output: %s", out)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
