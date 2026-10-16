package awg

import (
	"encoding/binary"
	"errors"
	"math/rand/v2"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// seeded — детерминированный источник случайности для тестов (AGENTS.md: seed).
func seeded(seed uint64) *rand.ChaCha8 {
	var s [32]byte
	binary.LittleEndian.PutUint64(s[:], seed)
	return rand.NewChaCha8(s)
}

func val(t *testing.T, p Params, key string) string {
	t.Helper()
	v, ok := p.Get(key)
	if !ok {
		t.Fatalf("%s missing in %v", key, p)
	}
	return v
}

func n(t *testing.T, p Params, key string) int {
	t.Helper()
	x, err := strconv.Atoi(val(t, p, key))
	if err != nil {
		t.Fatalf("%s: %v", key, err)
	}
	return x
}

func rangeOf(t *testing.T, p Params, key string) Range {
	t.Helper()
	r, _, ok, _ := parseRange(val(t, p, key), maxU32)
	if !ok {
		t.Fatalf("%s = %q is not a range", key, val(t, p, key))
	}
	return r
}

func in(x, lo, hi int) bool { return x >= lo && x <= hi }

// Политика генерации SPEC 7.2: общие для awg/awg2/awg3 параметры Jc/Jmin/Jmax.
func checkJunk(t *testing.T, p Params, mtu int) {
	t.Helper()
	jc, jmin, jmax := n(t, p, "Jc"), n(t, p, "Jmin"), n(t, p, "Jmax")
	if !in(jc, 4, 12) || !in(jmin, 64, 128) || !in(jmax, jmin+64, min(1024, mtu)) {
		t.Fatalf("junk policy violated: Jc=%d Jmin=%d Jmax=%d mtu=%d", jc, jmin, jmax, mtu)
	}
}

// SPEC 7, 8.2: для каждого типа 10 000 генераций с разными seed — каждая проходит валидацию своего типа.
func TestGenerateProperty(t *testing.T) {
	const runs = 10000
	const i1 = "<b 0xf6ab3267fa><r 16>"
	kinds := []struct {
		name string
		typ  Type
		i1   string
	}{
		{"awg", TypeAWG, ""},
		{"awg+i1", TypeAWG, i1},
		{"awg2", TypeAWG2, ""},
		{"awg3", TypeAWG3, ""},
	}
	for _, k := range kinds {
		t.Run(k.name, func(t *testing.T) {
			for seed := uint64(0); seed < runs; seed++ {
				mtu := DefaultMTU(k.typ)
				if seed%2 == 1 {
					mtu = 576 + int(seed%8425) // разные MTU в пределах 576..9000
				}
				p, err := Generate(k.typ, mtu, k.i1, seeded(seed))
				if err != nil {
					t.Fatalf("seed %d: %v", seed, err)
				}
				if err := Validate(k.typ, p, mtu); err != nil {
					t.Fatalf("seed %d mtu %d: generated params fail validation: %v\n%v", seed, mtu, err, p)
				}
				if got, _ := p.Get("I1"); got != k.i1 {
					t.Fatalf("seed %d: I1 = %q; want %q", seed, got, k.i1)
				}
				checkJunk(t, p, mtu)
				switch k.typ {
				case TypeAWG:
					checkAWG1Policy(t, p)
				case TypeAWG2:
					checkAWG2Policy(t, p)
				case TypeAWG3:
					checkAWG3Policy(t, p)
				}
			}
		})
	}
}

// SPEC 7.2: S1,S2 ∈ [15,150], S1+56 ≠ S2 (отдельная проверка из 8.2), H в [5, 2^31-1] различны.
func checkAWG1Policy(t *testing.T, p Params) {
	t.Helper()
	s1, s2 := n(t, p, "S1"), n(t, p, "S2")
	if !in(s1, 15, 150) || !in(s2, 15, 150) {
		t.Fatalf("S1=%d S2=%d out of [15,150]", s1, s2)
	}
	if s1+56 == s2 {
		t.Fatalf("awg generated S1+56=S2 (S1=%d S2=%d)", s1, s2)
	}
	var hs []int
	for _, k := range hKeys {
		h := n(t, p, k)
		if !in(h, 5, maxH1x) || slices.Contains(hs, h) {
			t.Fatalf("bad %s = %d (%v)", k, h, hs)
		}
		hs = append(hs, h)
	}
	if p.Has("S3") || p.Has("S4") {
		t.Fatal("awg must not have S3/S4")
	}
}

// SPEC 7.3: S3 ∈ [15,150], S4 ∈ [0,32], размеры 148+S1, 92+S2, 64+S3 попарно различны;
// H — непересекающиеся диапазоны (отдельная проверка из 8.2) шириной [2^16, 2^22] с зазором ≥ 2^16.
func checkAWG2Policy(t *testing.T, p Params) {
	t.Helper()
	s1, s2, s3, s4 := n(t, p, "S1"), n(t, p, "S2"), n(t, p, "S3"), n(t, p, "S4")
	if !in(s1, 15, 150) || !in(s2, 15, 150) || !in(s3, 15, 150) || !in(s4, 0, 32) {
		t.Fatalf("S out of policy: %d %d %d %d", s1, s2, s3, s4)
	}
	if a, b, c := 148+s1, 92+s2, 64+s3; a == b || b == c || a == c {
		t.Fatalf("packet sizes not distinct: %d %d %d", a, b, c)
	}
	var rs []Range
	for _, k := range hKeys {
		r := rangeOf(t, p, k)
		w := r.Hi - r.Lo + 1
		if r.Lo < 5 || r.Hi > maxH1x || w < 1<<16 || w > 1<<22 {
			t.Fatalf("%s = %v violates width/bounds policy", k, r)
		}
		for _, o := range rs {
			if r.overlaps(o) {
				t.Fatalf("awg2 generated overlapping H ranges %v and %v", r, o)
			}
			lo, hi := o, r
			if r.Lo < o.Lo {
				lo, hi = r, o
			}
			if hi.Lo-lo.Hi-1 < 1<<16 {
				t.Fatalf("gap between %v and %v is less than 2^16", lo, hi)
			}
		}
		rs = append(rs, r)
	}
}

// SPEC 7.4: генерация awg3.
func checkAWG3Policy(t *testing.T, p Params) {
	t.Helper()
	s := n(t, p, "S1")
	if !in(s, 12, 64) {
		t.Fatalf("S=%d out of [12,64]", s)
	}
	for _, k := range []string{"S2", "S3", "S4"} {
		if n(t, p, k) != s {
			t.Fatalf("%s differs from S1", k)
		}
	}
	for i, k := range hKeys {
		if n(t, p, k) != i+1 {
			t.Fatalf("%s != %d", k, i+1)
		}
	}
	cpa := rangeOf(t, p, "ContentPaddingAddition")
	if cpa.Lo != 0 || cpa.Hi < 16 || cpa.Hi > 64 {
		t.Fatalf("ContentPaddingAddition = %v", cpa)
	}
	bounds := map[string][4]uint64{
		"RekeyAfterTime":       {100, 115, 125, 140},
		"RejectAfterTime":      {170, 180, 185, 200},
		"RekeyTimeout":         {4, 5, 6, 8},
		"KeepaliveTimeout":     {8, 10, 11, 15},
		"MaxHandshakeAttempts": {15, 18, 19, 22},
	}
	for k, b := range bounds {
		r := rangeOf(t, p, k)
		if r.Lo < b[0] || r.Lo > b[1] || r.Hi < b[2] || r.Hi > b[3] {
			t.Fatalf("%s = %v out of policy %v", k, r, b)
		}
	}
	if val(t, p, "RandomTrailers") != "on" || val(t, p, "DisableCookies") != "on" {
		t.Fatal("RandomTrailers/DisableCookies must be on")
	}
}

func TestGenerateDeterministicAndOrdered(t *testing.T) {
	for _, typ := range []Type{TypeAWG, TypeAWG2, TypeAWG3} {
		a, err1 := Generate(typ, DefaultMTU(typ), "", seeded(42))
		b, err2 := Generate(typ, DefaultMTU(typ), "", seeded(42))
		if err1 != nil || err2 != nil || !slices.Equal(a, b) {
			t.Fatalf("%s: same seed gives different params", typ)
		}
		// Порядок ключей — канонический.
		last := -1
		for _, kv := range a {
			idx := slices.Index(Keys, kv.Key)
			if idx <= last {
				t.Fatalf("%s: key %s out of canonical order: %v", typ, kv.Key, a)
			}
			last = idx
		}
	}
	if p, err := Generate(TypeWG, 1420, "", seeded(1)); err != nil || len(p) != 0 {
		t.Fatalf("wg: %v, %v", p, err)
	}
}

func TestDefaultMTU(t *testing.T) {
	want := map[Type]int{TypeWG: 1420, TypeAWG: 1376, TypeAWG2: 1376, TypeAWG3: 1280}
	for typ, m := range want {
		if got := DefaultMTU(typ); got != m {
			t.Errorf("DefaultMTU(%s) = %d; want %d", typ, got, m)
		}
	}
}

type failReader struct{}

func (failReader) Read([]byte) (int, error) { return 0, errors.New("no entropy") }

func TestGenerateRandError(t *testing.T) {
	for _, typ := range []Type{TypeAWG, TypeAWG2, TypeAWG3} {
		if _, err := Generate(typ, DefaultMTU(typ), "", failReader{}); err == nil || !strings.Contains(err.Error(), "entropy") {
			t.Errorf("%s: err = %v", typ, err)
		}
	}
}
