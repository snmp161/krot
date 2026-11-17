package portcheck

import (
	"context"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"net"
	"slices"
	"time"
)

// Result — результат проверки одного порта.
type Result struct {
	Port                   int
	Status                 string
	PingSent, PingOK       int
	EchoSent, EchoOK       int
	TimeSent, TimeOK       int
	RTTMin, RTTAvg, RTTMax time.Duration
	Detail                 string
}

// Checker проверяет UDP-порты удалённого listener.
type Checker struct {
	Address  string
	Ports    []int
	Timeout  time.Duration
	Count    int
	EchoSize int
	Rand     io.Reader
}

// Check проверяет все порты и возвращает результаты.
func (c *Checker) Check(ctx context.Context) ([]Result, error) {
	if c.Timeout <= 0 {
		c.Timeout = 2 * time.Second
	}
	if c.Count <= 0 {
		c.Count = 3
	}
	if c.EchoSize <= 0 {
		c.EchoSize = 64
	}
	rng := c.rand()

	results := make([]Result, len(c.Ports))
	for i, port := range c.Ports {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		results[i] = c.checkPort(ctx, rng, port)
	}
	return results, nil
}

func (c *Checker) rand() *rand.Rand {
	if c.Rand != nil {
		var seed [16]byte
		_, _ = io.ReadFull(c.Rand, seed[:])
		return rand.New(rand.NewPCG(binaryLittleEndianUint64(seed[0:8]), binaryLittleEndianUint64(seed[8:16])))
	}
	return rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 0))
}

func binaryLittleEndianUint64(b []byte) uint64 {
	return uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
		uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56
}

func (c *Checker) checkPort(ctx context.Context, rng *rand.Rand, port int) Result {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", c.Address, port))
	if err != nil {
		return Result{Port: port, Status: "error", Detail: err.Error()}
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return Result{Port: port, Status: "error", Detail: err.Error()}
	}
	defer conn.Close()

	r := Result{Port: port}
	var rtts []time.Duration

	for seq := 1; seq <= c.Count; seq++ {
		// ping
		r.PingSent++
		if c.exchange(ctx, conn, PingRequest(seq), func(data []byte) bool {
			s, ok := ParsePong(data)
			return ok && s == seq
		}) {
			r.PingOK++
		}
		// echo
		r.EchoSent++
		text := randomText(rng, c.EchoSize)
		if c.exchange(ctx, conn, EchoRequest(text), func(data []byte) bool {
			return IsEchoReply(data, text)
		}) {
			r.EchoOK++
		}
		// time
		r.TimeSent++
		if d, ok := c.exchangeRTT(ctx, conn, TimeRequest(), func(data []byte) bool {
			_, ok := ParseTimeReply(data)
			return ok
		}); ok {
			r.TimeOK++
			rtts = append(rtts, d)
		}
	}

	r.Status = "ok"
	if r.PingOK != r.PingSent || r.EchoOK != r.EchoSent {
		r.Status = "filtered"
		r.Detail = "timeout"
	}
	if len(rtts) > 0 {
		slices.Sort(rtts)
		r.RTTMin = rtts[0]
		r.RTTMax = rtts[len(rtts)-1]
		var sum time.Duration
		for _, d := range rtts {
			sum += d
		}
		r.RTTAvg = sum / time.Duration(len(rtts))
	}
	return r
}

func (c *Checker) exchange(ctx context.Context, conn *net.UDPConn, req []byte, check func([]byte) bool) bool {
	_, ok := c.exchangeRTT(ctx, conn, req, check)
	return ok
}

func (c *Checker) exchangeRTT(ctx context.Context, conn *net.UDPConn, req []byte, check func([]byte) bool) (time.Duration, bool) {
	_ = conn.SetWriteDeadline(time.Now().Add(c.Timeout))
	if _, err := conn.Write(req); err != nil {
		return 0, false
	}
	buf := make([]byte, 2048)
	start := time.Now()
	_ = conn.SetReadDeadline(time.Now().Add(c.Timeout))
	n, err := conn.Read(buf)
	if err != nil {
		return 0, false
	}
	rtt := time.Since(start)
	if !check(buf[:n]) {
		return 0, false
	}
	return rtt, true
}

func randomText(rng *rand.Rand, size int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, size)
	for i := range b {
		b[i] = chars[rng.IntN(len(chars))]
	}
	return string(b)
}

// FormatRTT возвращает строку с min/avg/max RTT в миллисекундах.
func FormatRTT(r Result) string {
	if r.RTTAvg == 0 {
		return "-"
	}
	return fmt.Sprintf("%d/%d/%d ms",
		int64(math.Round(float64(r.RTTMin.Nanoseconds())/1e6)),
		int64(math.Round(float64(r.RTTAvg.Nanoseconds())/1e6)),
		int64(math.Round(float64(r.RTTMax.Nanoseconds())/1e6)))
}
