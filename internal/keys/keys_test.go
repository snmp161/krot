package keys

import (
	"bytes"
	"encoding/hex"
	"errors"
	"testing"
)

func mustHex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

// RFC 7748, раздел 6.1: тест-вектор X25519 (Alice).
func TestPublicRFC7748(t *testing.T) {
	var priv Key
	copy(priv[:], mustHex("77076d0a7318a57d3c16c17251b26645df4c2f87ebc0992ab177fba51db92c2a"))
	pub, err := priv.Public()
	if err != nil {
		t.Fatal(err)
	}
	if want := mustHex("8520f0098930a754748b7ddcb43ef75a0dbf3a0d26381af4eba4a98eaa9b4e6a"); !bytes.Equal(pub[:], want) {
		t.Fatalf("public = %x", pub)
	}
}

// Приватный ключ clamp'ится как у `wg genkey`.
func TestNewPrivateClamp(t *testing.T) {
	k, err := NewPrivate(bytes.NewReader(bytes.Repeat([]byte{0xff}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	if k[0] != 0xf8 || k[31] != 0x7f {
		t.Fatalf("clamp: %x", k)
	}
	k, _ = NewPrivate(bytes.NewReader(make([]byte, 32)))
	if k[0] != 0 || k[31] != 0x40 {
		t.Fatalf("clamp: %x", k)
	}
	if _, err := NewPrivate(bytes.NewReader(make([]byte, 10))); err == nil {
		t.Fatal("short read must fail")
	}
}

func TestPresharedAndEncoding(t *testing.T) {
	src := bytes.Repeat([]byte{7}, 32)
	psk, err := NewPreshared(bytes.NewReader(src))
	if err != nil || !bytes.Equal(psk[:], src) {
		t.Fatalf("psk = %x, %v", psk, err)
	}
	s := psk.String()
	if len(s) != 44 {
		t.Fatalf("base64 length = %d", len(s))
	}
	back, err := ParseKey(s)
	if err != nil || back != psk {
		t.Fatalf("ParseKey(%q) = %x, %v", s, back, err)
	}
	for _, bad := range []string{"", "abc", "!!!!", "AAAA"} {
		if _, err := ParseKey(bad); err == nil {
			t.Errorf("ParseKey(%q): expected error", bad)
		}
	}
	if _, err := NewPreshared(errReader{}); err == nil {
		t.Fatal("expected error")
	}
}

type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, errors.New("boom") }
