package awg

import "testing"

// FuzzValidateCPS: грамматика CPS (SPEC 7.1 V-05) не должна падать на произвольном вводе.
func FuzzValidateCPS(f *testing.F) {
	for _, s := range []string{"", "<b 0xf6ab3267fa><r 16>", "<t><rc 10><rd 5>", "<x>", "<b 0xabc>", "text", "<r 1001>"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_ = ValidateCPS(s)
	})
}

// FuzzParseVersion: разбор версий (SPEC 3.1) не должен падать на произвольном вводе.
func FuzzParseVersion(f *testing.F) {
	for _, s := range []string{"", "1.0.20250901", "v3.1.20260812", "v3.0.20260805-3-gabc-dirty", "1.0", "999.999.99999999"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_, _ = ParseVersion(s)
	})
}
