package portcheck

import (
	"bytes"
	"context"
	"math/rand/v2"
	"testing"
	"time"
)

// seededReader возвращает детерминированный io.Reader для тестов.
func seededReader(seed uint64) *bytes.Reader {
	var s [32]byte
	for i := 0; i < 8; i++ {
		s[i] = byte(seed >> (8 * i))
	}
	r := rand.NewChaCha8(s)
	// Генерируем достаточно байт для random-text.
	data := make([]byte, 4096)
	_, _ = r.Read(data)
	return bytes.NewReader(data)
}

// SPEC 6.15: checker помечает доступный порт ok и считает RTT.
func TestCheckerOK(t *testing.T) {
	port := freeUDPPort(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := NewListener([]int{port})
	go l.Listen(ctx)
	time.Sleep(50 * time.Millisecond)

	c := &Checker{
		Address:  "127.0.0.1",
		Ports:    []int{port},
		Timeout:  time.Second,
		Count:    3,
		EchoSize: 64,
		Rand:     seededReader(1),
	}
	results, err := c.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	r := results[0]
	if r.Port != port {
		t.Errorf("port = %d, want %d", r.Port, port)
	}
	if r.Status != "ok" {
		t.Errorf("status = %q, want ok", r.Status)
	}
	if r.PingSent != 3 || r.PingOK != 3 {
		t.Errorf("ping = %d/%d, want 3/3", r.PingOK, r.PingSent)
	}
	if r.EchoSent != 3 || r.EchoOK != 3 {
		t.Errorf("echo = %d/%d, want 3/3", r.EchoOK, r.EchoSent)
	}
	if r.TimeSent != 3 || r.TimeOK != 3 {
		t.Errorf("time = %d/%d, want 3/3", r.TimeOK, r.TimeSent)
	}
	if r.RTTAvg <= 0 || r.RTTMin > r.RTTAvg || r.RTTAvg > r.RTTMax {
		t.Errorf("rtt min/avg/max = %v/%v/%v", r.RTTMin, r.RTTAvg, r.RTTMax)
	}
}

// SPEC 6.15: checker помечает недоступный порт filtered.
func TestCheckerFiltered(t *testing.T) {
	port := freeUDPPort(t)
	c := &Checker{
		Address:  "127.0.0.1",
		Ports:    []int{port},
		Timeout:  100 * time.Millisecond,
		Count:    2,
		EchoSize: 64,
		Rand:     seededReader(1),
	}
	results, err := c.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	r := results[0]
	if r.Status != "filtered" {
		t.Errorf("status = %q, want filtered", r.Status)
	}
	if r.PingOK != 0 || r.EchoOK != 0 || r.TimeOK != 0 {
		t.Errorf("ok counts should be zero: %+v", r)
	}
}
