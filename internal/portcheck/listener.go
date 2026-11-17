package portcheck

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// Listener слушает UDP-порты и отвечает на ping/echo/time.
type Listener struct {
	ports []int
	conns []*net.UDPConn
	wg    sync.WaitGroup
}

// NewListener создаёт listener для заданных портов.
func NewListener(ports []int) *Listener {
	return &Listener{ports: append([]int(nil), ports...)}
}

// Listen открывает сокет на каждый порт и слушает до отмены ctx.
func (l *Listener) Listen(ctx context.Context) error {
	if len(l.ports) == 0 {
		return fmt.Errorf("no ports to listen on")
	}
	for _, port := range l.ports {
		addr := &net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: port}
		conn, err := net.ListenUDP("udp", addr)
		if err != nil {
			l.Close()
			return fmt.Errorf("listen udp :%d: %w", port, err)
		}
		l.conns = append(l.conns, conn)
	}
	for _, conn := range l.conns {
		l.wg.Add(1)
		go l.serve(ctx, conn)
	}
	<-ctx.Done()
	l.Close()
	l.wg.Wait()
	return ctx.Err()
}

func (l *Listener) serve(ctx context.Context, conn *net.UDPConn) {
	defer l.wg.Done()
	buf := make([]byte, 2048)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		n, client, err := conn.ReadFromUDP(buf)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			return
		}
		reply := l.handle(buf[:n])
		if reply == nil {
			continue
		}
		_ = conn.SetWriteDeadline(time.Now().Add(time.Second))
		_, _ = conn.WriteToUDP(reply, client)
	}
}

func (l *Listener) handle(data []byte) []byte {
	s := string(data)
	switch {
	case s == "TIME":
		return TimeReply(time.Now().UnixMilli())
	case len(data) > len(echoPrefix) && s[:len(echoPrefix)] == echoPrefix:
		return data[len(echoPrefix):]
	default:
		if seq, ok := parseSeq("ping", data); ok {
			return PongReply(seq)
		}
	}
	return nil
}

// Close закрывает все сокеты.
func (l *Listener) Close() error {
	for _, conn := range l.conns {
		_ = conn.Close()
	}
	return nil
}
