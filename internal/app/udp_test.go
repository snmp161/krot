package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/snmp161/krot/internal/cli"
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

// SPEC 2.4: --udp-checker работает без root.
func TestUDPCheckerWithoutRoot(t *testing.T) {
	port := freeUDPPort(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l := portcheck.NewListener([]int{port})
	go l.Listen(ctx)
	time.Sleep(50 * time.Millisecond)

	e := newEnv(t)
	e.app.Euid = func() int { return 1000 }
	e.mustRun(0, "--udp-checker", "--address", "127.0.0.1", "--port", itoa(port), "--timeout", "500ms")
	if !strings.Contains(e.stdout.String(), "ok") {
		t.Fatalf("stdout:\n%s", e.stdout)
	}
}

// SPEC 2.4, 6.14: --udp-listener без root разрешён для непривилегированных портов
// при доступных /proc/net/udp{,6}.
func TestUDPListenerWithoutRoot(t *testing.T) {
	port := freeUDPPort(t)
	e := newEnv(t)
	e.app.Euid = func() int { return 1000 }

	done := make(chan struct{})
	go func() {
		defer close(done)
		e.run("--udp-listener", "--port", itoa(port))
	}()
	select {
	case <-done:
		// listener не должен был завершиться сам; если завершился — проверим код.
		if code := e.run("--udp-listener", "--port", itoa(port)); code != 0 {
			t.Fatalf("listener exited with code %d; stdout:\n%s\nstderr:\n%s", code, e.stdout, e.stderr)
		}
	case <-time.After(200 * time.Millisecond):
		// listener запустился и работает — это ожидаемо.
	}
}

// SPEC 2.4, 6.14: --udp-listener требует root для привилегированного порта.
func TestUDPListenerPrivilegedPortRequiresRoot(t *testing.T) {
	e := newEnv(t)
	e.app.Euid = func() int { return 1000 }
	e.mustRun(errs.Env, "--udp-listener", "--port", "1023")
	if !strings.Contains(e.stderr.String(), "root") {
		t.Errorf("stderr = %q", e.stderr)
	}
}

// SPEC 2.4, 6.14: --udp-listener требует root, если /proc/net/udp недоступен.
func TestUDPListenerProcNetRequiresRoot(t *testing.T) {
	port := freeUDPPort(t)
	e := newEnv(t)
	e.app.Euid = func() int { return 1000 }
	// Удаляем фейковые файлы, чтобы имитировать недоступность /proc/net/udp{,6}.
	e.app.Paths.ProcNet = t.TempDir() + "/nonexistent"

	e.mustRun(errs.Env, "--udp-listener", "--port", itoa(port))
	if !strings.Contains(e.stderr.String(), "root") {
		t.Errorf("stderr = %q", e.stderr)
	}
}

// SPEC 6.14: --udp-listener не стартует, если порт занят (код 5).
func TestUDPListenerPortConflict(t *testing.T) {
	port := freeUDPPort(t)
	e := newEnv(t)
	// Помечаем порт занятым в фейковом /proc/net/udp, чтобы checkUDPPortsFree
	// сработала без root и без реального слушающего процесса.
	hexPort := fmt.Sprintf("%04X", port)
	e.write(e.app.Paths.ProcNet+"/udp", udpHdr+
		"  123: 00000000:"+hexPort+" 00000000:0000 07 00000000:00000000 00:00000000 00000000     0        0 4242 2 0000000000000000 0\n")

	if got := e.run("--udp-listener", "--port", itoa(port)); got != errs.Conflict {
		t.Fatalf("exit code %d; want %d (conflict)", got, errs.Conflict)
	}
}

// SPEC 6.14: стартовое сообщение выводится только после успешного открытия сокетов.
func TestUDPListenerStartMessageAfterBind(t *testing.T) {
	for _, mode := range []struct {
		name string
		json bool
	}{{"text", false}, {"json", true}} {
		t.Run(mode.name, func(t *testing.T) {
			conn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			port := conn.LocalAddr().(*net.UDPAddr).Port

			args := []string{"--udp-listener", "--port", itoa(port)}
			if mode.json {
				args = append(args, "--json")
			}
			e := newEnv(t)
			if got := e.run(args...); got != errs.Internal {
				t.Fatalf("exit code %d; want %d", got, errs.Internal)
			}
			if got := e.stdout.String(); got != "" {
				t.Fatalf("startup output after bind error: %q", got)
			}
		})
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

// SPEC 6.15: --verbose сообщает диапазон и промежуточные результаты в stderr,
// не смешивая их с JSON в stdout; --workers задаёт число работников.
func TestUDPCheckerVerbose(t *testing.T) {
	port := freeUDPPort(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := portcheck.NewListener([]int{port})
	go l.Listen(ctx)
	time.Sleep(50 * time.Millisecond)

	e := newEnv(t)
	e.mustRun(0, "--udp-checker", "--address", "127.0.0.1", "--port", itoa(port),
		"--timeout", "500ms", "--count", "1", "--workers", "2", "--verbose", "--json")
	var results []map[string]any
	if err := json.Unmarshal(e.stdout.Bytes(), &results); err != nil {
		t.Fatalf("stdout is not clean JSON: %v\n%s", err, e.stdout.Bytes())
	}
	if len(results) != 1 || results[0]["status"] != "ok" {
		t.Fatalf("JSON results: %+v", results)
	}
	for _, want := range []string{
		"Checking UDP ports " + itoa(port) + " on 127.0.0.1 with 2 workers",
		"UDP port " + itoa(port) + ": ok",
	} {
		if !strings.Contains(e.stderr.String(), want) {
			t.Errorf("stderr misses %q:\n%s", want, e.stderr.String())
		}
	}
}

// SPEC 6.15: отменённый checker печатает предварительные результаты и возвращает код 10.
func TestUDPCheckerCancellationPrintsResults(t *testing.T) {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	port := conn.LocalAddr().(*net.UDPAddr).Port
	started := make(chan struct{})
	go func() {
		buf := make([]byte, 2048)
		_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		if _, _, err := conn.ReadFromUDP(buf); err == nil {
			close(started)
		}
	}()

	cmd, err := cli.Parse([]string{"--udp-checker", "--address", "127.0.0.1", "--port", itoa(port),
		"--timeout", "5s", "--count", "1", "--workers", "1"})
	if err != nil {
		t.Fatal(err)
	}
	e := newEnv(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	checkDone := make(chan error, 1)
	go func() { checkDone <- e.app.runUDPChecker(ctx, cmd) }()
	select {
	case <-started:
		cancel()
	case <-time.After(time.Second):
		t.Fatal("checker did not send a request")
	}
	select {
	case err := <-checkDone:
		if got := errs.Code(err); got != errs.Cancelled {
			t.Fatalf("exit code = %d, want %d; error: %v", got, errs.Cancelled, err)
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled checker did not finish")
	}
	if !strings.Contains(e.stdout.String(), "PORT") {
		t.Fatalf("partial table was not printed:\n%s", e.stdout.String())
	}
	if strings.Contains(e.stdout.String(), itoa(port)) {
		t.Fatalf("unfinished port %d appeared in partial results:\n%s", port, e.stdout.String())
	}
}

// SPEC 6.15: при отмене до первого результата JSON остаётся валидным пустым массивом.
func TestUDPCheckerCancellationJSON(t *testing.T) {
	cmd, err := cli.Parse([]string{"--udp-checker", "--address", "127.0.0.1", "--port", "51820", "--json"})
	if err != nil {
		t.Fatal(err)
	}
	e := newEnv(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = e.app.runUDPChecker(ctx, cmd)
	if got := errs.Code(err); got != errs.Cancelled {
		t.Fatalf("exit code = %d, want %d; error: %v", got, errs.Cancelled, err)
	}
	var results []map[string]any
	if err := json.Unmarshal(e.stdout.Bytes(), &results); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, e.stdout.Bytes())
	}
	if len(results) != 0 {
		t.Fatalf("partial results = %+v, want empty", results)
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

// SPEC 6.14: --udp-listener --json при старте выводит одну строку валидного JSON
// {"port_start":N,"port_end":M,"status":"listening"}; для одного порта N = M.
func TestUDPListenerJSONStart(t *testing.T) {
	for _, span := range []int{0, 2} {
		t.Run(fmt.Sprintf("span%d", span), func(t *testing.T) {
			start := freeUDPPort(t)
			ports := itoa(start)
			if span > 0 {
				ports += ":" + itoa(start+span)
			}
			cmd, err := cli.Parse([]string{"--udp-listener", "--port", ports, "--json"})
			if err != nil {
				t.Fatal(err)
			}
			e := newEnv(t)
			ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
			defer cancel()
			_ = e.app.runUDPListener(ctx, cmd) // занятый соседний порт не влияет на стартовую строку

			lines := strings.Split(strings.TrimRight(e.stdout.String(), "\n"), "\n")
			if len(lines) != 1 {
				t.Fatalf("want exactly one JSON line, got:\n%s", e.stdout)
			}
			var got map[string]any
			if err := json.Unmarshal([]byte(lines[0]), &got); err != nil {
				t.Fatalf("start line is not valid JSON: %v\n%s", err, lines[0])
			}
			want := map[string]any{"port_start": float64(start), "port_end": float64(start + span), "status": "listening"}
			if fmt.Sprint(got) != fmt.Sprint(want) {
				t.Fatalf("start line = %v; want %v", got, want)
			}
		})
	}
}
