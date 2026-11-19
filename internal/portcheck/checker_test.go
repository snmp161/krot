package portcheck

import (
	"bytes"
	"context"
	"fmt"
	"math/rand/v2"
	"net"
	"sync"
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

// SPEC 6.15: checker параллельно проверяет разные порты и сохраняет их порядок.
func TestCheckerChecksPortsInParallel(t *testing.T) {
	const portCount = 2

	conns := make([]*net.UDPConn, portCount)
	ports := make([]int, portCount)
	for i := range conns {
		conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
		if err != nil {
			for j := 0; j < i; j++ {
				_ = conns[j].Close()
			}
			t.Fatal(err)
		}
		conns[i] = conn
		ports[i] = conn.LocalAddr().(*net.UDPAddr).Port
	}

	arrived := make(chan struct{}, portCount)
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseServers := func() { releaseOnce.Do(func() { close(release) }) }
	defer releaseServers()

	serverDone := make(chan error, portCount)
	for _, conn := range conns {
		go func() {
			serverDone <- serveBarrierUDP(conn, arrived, release)
		}()
	}

	wantOrder := []int{ports[1], ports[0]}
	c := &Checker{
		Address:  "127.0.0.1",
		Ports:    wantOrder,
		Timeout:  3 * time.Second,
		Count:    1,
		EchoSize: 64,
		Rand:     seededReader(2),
	}
	type outcome struct {
		results []Result
		err     error
	}
	checkDone := make(chan outcome, 1)
	go func() {
		results, err := c.Check(context.Background())
		checkDone <- outcome{results: results, err: err}
	}()

	concurrent := true
	timer := time.NewTimer(2 * time.Second)
waitForServers:
	for range portCount {
		select {
		case <-arrived:
		case <-timer.C:
			concurrent = false
			break waitForServers
		}
	}
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	releaseServers()

	var got outcome
	select {
	case got = <-checkDone:
	case <-time.After(5 * time.Second):
		t.Fatal("checker did not finish")
	}
	if got.err != nil {
		t.Fatal(got.err)
	}
	for range portCount {
		if err := <-serverDone; err != nil {
			t.Errorf("test UDP server: %v", err)
		}
	}
	if !concurrent {
		t.Fatal("ports were checked sequentially")
	}
	if len(got.results) != portCount {
		t.Fatalf("len(results) = %d, want %d", len(got.results), portCount)
	}
	for i, r := range got.results {
		if r.Port != wantOrder[i] {
			t.Errorf("results[%d].Port = %d, want %d", i, r.Port, wantOrder[i])
		}
		if r.Status != "ok" || r.PingOK != 1 || r.EchoOK != 1 || r.TimeOK != 1 {
			t.Errorf("port %d result = %+v, want all checks ok", r.Port, r)
		}
	}
}

// SPEC 6.14, 6.15: параллельный checker совместим с listener на диапазоне портов.
func TestCheckerParallelRangeWithListener(t *testing.T) {
	const portCount = 4
	ports := freeUDPPorts(t, portCount)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := NewListener(ports)
	listenerDone := make(chan error, 1)
	go func() { listenerDone <- l.Listen(ctx) }()
	time.Sleep(50 * time.Millisecond)

	wantOrder := []int{ports[3], ports[1], ports[2], ports[0]}
	c := &Checker{
		Address:  "127.0.0.1",
		Ports:    wantOrder,
		Timeout:  time.Second,
		Count:    2,
		EchoSize: 64,
		Rand:     seededReader(3),
	}
	results, err := c.Check(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != portCount {
		t.Fatalf("len(results) = %d, want %d", len(results), portCount)
	}
	for i, r := range results {
		if r.Port != wantOrder[i] {
			t.Errorf("results[%d].Port = %d, want %d", i, r.Port, wantOrder[i])
		}
		if r.Status != "ok" || r.PingOK != 2 || r.EchoOK != 2 || r.TimeOK != 2 {
			t.Errorf("port %d result = %+v, want all checks ok", r.Port, r)
		}
	}

	cancel()
	select {
	case <-listenerDone:
	case <-time.After(time.Second):
		t.Fatal("listener did not stop")
	}
}

func serveBarrierUDP(conn *net.UDPConn, arrived chan<- struct{}, release <-chan struct{}) error {
	defer conn.Close()
	l := &Listener{}
	buf := make([]byte, 2048)
	for i := 0; i < 3; i++ {
		if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
			return err
		}
		n, client, err := conn.ReadFromUDP(buf)
		if err != nil {
			return err
		}
		if i == 0 {
			arrived <- struct{}{}
			<-release
		}
		reply := l.handle(buf[:n])
		if reply == nil {
			return fmt.Errorf("unexpected request %q", buf[:n])
		}
		if err := conn.SetWriteDeadline(time.Now().Add(time.Second)); err != nil {
			return err
		}
		if _, err := conn.WriteToUDP(reply, client); err != nil {
			return err
		}
	}
	return nil
}

func freeUDPPorts(t *testing.T, count int) []int {
	t.Helper()
	conns := make([]*net.UDPConn, count)
	defer func() {
		for _, conn := range conns {
			if conn != nil {
				_ = conn.Close()
			}
		}
	}()

	ports := make([]int, count)
	for i := range conns {
		conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
		if err != nil {
			t.Fatal(err)
		}
		conns[i] = conn
		ports[i] = conn.LocalAddr().(*net.UDPAddr).Port
	}
	return ports
}
