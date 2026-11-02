package conf

import (
	"bytes"
	"testing"
)

// FuzzConfigRoundTrip: Parse→Bytes сохраняет произвольный ввод байт-в-байт (SPEC 5.3),
// а операции с peer'ами не паникуют.
func FuzzConfigRoundTrip(f *testing.F) {
	for _, s := range []string{"", "\n", "[Interface]\nPrivateKey = x\n",
		"# krot:type=awg2\n[Interface]\nAddress = 10.8.0.1/24\n\n# krot:id=a\n[Peer]\nPublicKey = p\nAllowedIPs = 10.8.0.2/32\n",
		"#~ [Peer]\r\n#~ PublicKey = p\r\n"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		c := Parse([]byte(s))
		b := c.Bytes()
		if !bytes.Equal(b, []byte(s)) {
			t.Fatalf("round trip changed the input:\n%q\n→\n%q", s, b)
		}
		if !bytes.Equal(Parse(b).Bytes(), b) {
			t.Fatal("Parse(Bytes()) is not idempotent")
		}
		_, _ = c.Interface()
		for _, p := range c.Peers() {
			_ = p.PublicKey()
		}
		c.AddPeer(NewPeer{ID: "fz", Created: "2026-11-02T00:00:00Z", PublicKey: "k", AllowedIPs: []string{"10.8.0.9/32"}})
		_ = c.LockPeer("fz", "2026-11-02T00:00:00Z")
		_ = c.UnlockPeer("fz")
		_ = c.RemovePeer("fz")
		_ = c.Bytes()
	})
}
