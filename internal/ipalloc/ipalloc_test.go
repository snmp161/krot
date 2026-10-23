package ipalloc

import (
	"errors"
	"net/netip"
	"testing"
	"time"
)

func pfx(ss ...string) []netip.Prefix {
	var out []netip.Prefix
	for _, s := range ss {
		out = append(out, netip.MustParsePrefix(s))
	}
	return out
}

// SPEC 6.5 п.2: первый свободный адрес; не выдаются адрес сети, broadcast (IPv4),
// адреса сервера и адреса любых клиентов (включая заблокированных).
func TestAllocate(t *testing.T) {
	cases := []struct {
		name   string
		server string
		used   []netip.Prefix
		want   string // "" — исчерпано
	}{
		{"v4 empty", "10.8.0.1/24", nil, "10.8.0.2"},
		{"v4 skip used", "10.8.0.1/24", pfx("10.8.0.2/32", "10.8.0.3/32"), "10.8.0.4"},
		{"v4 hole", "10.8.0.1/24", pfx("10.8.0.2/32", "10.8.0.4/32"), "10.8.0.3"},
		{"v4 locked clients count", "10.8.0.1/24", pfx("10.8.0.3/32", "10.8.0.2/32"), "10.8.0.4"},
		{"v4 foreign wide AllowedIPs", "10.8.0.1/24", pfx("10.8.0.0/28"), "10.8.0.16"},
		{"v4 server in the middle", "10.8.0.10/24", nil, "10.8.0.1"},
		{"v4 skip server", "10.8.0.10/24", pfx("10.8.0.1/32", "10.8.0.2/32", "10.8.0.3/32", "10.8.0.4/32",
			"10.8.0.5/32", "10.8.0.6/32", "10.8.0.7/32", "10.8.0.8/32", "10.8.0.9/32"), "10.8.0.11"},
		{"v4 covered by a default route peer", "10.8.0.1/24", pfx("0.0.0.0/0"), ""},
		{"v4 full except broadcast", "10.8.0.1/24", pfx("10.8.0.0/25", "10.8.0.128/26", "10.8.0.192/27",
			"10.8.0.224/28", "10.8.0.240/29", "10.8.0.248/30", "10.8.0.252/31", "10.8.0.254/32"), ""},
		{"v4 /30", "10.8.0.1/30", nil, "10.8.0.2"},
		{"v4 /30 exhausted", "10.8.0.1/30", pfx("10.8.0.2/32"), ""},
		// SPEC 6.5: в /31 (RFC 3021) и /127 (RFC 6164) оба адреса допустимы.
		{"v4 /31 server high", "10.8.0.1/31", nil, "10.8.0.0"},
		{"v4 /31 server low", "10.8.0.0/31", nil, "10.8.0.1"},
		{"v4 /31 exhausted", "10.8.0.0/31", pfx("10.8.0.1/32"), ""},
		{"v4 /32 has no addresses", "10.8.0.1/32", nil, ""},
		{"v6 empty", "fd00:8::1/64", nil, "fd00:8::2"},
		{"v6 skip used", "fd00:8::1/64", pfx("fd00:8::2/128"), "fd00:8::3"},
		{"v6 /126", "fd00:8::1/126", pfx("fd00:8::2/128"), "fd00:8::3"},
		{"v6 /126 exhausted", "fd00:8::1/126", pfx("fd00:8::2/128", "fd00:8::3/128"), ""},
		{"v6 jump over huge prefix", "fd00:8::1/64", pfx("fd00:8::/65"), "fd00:8:0:0:8000::"},
		{"v6 /127 server high", "fd00:8::1/127", nil, "fd00:8::"},
		{"v6 /127 server low", "fd00:8::/127", nil, "fd00:8::1"},
		{"v6 /127 exhausted", "fd00:8::1/127", pfx("fd00:8::/128"), ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := Allocate(netip.MustParsePrefix(c.server), c.used)
			if c.want == "" {
				if !errors.Is(err, ErrExhausted) {
					t.Fatalf("got %v, %v; want ErrExhausted", got, err)
				}
				return
			}
			if err != nil || got != netip.MustParseAddr(c.want) {
				t.Fatalf("got %v, %v; want %s", got, err, c.want)
			}
		})
	}
}

func TestAllocateOtherFamilyIgnored(t *testing.T) {
	got, err := Allocate(netip.MustParsePrefix("10.8.0.1/24"), pfx("10.9.0.2/32", "fd00::2/128", "::/0"))
	if err != nil || got != netip.MustParseAddr("10.8.0.2") {
		t.Fatalf("got %v, %v", got, err)
	}
}

// Большая подсеть с тысячами клиентов — без квадратичного перебора.
func TestAllocateManyClients(t *testing.T) {
	var used []netip.Prefix
	a := netip.MustParseAddr("10.0.0.2")
	for i := 0; i < 60000; i++ {
		used = append(used, netip.PrefixFrom(a, 32))
		a = a.Next()
	}
	start := time.Now()
	got, err := Allocate(netip.MustParsePrefix("10.0.0.1/16"), used)
	if err != nil || got != a {
		t.Fatalf("got %v, %v; want %v", got, err, a)
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("too slow: %v", d)
	}
}

// SPEC 6.5 п.2: адрес, заданный вручную, должен быть в подсети интерфейса и свободен.
func TestCheck(t *testing.T) {
	servers := pfx("10.8.0.1/24", "fd00:8::1/64")
	used := pfx("10.8.0.2/32", "fd00:8::2/128", "10.8.0.64/26")
	cases := []struct {
		addr string
		want error
	}{
		{"10.8.0.3", nil},
		{"fd00:8::3", nil},
		{"10.9.0.3", ErrOutsideSubnet},
		{"fd01::3", ErrOutsideSubnet},
		{"10.8.0.0", ErrReserved},
		{"10.8.0.255", ErrReserved},
		{"10.8.0.1", ErrReserved},
		{"fd00:8::", ErrReserved},
		{"fd00:8::1", ErrReserved},
		{"10.8.0.2", ErrInUse},
		{"10.8.0.70", ErrInUse},
		{"fd00:8::2", ErrInUse},
	}
	for _, c := range cases {
		err := Check(netip.MustParseAddr(c.addr), servers, used)
		if !errors.Is(err, c.want) || (c.want == nil && err != nil) {
			t.Errorf("Check(%s) = %v; want %v", c.addr, err, c.want)
		}
	}
	if err := Check(netip.MustParseAddr("fd00:8::3"), pfx("10.8.0.1/24"), nil); !errors.Is(err, ErrOutsideSubnet) {
		t.Errorf("v6 without v6 subnet: %v", err)
	}
}

func TestIsNetworkOrBroadcast(t *testing.T) {
	cases := map[string]bool{
		"10.8.0.0/24":                  true,
		"10.8.0.255/24":                true,
		"10.8.0.1/24":                  false,
		"10.8.0.1/32":                  true,
		"10.8.0.0/30":                  true,
		"10.8.0.3/30":                  true,
		"10.8.0.1/31":                  false, // RFC 3021
		"10.8.0.0/31":                  false,
		"fd00::/127":                   false, // RFC 6164
		"fd00::1/127":                  false,
		"fd00::1/128":                  true,
		"fd00::/64":                    true,
		"fd00::1/64":                   false,
		"fd00::ffff:ffff:ffff:ffff/64": false,
	}
	for s, want := range cases {
		if got := IsNetworkOrBroadcast(netip.MustParsePrefix(s)); got != want {
			t.Errorf("IsNetworkOrBroadcast(%s) = %v; want %v", s, got, want)
		}
	}
}

// SPEC 6.5: в /31 и /127 второй адрес подсети — адрес клиента, а не адрес сети или broadcast.
func TestCheckPointToPoint(t *testing.T) {
	servers := pfx("10.8.0.0/31", "fd00:8::1/127")
	cases := []struct {
		addr string
		used []netip.Prefix
		want error
	}{
		{"10.8.0.1", nil, nil},
		{"fd00:8::", nil, nil},
		{"10.8.0.0", nil, ErrReserved},
		{"fd00:8::1", nil, ErrReserved},
		{"10.8.0.1", pfx("10.8.0.1/32"), ErrInUse},
		{"fd00:8::", pfx("fd00:8::/128"), ErrInUse},
		{"10.8.0.2", nil, ErrOutsideSubnet},
		{"fd00:8::2", nil, ErrOutsideSubnet},
	}
	for _, c := range cases {
		err := Check(netip.MustParseAddr(c.addr), servers, c.used)
		if !errors.Is(err, c.want) || (c.want == nil && err != nil) {
			t.Errorf("Check(%s, used %v) = %v; want %v", c.addr, c.used, err, c.want)
		}
	}
}
