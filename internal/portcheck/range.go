package portcheck

import (
	"fmt"
	"strconv"
	"strings"
)

// MaxRangeSize — максимальное количество портов в диапазоне (SPEC 6.14).
const MaxRangeSize = 1024

// PortRange описывает диапазон UDP-портов, включительно.
type PortRange struct {
	Start int
	End   int
}

// ParsePortRange разбирает строку вида "N" или "N:M".
func ParsePortRange(s string) (PortRange, error) {
	if s == "" {
		return PortRange{}, fmt.Errorf("empty port range")
	}
	parts := strings.Split(s, ":")
	if len(parts) > 2 {
		return PortRange{}, fmt.Errorf("invalid port range %q", s)
	}
	start, err := parsePort(parts[0])
	if err != nil {
		return PortRange{}, fmt.Errorf("invalid port range %q: %w", s, err)
	}
	end := start
	if len(parts) == 2 {
		end, err = parsePort(parts[1])
		if err != nil {
			return PortRange{}, fmt.Errorf("invalid port range %q: %w", s, err)
		}
	}
	if start > end {
		return PortRange{}, fmt.Errorf("invalid port range %q: start > end", s)
	}
	if end-start+1 > MaxRangeSize {
		return PortRange{}, fmt.Errorf("invalid port range %q: %d ports, at most %d allowed", s, end-start+1, MaxRangeSize)
	}
	return PortRange{Start: start, End: end}, nil
}

func parsePort(s string) (int, error) {
	if s == "" {
		return 0, fmt.Errorf("empty port")
	}
	p, err := strconv.Atoi(s)
	if err != nil || p < 1 || p > 65535 {
		return 0, fmt.Errorf("port must be an integer 1..65535")
	}
	return p, nil
}

// Ports возвращает список портов диапазона.
func (r PortRange) Ports() []int {
	ports := make([]int, 0, r.End-r.Start+1)
	for p := r.Start; p <= r.End; p++ {
		ports = append(ports, p)
	}
	return ports
}

// Size возвращает количество портов в диапазоне.
func (r PortRange) Size() int { return r.End - r.Start + 1 }

// String возвращает строковое представление диапазона.
func (r PortRange) String() string {
	if r.Start == r.End {
		return strconv.Itoa(r.Start)
	}
	return fmt.Sprintf("%d:%d", r.Start, r.End)
}
