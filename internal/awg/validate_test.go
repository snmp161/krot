package awg

import (
	"encoding/base64"
	"strings"
	"testing"
)

var hpKey = base64.StdEncoding.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))

var baseParams = map[Type]Params{
	TypeAWG: {
		{"Jc", "6"}, {"Jmin", "64"}, {"Jmax", "512"}, {"S1", "43"}, {"S2", "117"},
		{"H1", "194847201"}, {"H2", "905128442"}, {"H3", "1503002211"}, {"H4", "2001009901"},
	},
	TypeAWG2: {
		{"Jc", "6"}, {"Jmin", "64"}, {"Jmax", "512"}, {"S1", "43"}, {"S2", "117"}, {"S3", "31"}, {"S4", "12"},
		{"H1", "194847201-195371488"}, {"H2", "905128442-905652729"},
		{"H3", "1503002211-1503526498"}, {"H4", "2001009901-2001534188"},
	},
	TypeAWG3: {
		{"Jc", "6"}, {"Jmin", "64"}, {"Jmax", "512"}, {"S1", "32"}, {"S2", "32"}, {"S3", "32"}, {"S4", "32"},
		{"H1", "1"}, {"H2", "2"}, {"H3", "3"}, {"H4", "4"},
		{"HeaderProtectionKey", hpKey}, {"ContentPaddingAddition", "0-32"},
		{"RekeyAfterTime", "110-130"}, {"RejectAfterTime", "175-190"}, {"RekeyTimeout", "5-7"},
		{"KeepaliveTimeout", "9-12"}, {"MaxHandshakeAttempts", "16-20"},
		{"RandomTrailers", "on"}, {"DisableCookies", "on"},
	},
}

// with копирует базовый набор и применяет изменения: пустое значение — удалить ключ.
func with(t Type, kv ...string) Params {
	p := append(Params(nil), baseParams[t]...)
	for i := 0; i < len(kv); i += 2 {
		k, v := kv[i], kv[i+1]
		idx := -1
		for j := range p {
			if strings.EqualFold(p[j].Key, k) {
				idx = j
			}
		}
		switch {
		case v == "" && idx >= 0:
			p = append(p[:idx], p[idx+1:]...)
		case idx >= 0:
			p[idx].Value = v
		case v != "":
			p = append(p, Param{k, v})
		}
	}
	return p
}

func TestValidate(t *testing.T) {
	cases := []struct {
		name string
		typ  Type
		mtu  int
		p    Params
		rule string // "" — валидно
	}{
		{"awg base", TypeAWG, 1376, with(TypeAWG), ""},
		{"awg2 base", TypeAWG2, 1376, with(TypeAWG2), ""},
		{"awg3 base", TypeAWG3, 1280, with(TypeAWG3), ""},
		{"wg without params", TypeWG, 1420, nil, ""},
		{"wg with params", TypeWG, 1420, Params{{"Jc", "4"}}, "AmneziaWG"},

		// SPEC 7.1 V-01: uint16, H — uint32.
		{"V-01 Jc max", TypeAWG, 1376, with(TypeAWG, "Jc", "65535"), ""},
		{"V-01 Jc overflow", TypeAWG, 1376, with(TypeAWG, "Jc", "65536"), "V-01"},
		{"V-01 Jc negative", TypeAWG, 1376, with(TypeAWG, "Jc", "-1"), "V-01"},
		{"V-01 Jc not a number", TypeAWG, 1376, with(TypeAWG, "Jc", "abc"), "V-01"},
		{"V-01 S1 overflow", TypeAWG2, 1376, with(TypeAWG2, "S1", "65536"), "V-01"},
		{"V-01 S4 overflow", TypeAWG2, 1376, with(TypeAWG2, "S4", "70000"), "V-01"},
		{"V-01 H uint32 overflow awg", TypeAWG, 1376, with(TypeAWG, "H1", "4294967296"), "V-01"},
		{"V-01 H uint32 max awg2", TypeAWG2, 1376, with(TypeAWG2, "H4", "4000000000-4294967295"), ""},
		{"V-01 H uint32 overflow awg2", TypeAWG2, 1376, with(TypeAWG2, "H4", "4000000000-4294967296"), "V-01"},
		{"V-01 timing overflow", TypeAWG3, 1280, with(TypeAWG3, "RejectAfterTime", "175-65536"), "V-01"},

		// V-02: Jmin ≤ Jmax.
		{"V-02 equal", TypeAWG, 1376, with(TypeAWG, "Jmin", "512", "Jmax", "512"), ""},
		{"V-02 greater", TypeAWG, 1376, with(TypeAWG, "Jmin", "513", "Jmax", "512"), "V-02"},

		// V-03: Jmax ≤ MTU.
		{"V-03 equal", TypeAWG, 1376, with(TypeAWG, "Jmax", "1376"), ""},
		{"V-03 greater", TypeAWG, 1376, with(TypeAWG, "Jmax", "1377"), "V-03"},

		// V-04: 148+S1 ≤ MTU+80, 92+S2 ≤ MTU+80.
		{"V-04 S1 boundary", TypeAWG, 576, with(TypeAWG, "S1", "508"), ""},
		{"V-04 S1 over", TypeAWG, 576, with(TypeAWG, "S1", "509"), "V-04"},
		{"V-04 S2 boundary", TypeAWG, 576, with(TypeAWG, "S2", "564"), ""},
		{"V-04 S2 over", TypeAWG, 576, with(TypeAWG, "S2", "565"), "V-04"},

		// V-05: CPS в I1–I5.
		{"V-05 valid I1", TypeAWG, 1376, with(TypeAWG, "I1", "<b 0xf6ab3267fa><r 16>"), ""},
		{"V-05 valid I5", TypeAWG2, 1376, with(TypeAWG2, "I5", "<rc 10><rd 5><t>"), ""},
		{"V-05 unknown tag", TypeAWG, 1376, with(TypeAWG, "I1", "<x>"), "V-05"},
		{"V-05 N too big", TypeAWG2, 1376, with(TypeAWG2, "I3", "<r 1001>"), "V-05"},
		{"V-05 odd hex", TypeAWG3, 1280, with(TypeAWG3, "I2", "<b 0xabc>"), "V-05"},

		// SPEC 7.2 V-10: H одиночные, 5..2147483647, попарно различны.
		{"V-10 min", TypeAWG, 1376, with(TypeAWG, "H1", "5"), ""},
		{"V-10 below", TypeAWG, 1376, with(TypeAWG, "H1", "4"), "V-10"},
		{"V-10 max", TypeAWG, 1376, with(TypeAWG, "H1", "2147483647"), ""},
		{"V-10 above", TypeAWG, 1376, with(TypeAWG, "H1", "2147483648"), "V-10"},
		{"V-10 duplicate", TypeAWG, 1376, with(TypeAWG, "H3", "194847201"), "V-10"},
		{"V-10 missing", TypeAWG, 1376, with(TypeAWG, "H4", ""), "V-10"},

		// V-11: S1 + 56 ≠ S2 (awg; в awg2 размеры пакетов также различаются, для awg3 не применяется).
		{"V-11 equal sizes", TypeAWG, 1376, with(TypeAWG, "S1", "43", "S2", "99"), "V-11"},
		{"V-11 differ by one", TypeAWG, 1376, with(TypeAWG, "S1", "43", "S2", "100"), ""},
		{"V-11 awg2", TypeAWG2, 1376, with(TypeAWG2, "S1", "43", "S2", "99"), "V-11"},
		{"V-11 not for awg3", TypeAWG3, 1280, with(TypeAWG3, "RandomTrailers", "off", "S1", "32", "S2", "88"), ""},

		// V-12: в awg нет S3/S4, диапазонов H, ключей 3.x.
		{"V-12 S3", TypeAWG, 1376, with(TypeAWG, "S3", "10"), "V-12"},
		{"V-12 S4", TypeAWG, 1376, with(TypeAWG, "S4", "10"), "V-12"},
		{"V-12 H range", TypeAWG, 1376, with(TypeAWG, "H1", "5-10"), "V-12"},
		{"V-12 3.x key", TypeAWG, 1376, with(TypeAWG, "HeaderProtectionKey", hpKey), "V-12"},
		{"V-12 3.1 key", TypeAWG, 1376, with(TypeAWG, "RandomTrailers", "on"), "V-12"},

		// SPEC 7.3 V-20: Hn — число или a-b, 5 ≤ a ≤ b ≤ 4294967295.
		{"V-20 single", TypeAWG2, 1376, with(TypeAWG2, "H1", "7"), ""},
		{"V-20 a=b", TypeAWG2, 1376, with(TypeAWG2, "H1", "5-5"), ""},
		{"V-20 a<5", TypeAWG2, 1376, with(TypeAWG2, "H1", "4-10"), "V-20"},
		{"V-20 a>b", TypeAWG2, 1376, with(TypeAWG2, "H1", "10-5"), "V-20"},
		{"V-20 malformed", TypeAWG2, 1376, with(TypeAWG2, "H1", "1-2-3"), "V-20"},
		{"V-20 missing", TypeAWG2, 1376, with(TypeAWG2, "H2", ""), "V-20"},

		// V-21: диапазоны H не пересекаются.
		{"V-21 touching", TypeAWG2, 1376, with(TypeAWG2, "H1", "100-200", "H2", "200-300"), "V-21"},
		{"V-21 adjacent", TypeAWG2, 1376, with(TypeAWG2, "H1", "100-199", "H2", "200-300"), ""},
		{"V-21 single inside", TypeAWG2, 1376, with(TypeAWG2, "H1", "150", "H2", "100-200"), "V-21"},
		{"V-21 equal singles", TypeAWG2, 1376, with(TypeAWG2, "H1", "150", "H4", "150"), "V-21"},

		// V-22: S3, S4 есть, ключей 3.x нет.
		{"V-22 no S3", TypeAWG2, 1376, with(TypeAWG2, "S3", ""), "V-22"},
		{"V-22 no S4", TypeAWG2, 1376, with(TypeAWG2, "S4", ""), "V-22"},
		{"V-22 3.x key", TypeAWG2, 1376, with(TypeAWG2, "HeaderProtectionKey", hpKey), "V-22"},
		{"V-22 timing", TypeAWG2, 1376, with(TypeAWG2, "RekeyAfterTime", "110-130"), "V-22"},

		// SPEC 7.4 V-30: HeaderProtectionKey — base64 32 байт.
		{"V-30 missing", TypeAWG3, 1280, with(TypeAWG3, "HeaderProtectionKey", ""), "V-30"},
		{"V-30 short", TypeAWG3, 1280, with(TypeAWG3, "HeaderProtectionKey",
			base64.StdEncoding.EncodeToString(make([]byte, 31))), "V-30"},
		{"V-30 not base64", TypeAWG3, 1280, with(TypeAWG3, "HeaderProtectionKey", "not base64!"), "V-30"},

		// V-31: при HP S1..S4 ≥ 12.
		{"V-31 boundary", TypeAWG3, 1280, with(TypeAWG3, "S1", "12", "S2", "12", "S3", "12", "S4", "12"), ""},
		{"V-31 below", TypeAWG3, 1280, with(TypeAWG3, "S1", "11", "S2", "11", "S3", "11", "S4", "11"), "V-31"},
		{"V-31 S4 missing", TypeAWG3, 1280, with(TypeAWG3, "RandomTrailers", "off", "S4", ""), "V-31"},

		// V-32: при HP H1..H4 = 1, 2, 3, 4.
		{"V-32 H1=5", TypeAWG3, 1280, with(TypeAWG3, "H1", "5"), "V-32"},
		{"V-32 swapped", TypeAWG3, 1280, with(TypeAWG3, "H1", "2", "H2", "1"), "V-32"},

		// V-33: RandomTrailers = on → S1 = S2 = S3 = S4.
		{"V-33 differ", TypeAWG3, 1280, with(TypeAWG3, "S2", "33"), "V-33"},
		{"V-33 off allows differ", TypeAWG3, 1280, with(TypeAWG3, "RandomTrailers", "off", "S2", "33"), ""},

		// V-34: тайминги.
		{"V-34 rekeytimeout boundary", TypeAWG3, 1280, with(TypeAWG3, "RekeyTimeout", "5-109"), ""},
		{"V-34 rekeytimeout", TypeAWG3, 1280, with(TypeAWG3, "RekeyTimeout", "5-110"), "V-34"},
		{"V-34 keepalive boundary", TypeAWG3, 1280, with(TypeAWG3, "KeepaliveTimeout", "9-109"), ""},
		{"V-34 keepalive", TypeAWG3, 1280, with(TypeAWG3, "KeepaliveTimeout", "9-110"), "V-34"},
		{"V-34 reject boundary", TypeAWG3, 1280, with(TypeAWG3, "RekeyAfterTime", "110-174"), ""},
		{"V-34 reject", TypeAWG3, 1280, with(TypeAWG3, "RekeyAfterTime", "110-175"), "V-34"},
		{"V-34 attempts min 1", TypeAWG3, 1280, with(TypeAWG3, "MaxHandshakeAttempts", "1-20"), ""},
		{"V-34 attempts min 0", TypeAWG3, 1280, with(TypeAWG3, "MaxHandshakeAttempts", "0-20"), "V-34"},
		{"V-34 a>b", TypeAWG3, 1280, with(TypeAWG3, "RekeyTimeout", "7-5"), "V-34"},
		{"V-34 malformed", TypeAWG3, 1280, with(TypeAWG3, "RekeyTimeout", "five"), "V-34"},
		{"V-34 single value", TypeAWG3, 1280, with(TypeAWG3, "RekeyTimeout", "5"), ""},

		// V-35: ContentPaddingAddition a-b, a ≤ b, b ≤ MTU-80.
		{"V-35 boundary", TypeAWG3, 1280, with(TypeAWG3, "ContentPaddingAddition", "0-1200"), ""},
		{"V-35 over", TypeAWG3, 1280, with(TypeAWG3, "ContentPaddingAddition", "0-1201"), "V-35"},
		{"V-35 a>b", TypeAWG3, 1280, with(TypeAWG3, "ContentPaddingAddition", "10-5"), "V-35"},

		// V-36: RandomTrailers, DisableCookies — on/off.
		{"V-36 off", TypeAWG3, 1280, with(TypeAWG3, "DisableCookies", "off"), ""},
		{"V-36 yes", TypeAWG3, 1280, with(TypeAWG3, "RandomTrailers", "yes"), "V-36"},
		{"V-36 numeric", TypeAWG3, 1280, with(TypeAWG3, "DisableCookies", "1"), "V-36"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := Validate(c.typ, c.p, c.mtu)
			if c.rule == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), c.rule) {
				t.Fatalf("error = %v; want %s", err, c.rule)
			}
		})
	}
}

// SPEC 7.1 V-05: грамматика CPS (теги <b 0x…>, <t>, <r N>, <rc N>, <rd N>; N ≤ 1000; hex чётной длины).
func TestValidateCPS(t *testing.T) {
	valid := []string{
		"<b 0xf6ab3267fa>",
		"<b 0x00><t><r 10><rc 1000><rd 0>",
		"<b 0xABCDEF0123>",
		"<b 0xf6ab> <r 16>",
		"< r  16 >",
		"<t>",
	}
	for _, s := range valid {
		if err := ValidateCPS(s); err != nil {
			t.Errorf("ValidateCPS(%q) = %v; want ok", s, err)
		}
	}
	invalid := []string{
		"",
		"   ",
		"plain text",
		"<b 0xabc>",
		"<b 0x>",
		"<b abcd>",
		"<b 0xzz>",
		"<b>",
		"<t 5>",
		"<r>",
		"<r 1001>",
		"<r -1>",
		"<r 1e3>",
		"<rc abc>",
		"<rd 5 6>",
		"<c>",
		"<d>",
		"<ds>",
		"<dz 2>",
		"<x>",
		"<>",
		"<b 0x00",
		"<b 0x00>junk",
		"junk<t>",
		"<r 10>>",
	}
	for _, s := range invalid {
		if err := ValidateCPS(s); err == nil {
			t.Errorf("ValidateCPS(%q): expected error", s)
		}
	}
}
