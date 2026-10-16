package awg

import "testing"

// SPEC 3.2: тип и версия существующего интерфейса — по метаданным krot или по параметрам [Interface].
func TestDetect(t *testing.T) {
	hp := Param{"HeaderProtectionKey", hpKey}
	cases := []struct {
		name     string
		amnezia  bool
		meta     string
		p        Params
		wantType Type
		wantVer  string
	}{
		{"wg dir", false, "", nil, TypeWG, "-"},
		{"wg dir ignores params", false, "", Params{{"Jc", "4"}}, TypeWG, "-"},
		{"meta awg", true, "awg", Params{{"Jc", "4"}}, TypeAWG, "1.0"},
		{"meta awg with I1", true, "awg", Params{{"Jc", "4"}, {"I1", "<r 2>"}}, TypeAWG, "1.5"},
		{"meta awg2", true, "awg2", nil, TypeAWG2, "2.0"},
		{"meta awg3", true, "awg3", nil, TypeAWG3, "3.1"},
		{"meta wins over params", true, "awg2", Params{{"RandomTrailers", "on"}}, TypeAWG2, "2.0"},
		{"unknown meta falls back", true, "bogus", Params{{"S3", "1"}}, TypeAWG2, "2.0"},
		{"3.1 by RandomTrailers", true, "", Params{hp, {"RandomTrailers", "on"}}, TypeAWG3, "3.1"},
		{"3.1 by DisableCookies", true, "", Params{{"DisableCookies", "off"}}, TypeAWG3, "3.1"},
		{"3.0 by HP", true, "", Params{{"Jc", "4"}, hp, {"S3", "20"}}, TypeAWG3, "3.0"},
		{"3.0 by padding", true, "", Params{{"ContentPaddingAddition", "0-10"}}, TypeAWG3, "3.0"},
		{"3.0 by timing", true, "", Params{{"RekeyTimeout", "4-6"}}, TypeAWG3, "3.0"},
		{"2.0 by S3", true, "", Params{{"Jc", "4"}, {"S3", "20"}}, TypeAWG2, "2.0"},
		{"2.0 by S4", true, "", Params{{"S4", "2"}}, TypeAWG2, "2.0"},
		{"2.0 by H range", true, "", Params{{"Jc", "4"}, {"H1", "5-100"}}, TypeAWG2, "2.0"},
		{"2.0 with I1", true, "", Params{{"S3", "20"}, {"I1", "<r 2>"}}, TypeAWG2, "2.0"},
		{"1.5 by I1", true, "", Params{{"Jc", "4"}, {"I1", "<r 2>"}}, TypeAWG, "1.5"},
		{"1.5 by I5", true, "", Params{{"I5", "<t>"}}, TypeAWG, "1.5"},
		{"1.0 by Jc", true, "", Params{{"Jc", "4"}}, TypeAWG, "1.0"},
		{"1.0 by S1", true, "", Params{{"S1", "40"}}, TypeAWG, "1.0"},
		{"1.0 by H1", true, "", Params{{"H1", "12345"}}, TypeAWG, "1.0"},
		{"compat", true, "", nil, TypeAWG, "compat"},
	}
	for _, c := range cases {
		typ, ver := Detect(c.amnezia, c.meta, c.p)
		if typ != c.wantType || ver != c.wantVer {
			t.Errorf("%s: Detect = %s/%s; want %s/%s", c.name, typ, ver, c.wantType, c.wantVer)
		}
	}
}
