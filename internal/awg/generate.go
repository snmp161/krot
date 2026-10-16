package awg

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
)

// DefaultMTU — MTU по умолчанию для типа (SPEC 4.4).
func DefaultMTU(t Type) int {
	switch t {
	case TypeAWG, TypeAWG2:
		return 1376
	case TypeAWG3:
		return 1280
	}
	return 1420
}

// gen — генератор равномерных чисел поверх io.Reader (crypto/rand.Reader в рабочем коде).
type gen struct {
	r   io.Reader
	err error
}

// intn возвращает равномерное число в [lo, hi] (rejection sampling, без смещения).
func (g *gen) intn(lo, hi uint64) uint64 {
	if g.err != nil {
		return lo
	}
	span := hi - lo + 1
	var buf [8]byte
	for {
		if _, err := io.ReadFull(g.r, buf[:]); err != nil {
			g.err = fmt.Errorf("read random: %w", err)
			return lo
		}
		x := binary.LittleEndian.Uint64(buf[:])
		if span == 0 { // весь диапазон uint64
			return x
		}
		if limit := -span % span; x >= limit { // отбрасываем неполный «хвост»
			return lo + x%span
		}
	}
}

func (g *gen) int(lo, hi int) int { return int(g.intn(uint64(lo), uint64(hi))) }

func itoa(n int) string { return strconv.Itoa(n) }

func rng(lo, hi uint64) string { return strconv.FormatUint(lo, 10) + "-" + strconv.FormatUint(hi, 10) }

// Generate генерирует параметры обфускации для типа по политике SPEC 7.2–7.4.
// i1 — значение --i1 ("" — без I1). Для wg параметров нет.
func Generate(t Type, mtu int, i1 string, r io.Reader) (Params, error) {
	if t == TypeWG {
		return nil, nil
	}
	g := &gen{r: r}
	v := map[string]string{}

	// Jc, Jmin, Jmax — SPEC 7.2, общие для всех awg*.
	jmin := g.int(64, 128)
	v["Jc"] = itoa(g.int(4, 12))
	v["Jmin"] = itoa(jmin)
	v["Jmax"] = itoa(g.int(jmin+64, min(1024, mtu)))

	switch t {
	case TypeAWG, TypeAWG2:
		s1, s2 := g.int(15, 150), g.int(15, 150)
		for s1+56 == s2 && g.err == nil {
			s2 = g.int(15, 150)
		}
		v["S1"], v["S2"] = itoa(s1), itoa(s2)
		if t == TypeAWG {
			var hs []uint64
			for len(hs) < 4 && g.err == nil {
				h := g.intn(5, maxH1x)
				if !containsU64(hs, h) {
					hs = append(hs, h)
				}
			}
			for i, h := range hs {
				v[hKeys[i]] = strconv.FormatUint(h, 10)
			}
			break
		}
		// awg2: размеры 148+S1, 92+S2, 64+S3 попарно различны (политика SPEC 7.3).
		s3 := g.int(15, 150)
		for (64+s3 == 148+s1 || 64+s3 == 92+s2) && g.err == nil {
			s3 = g.int(15, 150)
		}
		v["S3"], v["S4"] = itoa(s3), itoa(g.int(0, 32))
		for i, r := range genHRanges(g) {
			v[hKeys[i]] = rng(r.Lo, r.Hi)
		}
	case TypeAWG3:
		var hp [32]byte
		if _, err := io.ReadFull(r, hp[:]); err != nil && g.err == nil {
			g.err = fmt.Errorf("read random: %w", err)
		}
		v["HeaderProtectionKey"] = base64.StdEncoding.EncodeToString(hp[:])
		s := itoa(g.int(12, 64))
		v["S1"], v["S2"], v["S3"], v["S4"] = s, s, s, s
		v["H1"], v["H2"], v["H3"], v["H4"] = "1", "2", "3", "4"
		v["ContentPaddingAddition"] = rng(0, g.intn(16, 64))
		timings := []struct {
			key                string
			aLo, aHi, bLo, bHi uint64
		}{
			{"RekeyAfterTime", 100, 115, 125, 140},
			{"RejectAfterTime", 170, 180, 185, 200},
			{"RekeyTimeout", 4, 5, 6, 8},
			{"KeepaliveTimeout", 8, 10, 11, 15},
			{"MaxHandshakeAttempts", 15, 18, 19, 22},
		}
		for _, x := range timings {
			v[x.key] = rng(g.intn(x.aLo, x.aHi), g.intn(x.bLo, x.bHi))
		}
		v["RandomTrailers"], v["DisableCookies"] = "on", "on"
	}
	if i1 != "" {
		v["I1"] = i1
	}
	if g.err != nil {
		return nil, g.err
	}
	var p Params
	for _, k := range Keys {
		if x, ok := v[k]; ok {
			p = append(p, Param{k, x})
		}
	}
	return p, nil
}

func containsU64(s []uint64, x uint64) bool {
	for _, y := range s {
		if y == x {
			return true
		}
	}
	return false
}

const (
	hMinWidth = 1 << 16
	hMaxWidth = 1 << 22
	hMinGap   = 1 << 16
)

// genHRanges — четыре непересекающихся диапазона в [5, 2^31-1] шириной [2^16, 2^22]
// с зазором не меньше 2^16 (SPEC 7.3). Порядок H1..H4 случайный.
func genHRanges(g *gen) []Range {
	for g.err == nil {
		rs := make([]Range, 0, 4)
		for len(rs) < 4 {
			w := g.intn(hMinWidth, hMaxWidth)
			lo := g.intn(5, maxH1x-w+1)
			rs = append(rs, Range{lo, lo + w - 1})
		}
		if separated(rs) {
			return rs
		}
	}
	return make([]Range, 4)
}

func separated(rs []Range) bool {
	for i := range rs {
		for j := i + 1; j < len(rs); j++ {
			a, b := rs[i], rs[j]
			if b.Lo < a.Lo {
				a, b = b, a
			}
			if a.Hi >= b.Lo || b.Lo-a.Hi-1 < hMinGap {
				return false
			}
		}
	}
	return true
}
