package portcheck

import (
	"context"
	"net"
	"testing"
	"time"
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

// SPEC 6.14: listener отвечает ping/pong, echo и time.
func TestListenerReplies(t *testing.T) {
	port := freeUDPPort(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := NewListener([]int{port})
	errCh := make(chan error, 1)
	go func() { errCh <- l.Listen(ctx) }()
	// Даём listener время забиндиться.
	time.Sleep(50 * time.Millisecond)

	addr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// ping
	if _, err := conn.Write(PingRequest(1)); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(time.Second))
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if seq, ok := ParsePong(buf[:n]); !ok || seq != 1 {
		t.Errorf("pong: seq=%d ok=%v; want 1 true", seq, ok)
	}

	// echo
	sent := "hello"
	if _, err := conn.Write(EchoRequest(sent)); err != nil {
		t.Fatal(err)
	}
	conn.SetReadDeadline(time.Now().Add(time.Second))
	n, err = conn.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if !IsEchoReply(buf[:n], sent) {
		t.Errorf("echo reply = %q, want %q", string(buf[:n]), sent)
	}

	// time
	before := time.Now().UnixMilli()
	if _, err := conn.Write(TimeRequest()); err != nil {
		t.Fatal(err)
	}
	conn.SetReadDeadline(time.Now().Add(time.Second))
	n, err = conn.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	ms, ok := ParseTimeReply(buf[:n])
	if !ok || ms < before || ms > before+1000 {
		t.Errorf("time reply = %d ok=%v", ms, ok)
	}

	cancel()
	select {
	case <-errCh:
	case <-time.After(time.Second):
		t.Fatal("listener did not stop")
	}
}

// SPEC 6.14: listener сообщает о клиентских ping, echo и time через OnRequest.
func TestListenerOnRequest(t *testing.T) {
	port := freeUDPPort(t)
	ctx, cancel := context.WithCancel(context.Background())

	type event struct {
		client  string
		port    int
		request string
	}
	events := make(chan event, 3)
	l := NewListener([]int{port})
	l.OnRequest = func(client *net.UDPAddr, localPort int, request string) {
		events <- event{client: client.String(), port: localPort, request: request}
	}
	listenerDone := make(chan error, 1)
	go func() { listenerDone <- l.Listen(ctx) }()
	defer func() {
		cancel()
		select {
		case <-listenerDone:
		case <-time.After(time.Second):
			t.Error("listener did not stop")
		}
	}()
	time.Sleep(50 * time.Millisecond)

	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: port})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	requests := []struct {
		data []byte
		kind string
	}{
		{PingRequest(7), "ping"},
		{EchoRequest("hello"), "echo"},
		{TimeRequest(), "time"},
	}
	for _, req := range requests {
		if _, err := conn.Write(req.data); err != nil {
			t.Fatal(err)
		}
		if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
			t.Fatal(err)
		}
		buf := make([]byte, 1024)
		if _, err := conn.Read(buf); err != nil {
			t.Fatal(err)
		}
		select {
		case got := <-events:
			if got.client != conn.LocalAddr().String() || got.port != port || got.request != req.kind {
				t.Errorf("event = %+v, want client %s, port %d, request %s", got, conn.LocalAddr(), port, req.kind)
			}
		case <-time.After(time.Second):
			t.Fatalf("no %s event", req.kind)
		}
	}
}

// SPEC 6.14: listener на диапазоне — отдельный сокет на каждый порт.
func TestListenerRange(t *testing.T) {
	port := freeUDPPort(t)
	// Используем port, port+1, port+2; freeUDPPort возвращает случайный свободный,
	// но последовательные порты могут быть заняты. Для надёжности проверим, что
	// port+1 и port+2 свободны, иначе пропускаем.
	ports := []int{port, port + 1, port + 2}
	for _, p := range ports[1:] {
		c, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: p})
		if err != nil {
			t.Skipf("port %d not free: %v", p, err)
		}
		c.Close()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	l := NewListener(ports)
	go l.Listen(ctx)
	time.Sleep(50 * time.Millisecond)

	for _, p := range ports {
		addr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: p}
		conn, err := net.DialUDP("udp", nil, addr)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := conn.Write(PingRequest(1)); err != nil {
			t.Fatal(err)
		}
		buf := make([]byte, 1024)
		conn.SetReadDeadline(time.Now().Add(time.Second))
		n, err := conn.Read(buf)
		conn.Close()
		if err != nil {
			t.Fatalf("port %d: %v", p, err)
		}
		if seq, ok := ParsePong(buf[:n]); !ok || seq != 1 {
			t.Errorf("port %d: pong seq=%d ok=%v", p, seq, ok)
		}
	}
}
