package portcheck

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"math"
	"net"
	"slices"
	"sync"
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

const (
	// DefaultWorkers — число одновременно проверяемых портов по умолчанию.
	DefaultWorkers = 32
	// MaxWorkers — максимальное число одновременно проверяемых портов.
	MaxWorkers = 100
)

// Checker проверяет UDP-порты удалённого listener.
type Checker struct {
	Address  string
	Ports    []int
	Timeout  time.Duration
	Count    int
	EchoSize int
	Workers  int
	OnResult func(Result)
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
	if c.Workers <= 0 {
		c.Workers = DefaultWorkers
	}
	rng := c.Rand
	if rng == nil {
		rng = rand.Reader
	}

	results := make([]Result, len(c.Ports))
	completed := make([]bool, len(c.Ports))
	if len(c.Ports) == 0 {
		return results, nil
	}
	rng = &lockedReader{reader: rng}

	jobs := make(chan int)
	workers := min(c.Workers, MaxWorkers, len(c.Ports))
	var wg sync.WaitGroup
	var resultMu sync.Mutex
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			for i := range jobs {
				result, err := c.checkPort(ctx, rng, c.Ports[i])
				if err != nil {
					continue
				}
				results[i] = result
				completed[i] = true
				if c.OnResult != nil {
					resultMu.Lock()
					c.OnResult(result)
					resultMu.Unlock()
				}
			}
		}()
	}

sendJobs:
	for i := range c.Ports {
		select {
		case <-ctx.Done():
			break sendJobs
		case jobs <- i:
		}
	}
	close(jobs)
	wg.Wait()
	if err := ctx.Err(); err != nil {
		partial := make([]Result, 0, len(c.Ports))
		for i, result := range results {
			if completed[i] {
				partial = append(partial, result)
			}
		}
		return partial, err
	}
	return results, nil
}

func (c *Checker) checkPort(ctx context.Context, rng io.Reader, port int) (Result, error) {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", c.Address, port))
	if err != nil {
		if ctx.Err() != nil {
			return Result{Port: port}, ctx.Err()
		}
		return Result{Port: port, Status: "error", Detail: err.Error()}, nil
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		if ctx.Err() != nil {
			return Result{Port: port}, ctx.Err()
		}
		return Result{Port: port, Status: "error", Detail: err.Error()}, nil
	}
	defer conn.Close()
	stopClose := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stopClose()

	r := Result{Port: port}
	var rtts []time.Duration

	for seq := 1; seq <= c.Count; seq++ {
		if err := ctx.Err(); err != nil {
			return r, err
		}
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
	if err := ctx.Err(); err != nil {
		return r, err
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
	return r, nil
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

type lockedReader struct {
	mu     sync.Mutex
	reader io.Reader
}

func (r *lockedReader) Read(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.reader.Read(p)
}

func randomText(rng io.Reader, size int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, size)
	buf := make([]byte, size)
	if _, err := io.ReadFull(rng, buf); err != nil {
		return string(b)
	}
	for i := range b {
		b[i] = chars[int(buf[i])%len(chars)]
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
