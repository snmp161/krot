package awg

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strconv"
)

const (
	maxU16 = 1<<16 - 1
	maxU32 = 1<<32 - 1
	// SPEC 7.2 V-10: верхняя граница H для awg 1.x.
	maxH1x = 2147483647
	// SPEC 7.1 V-04: накладные расходы WireGuard IPv6.
	wgOverhead = 80
)

// Range — диапазон a-b; одиночное число — диапазон a-a.
type Range struct {
	Lo, Hi uint64
}

func (r Range) overlaps(o Range) bool { return r.Lo <= o.Hi && o.Lo <= r.Hi }

var (
	uintRe  = regexp.MustCompile(`^[0-9]+$`)
	rangeRe = regexp.MustCompile(`^([0-9]+)(?:-([0-9]+))?$`)
)

// parseRange разбирает "a" или "a-b". overflow — граница больше max.
func parseRange(s string, max uint64) (r Range, isRange, ok, overflow bool) {
	m := rangeRe.FindStringSubmatch(s)
	if m == nil {
		return Range{}, false, false, false
	}
	lo, err1 := strconv.ParseUint(m[1], 10, 64)
	hi, err2 := lo, error(nil)
	if m[2] != "" {
		isRange = true
		hi, err2 = strconv.ParseUint(m[2], 10, 64)
	}
	if err1 != nil || err2 != nil || lo > max || hi > max {
		return Range{}, isRange, false, true
	}
	return Range{lo, hi}, isRange, true, false
}

func parseU16(s string) (int, bool) {
	if !uintRe.MatchString(s) {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	return n, err == nil && n <= maxU16
}

// num возвращает числовой параметр (после проверки V-01).
func num(p Params, key string) (int, bool) {
	v, ok := p.Get(key)
	if !ok {
		return 0, false
	}
	return parseU16(v)
}

func rule(id, format string, a ...any) error {
	return fmt.Errorf(id+": "+format, a...)
}

var hKeys = []string{"H1", "H2", "H3", "H4"}

// Validate проверяет параметры AmneziaWG по правилам SPEC 7 для типа t и MTU интерфейса.
// Ошибка содержит номер нарушенного правила (V-xx).
func Validate(t Type, p Params, mtu int) error {
	if t == TypeWG {
		if len(p) > 0 {
			return fmt.Errorf("type wg has no AmneziaWG parameters, got %s", p[0].Key)
		}
		return nil
	}
	if err := validateCommon(p, mtu); err != nil {
		return err
	}
	switch t {
	case TypeAWG:
		return validateAWG1(p)
	case TypeAWG2:
		return validateAWG2(p)
	case TypeAWG3:
		return validateAWG3(p, mtu)
	}
	return fmt.Errorf("unknown type %q", t)
}

// SPEC 7.1: V-01…V-05.
func validateCommon(p Params, mtu int) error {
	for _, k := range []string{"Jc", "Jmin", "Jmax", "S1", "S2", "S3", "S4"} {
		if v, ok := p.Get(k); ok {
			if _, ok := parseU16(v); !ok {
				return rule("V-01", "%s = %q must be an integer 0..%d", k, v, maxU16)
			}
		}
	}
	for _, k := range hKeys {
		if v, ok := p.Get(k); ok {
			if _, _, _, overflow := parseRange(v, maxU32); overflow {
				return rule("V-01", "%s = %q exceeds uint32", k, v)
			}
		}
	}
	for _, k := range append([]string{"ContentPaddingAddition"}, keysTiming...) {
		if v, ok := p.Get(k); ok {
			if _, _, _, overflow := parseRange(v, maxU16); overflow {
				return rule("V-01", "%s = %q exceeds uint16", k, v)
			}
		}
	}
	jmin, okMin := num(p, "Jmin")
	jmax, okMax := num(p, "Jmax")
	if okMin && okMax && jmin > jmax {
		return rule("V-02", "Jmin (%d) must not exceed Jmax (%d)", jmin, jmax)
	}
	if okMax && jmax > mtu {
		return rule("V-03", "Jmax (%d) must not exceed MTU (%d)", jmax, mtu)
	}
	if s1, ok := num(p, "S1"); ok && 148+s1 > mtu+wgOverhead {
		return rule("V-04", "148 + S1 (%d) must not exceed MTU + 80 (%d)", 148+s1, mtu+wgOverhead)
	}
	if s2, ok := num(p, "S2"); ok && 92+s2 > mtu+wgOverhead {
		return rule("V-04", "92 + S2 (%d) must not exceed MTU + 80 (%d)", 92+s2, mtu+wgOverhead)
	}
	for _, k := range keysI {
		if v, ok := p.Get(k); ok {
			if err := ValidateCPS(v); err != nil {
				return rule("V-05", "%s: %v", k, err)
			}
		}
	}
	return nil
}

// V-11: размеры Init (148+S1) и Response (92+S2) не совпадают.
func checkV11(p Params) error {
	s1, ok1 := num(p, "S1")
	s2, ok2 := num(p, "S2")
	if ok1 && ok2 && s1+56 == s2 {
		return rule("V-11", "S1 + 56 must not equal S2 (S1=%d, S2=%d)", s1, s2)
	}
	return nil
}

// SPEC 7.2: awg 1.x.
func validateAWG1(p Params) error {
	for _, k := range keysS34 {
		if p.Has(k) {
			return rule("V-12", "%s is not allowed for awg (1.x)", k)
		}
	}
	if k, ok := hasAny(p, keys30); ok {
		return rule("V-12", "%s (AmneziaWG 3.x) is not allowed for awg (1.x)", k)
	}
	if k, ok := hasAny(p, keys31); ok {
		return rule("V-12", "%s (AmneziaWG 3.1) is not allowed for awg (1.x)", k)
	}
	var hs []uint64
	for _, k := range hKeys {
		v, ok := p.Get(k)
		if !ok {
			return rule("V-10", "%s is required", k)
		}
		r, isRange, valid, _ := parseRange(v, maxU32)
		if isRange {
			return rule("V-12", "%s = %q: ranges are not allowed for awg (1.x)", k, v)
		}
		if !valid || r.Lo < 5 || r.Lo > maxH1x {
			return rule("V-10", "%s = %q must be a number 5..%d", k, v, maxH1x)
		}
		for i, h := range hs {
			if h == r.Lo {
				return rule("V-10", "%s and %s must differ", hKeys[i], k)
			}
		}
		hs = append(hs, r.Lo)
	}
	return checkV11(p)
}

// SPEC 7.3: awg2 (2.0).
func validateAWG2(p Params) error {
	for _, k := range keysS34 {
		if !p.Has(k) {
			return rule("V-22", "%s is required for awg2", k)
		}
	}
	if k, ok := hasAny(p, append(keys30, keys31...)); ok {
		return rule("V-22", "%s (AmneziaWG 3.x) is not allowed for awg2", k)
	}
	var rs []Range
	for _, k := range hKeys {
		v, ok := p.Get(k)
		if !ok {
			return rule("V-20", "%s is required", k)
		}
		r, _, valid, _ := parseRange(v, maxU32)
		if !valid || r.Lo < 5 || r.Lo > r.Hi {
			return rule("V-20", "%s = %q must be a number or a range a-b with 5 <= a <= b <= %d", k, v, uint64(maxU32))
		}
		for i, o := range rs {
			if r.overlaps(o) {
				return rule("V-21", "%s and %s ranges overlap", hKeys[i], k)
			}
		}
		rs = append(rs, r)
	}
	return checkV11(p)
}

// SPEC 7.4: awg3 (3.1).
func validateAWG3(p Params, mtu int) error {
	for _, k := range keys31 {
		if v, ok := p.Get(k); ok && v != "on" && v != "off" {
			return rule("V-36", "%s = %q must be on or off", k, v)
		}
	}
	hp, ok := p.Get("HeaderProtectionKey")
	if !ok {
		return rule("V-30", "HeaderProtectionKey is required for awg3")
	}
	if b, err := base64.StdEncoding.DecodeString(hp); err != nil || len(b) != 32 {
		return rule("V-30", "HeaderProtectionKey must be 32 bytes encoded in base64")
	}
	var s [4]int
	for i, k := range []string{"S1", "S2", "S3", "S4"} {
		v, ok := num(p, k)
		if !ok || v < 12 {
			return rule("V-31", "%s must be set and at least 12 with header protection", k)
		}
		s[i] = v
	}
	for i, k := range hKeys {
		v, _ := p.Get(k)
		r, isRange, valid, _ := parseRange(v, maxU32)
		if !valid || isRange || r.Lo != uint64(i+1) {
			return rule("V-32", "with header protection H1..H4 must be 1, 2, 3, 4 (got %s = %q)", k, v)
		}
	}
	if v, _ := p.Get("RandomTrailers"); v == "on" && (s[0] != s[1] || s[1] != s[2] || s[2] != s[3]) {
		return rule("V-33", "with RandomTrailers = on S1, S2, S3, S4 must be equal (got %v)", s)
	}
	timing := map[string]Range{}
	for _, k := range keysTiming {
		v, ok := p.Get(k)
		if !ok {
			continue
		}
		r, _, valid, _ := parseRange(v, maxU16)
		if !valid || r.Lo > r.Hi {
			return rule("V-34", "%s = %q must be a range a-b with a <= b", k, v)
		}
		timing[k] = r
	}
	rel := []struct{ less, more string }{
		{"RekeyTimeout", "RekeyAfterTime"},
		{"KeepaliveTimeout", "RekeyAfterTime"},
		{"RekeyAfterTime", "RejectAfterTime"},
	}
	for _, x := range rel {
		a, ok1 := timing[x.less]
		b, ok2 := timing[x.more]
		if ok1 && ok2 && a.Hi >= b.Lo {
			return rule("V-34", "%s max (%d) must be less than %s min (%d)", x.less, a.Hi, x.more, b.Lo)
		}
	}
	if r, ok := timing["MaxHandshakeAttempts"]; ok && r.Lo < 1 {
		return rule("V-34", "MaxHandshakeAttempts min must be at least 1")
	}
	if v, ok := p.Get("ContentPaddingAddition"); ok {
		r, _, valid, _ := parseRange(v, maxU16)
		if !valid || r.Lo > r.Hi || r.Hi > uint64(mtu-wgOverhead) {
			return rule("V-35", "ContentPaddingAddition = %q must be a range a-b with a <= b <= MTU - 80 (%d)", v, mtu-wgOverhead)
		}
	}
	return nil
}
