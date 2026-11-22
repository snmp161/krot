package qr

import (
	"errors"
	"flag"
	"math/rand/v2"
	"os"
	"strings"
	"testing"
	"unicode/utf8"

	"rsc.io/qr"
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

// SPEC 6.5 T48: QR символами Unicode quadrant; конфиг помещается.
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
			if !strings.ContainsRune(" ▘▝▖▗▚▞▌▐▀▄▛▜▙▟█", r) {
				t.Fatalf("unexpected rune %q", r)
			}
		}
	}
	// Ячейка 2×2 сохраняет равное число терминальных строк и колонок.
	if h := len(lines); h != width {
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

var quadrantAlphabet = [16]rune{' ', '▘', '▝', '▀', '▖', '▌', '▞', '▛', '▗', '▚', '▐', '▜', '▄', '▙', '▟', '█'}

// SPEC 6.5 T48: биты UL, UR, LL, LR соответствуют стандартным quadrant-символам.
func TestQuadrantMapping(t *testing.T) {
	for mask, want := range quadrantAlphabet {
		if got := quadrantRune(byte(mask)); got != want {
			t.Errorf("mask %04b: got %q, want %q", mask, got, want)
		}
	}
}

// SPEC 6.5 T48: каждый символ содержит две колонки и две строки модулей.
func TestQuadrantDimensions(t *testing.T) {
	for _, length := range []int{1, 100, 500, 1000, 2000, 2953} {
		text := lowerText(length)
		code, err := qr.Encode(text, qr.L)
		if err != nil {
			t.Fatal(err)
		}
		out, err := Render(text)
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
		want := (code.Size + 2*margin + 1) / 2
		if len(lines) != want || utf8.RuneCountInString(lines[0]) != want {
			t.Errorf("length %d: dimensions %dx%d, want %dx%d", length,
				utf8.RuneCountInString(lines[0]), len(lines), want, want)
		}
	}
}

// SPEC 6.5 T48: вывод round-trip восстанавливает QR, quiet zone и padding без потерь.
func TestQuadrantRoundTrip(t *testing.T) {
	inverse := make(map[rune]byte)
	for mask, r := range quadrantAlphabet {
		inverse[r] = byte(mask)
	}
	cells := []struct {
		dx, dy int
		bit    byte
	}{{0, 0, 1}, {1, 0, 2}, {0, 1, 4}, {1, 1, 8}}

	for _, length := range []int{1, 100, 500, 1000, 2000, 2953} {
		text := lowerText(length)
		code, err := qr.Encode(text, qr.L)
		if err != nil {
			t.Fatal(err)
		}
		out, err := Render(text)
		if err != nil {
			t.Fatal(err)
		}
		for row, line := range strings.Split(strings.TrimSuffix(out, "\n"), "\n") {
			for col, r := range []rune(line) {
				mask, ok := inverse[r]
				if !ok {
					t.Fatalf("length %d: unexpected rune %q", length, r)
				}
				for _, cell := range cells {
					x, y := col*2+cell.dx, row*2+cell.dy
					want := true
					if x >= margin && y >= margin && x < margin+code.Size && y < margin+code.Size {
						want = !code.Black(x-margin, y-margin)
					}
					got := mask&cell.bit != 0
					if got != want {
						t.Fatalf("length %d: module (%d,%d): got light=%v, want %v", length, x, y, got, want)
					}
				}
			}
		}
	}
}
