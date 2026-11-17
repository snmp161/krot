package portcheck

import (
	"slices"
	"testing"
)

// SPEC 6.14, 6.15: диапазон портов N:M, включительно, 1..65535, максимум 1024 порта.
func TestParsePortRange(t *testing.T) {
	cases := []struct {
		in   string
		want PortRange
		err  bool
	}{
		{"51820", PortRange{51820, 51820}, false},
		{"1", PortRange{1, 1}, false},
		{"65535", PortRange{65535, 65535}, false},
		{"51820:51822", PortRange{51820, 51822}, false},
		{"1:1024", PortRange{1, 1024}, false},
		{"", PortRange{}, true},
		{"0", PortRange{}, true},
		{"65536", PortRange{}, true},
		{"51822:51820", PortRange{}, true},
		{"51820:52852", PortRange{}, true}, // 1033 порта > 1024
		{"abc", PortRange{}, true},
		{"51820:", PortRange{}, true},
		{":51822", PortRange{}, true},
		{"51820-51822", PortRange{}, true},
	}
	for _, c := range cases {
		got, err := ParsePortRange(c.in)
		if c.err {
			if err == nil {
				t.Errorf("ParsePortRange(%q) = %v, want error", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParsePortRange(%q) error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("ParsePortRange(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestPortRangePorts(t *testing.T) {
	r := PortRange{51820, 51822}
	got := r.Ports()
	want := []int{51820, 51821, 51822}
	if !slices.Equal(got, want) {
		t.Fatalf("Ports() = %v, want %v", got, want)
	}
}

func TestPortRangeMaxSize(t *testing.T) {
	// Ровно 1024 порта — допустимо.
	_, err := ParsePortRange("1:1024")
	if err != nil {
		t.Fatalf("1:1024 should be valid: %v", err)
	}
	// 1025 портов — ошибка.
	_, err = ParsePortRange("1:1025")
	if err == nil {
		t.Fatal("1:1025 should be invalid")
	}
}
