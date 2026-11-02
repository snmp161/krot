package app

import "testing"

// FuzzParseDump: разбор `wg|awg show <if> dump` (SPEC 6.9) не должен падать на произвольном вводе.
func FuzzParseDump(f *testing.F) {
	for _, s := range []string{"", "priv\teth0\t51820\t\n",
		"priv\tawg0\t51820\t\npub\tpsk\t203.0.113.7:51820\t10.8.0.2/32\t1730000000\t100\t200\t25\t\n",
		"a\tb\nc\td\te\tf\tg\th\ti\tj\tk\n\n\n\t\t\t"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_ = parseDump(s)
	})
}
