package qr

import (
	"errors"
	"flag"
	"math/rand/v2"
	"os"
	"strings"
	"testing"
	"unicode/utf8"
)

var update = flag.Bool("update", false, "update golden files")

func lowerText(n int) string {
	r := rand.New(rand.NewPCG(1, 2))
	b := make([]byte, n)
	for i := range b {
		b[i] = byte('a' + r.IntN(26))
	}
	return string(b)
}

// SPEC 6.5: QR полублоками Unicode; конфиг помещается.
func TestRenderFits(t *testing.T) {
	out, err := Render("[Interface]\nPrivateKey = x\n")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	width := utf8.RuneCountInString(lines[0])
	for _, l := range lines {
		if utf8.RuneCountInString(l) != width {
			t.Fatalf("ragged output:\n%s", out)
		}
		for _, r := range l {
			if !strings.ContainsRune(" █▀▄", r) {
				t.Fatalf("unexpected rune %q", r)
			}
		}
	}
	// Две строки модулей на строку текста: высота ≈ ширина / 2.
	if h := len(lines); h != (width+1)/2 {
		t.Fatalf("height %d for width %d", h, width)
	}
	if *update {
		os.WriteFile("testdata/hello.golden", []byte(mustRender(t, "hello")), 0o644)
	}
	want, err := os.ReadFile("testdata/hello.golden")
	if err != nil {
		t.Fatal(err)
	}
	if got := mustRender(t, "hello"); got != string(want) {
		t.Fatalf("golden mismatch:\n%s\nwant:\n%s", got, want)
	}
}

func mustRender(t *testing.T, s string) string {
	t.Helper()
	out, err := Render(s)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// SPEC 6.5: максимальная версия 40, уровень коррекции L — 2953 байта в байтовом режиме.
func TestRenderLimit(t *testing.T) {
	if _, err := Render(lowerText(2953)); err != nil {
		t.Fatalf("2953 bytes must fit: %v", err)
	}
	if _, err := Render(lowerText(2954)); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("2954 bytes: err = %v; want ErrTooLarge", err)
	}
}
