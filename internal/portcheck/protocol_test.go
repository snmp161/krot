package portcheck

import (
	"testing"
	"time"
)

// SPEC 6.14, 6.15: ping SEQ:N → pong SEQ:N.
func TestPingPong(t *testing.T) {
	for _, seq := range []int{1, 2, 100} {
		req := PingRequest(seq)
		if got, want := string(req), "ping SEQ:"+itoa(seq); got != want {
			t.Errorf("PingRequest(%d) = %q, want %q", seq, got, want)
		}
		reply := PongReply(seq)
		if got, want := string(reply), "pong SEQ:"+itoa(seq); got != want {
			t.Errorf("PongReply(%d) = %q, want %q", seq, got, want)
		}
		gotSeq, ok := ParsePong(reply)
		if !ok || gotSeq != seq {
			t.Errorf("ParsePong(%q) = %d, %v; want %d, true", string(reply), gotSeq, ok, seq)
		}
	}
	// Некорректные ответы.
	for _, bad := range [][]byte{
		[]byte("pong SEQ:"),
		[]byte("pong"),
		[]byte("pong SEQ:abc"),
		[]byte("pong SEQ:0"),
		[]byte("ping SEQ:1"),
	} {
		if _, ok := ParsePong(bad); ok {
			t.Errorf("ParsePong(%q) should fail", string(bad))
		}
	}
}

// SPEC 6.14, 6.15: ECHO <text> → <text>.
func TestEcho(t *testing.T) {
	text := "hello world"
	req := EchoRequest(text)
	if got, want := string(req), "ECHO "+text; got != want {
		t.Errorf("EchoRequest(%q) = %q, want %q", text, got, want)
	}
	if !IsEchoReply([]byte(text), text) {
		t.Errorf("IsEchoReply(%q, %q) = false, want true", text, text)
	}
	if IsEchoReply([]byte("hello"), text) {
		t.Error("IsEchoReply with mismatched text should be false")
	}
}

// SPEC 6.14, 6.15: TIME → unix-ms.
func TestTime(t *testing.T) {
	req := TimeRequest()
	if string(req) != "TIME" {
		t.Errorf("TimeRequest() = %q, want TIME", string(req))
	}
	now := time.Now().UnixMilli()
	reply := TimeReply(now)
	got, ok := ParseTimeReply(reply)
	if !ok || got != now {
		t.Errorf("ParseTimeReply(%q) = %d, %v; want %d, true", string(reply), got, ok, now)
	}
	if _, ok := ParseTimeReply([]byte("not-a-number")); ok {
		t.Error("ParseTimeReply should fail on non-number")
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
