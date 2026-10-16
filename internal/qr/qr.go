// Package qr выводит текст QR-кодом в терминал символами Unicode half-block (SPEC 6.5).
package qr

import (
	"errors"
	"strings"

	"rsc.io/qr"
)

// ErrTooLarge — текст не помещается в QR-код версии 40 с уровнем коррекции L.
var ErrTooLarge = errors.New("config too large for QR")

// margin — ширина «тихой зоны» в модулях.
const margin = 2

// Render кодирует текст (уровень коррекции L, версия подбирается до 40) и рисует его
// полублоками: одна строка текста — две строки модулей. Светлые модули рисуются символами,
// тёмные — пробелами (под тёмный фон терминала, как qrencode -t UTF8).
func Render(text string) (string, error) {
	code, err := qr.Encode(text, qr.L)
	if err != nil {
		return "", ErrTooLarge
	}
	n := code.Size + 2*margin
	dark := func(x, y int) bool {
		x, y = x-margin, y-margin
		if y >= code.Size+margin {
			return true // за нижним краем ничего не рисуем
		}
		return x >= 0 && y >= 0 && x < code.Size && y < code.Size && code.Black(x, y)
	}
	var b strings.Builder
	for y := 0; y < n; y += 2 {
		for x := 0; x < n; x++ {
			top, bottom := !dark(x, y), y+1 < n && !dark(x, y+1)
			switch {
			case top && bottom:
				b.WriteRune('█')
			case top:
				b.WriteRune('▀')
			case bottom:
				b.WriteRune('▄')
			default:
				b.WriteRune(' ')
			}
		}
		b.WriteByte('\n')
	}
	return b.String(), nil
}
