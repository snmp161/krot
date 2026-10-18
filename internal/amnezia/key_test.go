package amnezia

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"io"
	"regexp"
	"strings"
	"testing"
)

const wgConf = `[Interface]
PrivateKey = yAnz5TF+lXXJte14tji3zlMNq+hd2rYUIgJBgB3fBmk=
Address = 10.9.0.2/32
MTU = 1420

[Peer]
PublicKey = xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=
Endpoint = vpn.example.com:51820
AllowedIPs = 0.0.0.0/0
PersistentKeepalive = 25
`

const awgConf = `[Interface]
PrivateKey = yAnz5TF+lXXJte14tji3zlMNq+hd2rYUIgJBgB3fBmk=
Address = 10.8.0.2/32, fd00:8::2/128
DNS = 1.1.1.1, 9.9.9.9
MTU = 1376
Jc = 4
Jmin = 40
Jmax = 70
S1 = 15
S2 = 71
H1 = 1129453261
H2 = 2055471234
H3 = 3304523377
H4 = 4170316222
I1 = <b 0xf6ab3267fa><r 16>

[Peer]
PublicKey = xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=
PresharedKey = FpCyhws9cxwWoV4xELtfJvjJN+zQVRPISllRWgeopVE=
Endpoint = [2001:db8::1]:51820
AllowedIPs = 0.0.0.0/0, ::/0
PersistentKeepalive = 25
`

var keyRe = regexp.MustCompile(`^vpn://[A-Za-z0-9_-]+$`)

// decode — разбор ключа так, как это делает AmneziaVPN: base64url без `=`, затем qUncompress
// (4 байта длины big-endian + zlib-поток).
func decode(t *testing.T, key string) string {
	t.Helper()
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(key, "vpn://"))
	if err != nil {
		t.Fatalf("base64url: %v", err)
	}
	if len(raw) < 4 {
		t.Fatalf("payload too short: %d bytes", len(raw))
	}
	r, err := zlib.NewReader(bytes.NewReader(raw[4:]))
	if err != nil {
		t.Fatalf("zlib: %v", err)
	}
	text, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("zlib: %v", err)
	}
	if n := binary.BigEndian.Uint32(raw[:4]); int(n) != len(text) {
		t.Fatalf("length header %d, uncompressed %d bytes", n, len(text))
	}
	return string(text)
}

// SPEC 6.5: ключ AmneziaVPN — одна строка vpn://, base64url без `=`.
func TestKeyFormat(t *testing.T) {
	for _, c := range []string{wgConf, awgConf} {
		key := Key(c)
		if !keyRe.MatchString(key) {
			t.Fatalf("key %q does not match %s", key, keyRe)
		}
	}
}

// SPEC 6.5: декодирование (base64url → длина → zlib) возвращает конфиг байт-в-байт.
func TestKeyRoundTrip(t *testing.T) {
	big := strings.Replace(awgConf, "<b 0xf6ab3267fa><r 16>", "<b 0x"+strings.Repeat("ab", 40000)+">", 1)
	for name, c := range map[string]string{"wg": wgConf, "awg": awgConf, "big": big} {
		t.Run(name, func(t *testing.T) {
			if got := decode(t, Key(c)); got != c {
				t.Fatalf("decoded config differs:\n%s\nwant:\n%s", got, c)
			}
		})
	}
}

// Ключ детерминирован: один и тот же конфиг — одна и та же строка.
func TestKeyDeterministic(t *testing.T) {
	if Key(awgConf) != Key(awgConf) {
		t.Fatal("Key is not deterministic")
	}
	if Key(awgConf) == Key(wgConf) {
		t.Fatal("different configs give the same key")
	}
}
