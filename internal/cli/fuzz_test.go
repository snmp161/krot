package cli

import (
	"strings"
	"testing"
)

// FuzzParse: разбор аргументов CLI (SPEC 4) не должен падать на произвольном вводе.
func FuzzParse(f *testing.F) {
	for _, s := range []string{"--list", "--create --type awg2 --ip 10.8.0.1/24 --port 51820 --endpoint h.example",
		"--add-client --interface awg0 --id phone --allowed-ips 0.0.0.0/0 --psk --qr", "--", "--i1 <b 0xf6ab3267fa>"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_, _ = Parse(strings.Fields(s))
	})
}
