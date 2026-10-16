package execx

import (
	"errors"
	"slices"
	"sync"
	"testing"
)

// SPEC 8.2: реальный исполнитель — stdout, stderr, код выхода.
func TestOS(t *testing.T) {
	var r Runner = OS{}
	res, err := r.Run("echo", "hello", "world")
	if err != nil || string(res.Stdout) != "hello world\n" || res.ExitCode != 0 {
		t.Fatalf("echo: %+v, %v", res, err)
	}
	res, err = r.Run("sh", "-c", "echo out; echo err >&2; exit 3")
	if err == nil || res.ExitCode != 3 || string(res.Stdout) != "out\n" || string(res.Stderr) != "err\n" {
		t.Fatalf("exit 3: %+v, %v", res, err)
	}
	if res.Message() != "err" {
		t.Fatalf("Message() = %q", res.Message())
	}
	res, err = r.Run("false")
	if err == nil || res.ExitCode != 1 {
		t.Fatalf("false: %+v, %v", res, err)
	}
	res, err = r.Run("definitely-missing-program-krot")
	if err == nil || res.ExitCode != -1 {
		t.Fatalf("missing: %+v, %v", res, err)
	}
	// Вывод утилит разбирается, поэтому локаль фиксирована.
	res, _ = r.Run("sh", "-c", "echo $LC_ALL")
	if string(res.Stdout) != "C\n" {
		t.Fatalf("LC_ALL = %q", res.Stdout)
	}
	if _, err := r.LookPath("sh"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.LookPath("definitely-missing-program-krot"); err == nil {
		t.Fatal("LookPath must fail for a missing program")
	}
}

func TestResultMessage(t *testing.T) {
	if m := (Result{Stdout: []byte(" out \n")}).Message(); m != "out" {
		t.Fatalf("Message() = %q", m)
	}
	if m := (Result{Stdout: []byte("out"), Stderr: []byte("err\n")}).Message(); m != "err" {
		t.Fatalf("Message() = %q", m)
	}
}

// SPEC 8.2: фейк исполнителя с записью вызовов.
func TestFake(t *testing.T) {
	f := &Fake{}
	f.On("wg show interfaces", "wg0 wg1\n")
	f.Fail("systemctl start", 1, "Job failed")
	f.On("systemctl start wg-quick@wg9", "")
	f.Missing = map[string]bool{"awg": true}

	var r Runner = f
	res, err := r.Run("wg", "show", "interfaces")
	if err != nil || string(res.Stdout) != "wg0 wg1\n" {
		t.Fatalf("wg show: %+v, %v", res, err)
	}
	res, err = r.Run("systemctl", "start", "wg-quick@wg0")
	if err == nil || res.ExitCode != 1 || res.Message() != "Job failed" {
		t.Fatalf("systemctl start: %+v, %v", res, err)
	}
	// Более позднее правило важнее.
	if _, err := r.Run("systemctl", "start", "wg-quick@wg9"); err != nil {
		t.Fatalf("override: %v", err)
	}
	// Префикс сравнивается по словам.
	if _, err := r.Run("systemctl", "starts"); err != nil {
		t.Fatalf("word prefix: %v", err)
	}
	// По умолчанию — успех с пустым выводом.
	if res, err := r.Run("ip", "link", "show"); err != nil || len(res.Stdout) != 0 {
		t.Fatalf("default: %+v, %v", res, err)
	}
	want := []string{
		"wg show interfaces",
		"systemctl start wg-quick@wg0",
		"systemctl start wg-quick@wg9",
		"systemctl starts",
		"ip link show",
	}
	if got := f.CommandLines(); !slices.Equal(got, want) {
		t.Fatalf("calls = %q", got)
	}
	if p, err := r.LookPath("wg"); err != nil || p == "" {
		t.Fatalf("LookPath(wg) = %q, %v", p, err)
	}
	if _, err := r.LookPath("awg"); err == nil {
		t.Fatal("LookPath(awg) must fail")
	}
}

func TestFakeHandler(t *testing.T) {
	boom := errors.New("boom")
	f := &Fake{Handler: func(c Call) *Response {
		if c.Name == "awg" {
			return &Response{Result: Result{ExitCode: 2}, Err: boom}
		}
		return nil
	}}
	f.On("wg show", "x")
	if _, err := f.Run("awg", "set", "awg0"); !errors.Is(err, boom) {
		t.Fatalf("handler: %v", err)
	}
	if res, err := f.Run("wg", "show"); err != nil || string(res.Stdout) != "x" {
		t.Fatalf("fallthrough to rules: %+v, %v", res, err)
	}
	if got := f.Calls[0]; got.Name != "awg" || !slices.Equal(got.Args, []string{"set", "awg0"}) {
		t.Fatalf("call = %+v", got)
	}
}

func TestFakeConcurrent(t *testing.T) {
	f := &Fake{}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f.Run("wg", "show")
		}()
	}
	wg.Wait()
	if len(f.CommandLines()) != 50 {
		t.Fatalf("got %d calls", len(f.CommandLines()))
	}
}
