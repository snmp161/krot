// Package qr выводит текст QR-кодом в терминал символами Unicode quadrant (SPEC 6.5).
package qr

import (
	"strings"

	"rsc.io/qr"

	"github.com/snmp161/krot/internal/i18n"
)

// ErrTooLarge — текст не помещается в QR-код версии 40 с уровнем коррекции L.
var ErrTooLarge = i18n.NewError("config too large for QR")

// margin — ширина «тихой зоны» в модулях.
const margin = 2

var quadrantRunes = [16]rune{' ', '▘', '▝', '▀', '▖', '▌', '▞', '▛', '▗', '▚', '▐', '▜', '▄', '▙', '▟', '█'}

// quadrantRune отображает биты UL, UR, LL, LR на соответствующий Unicode quadrant.
func quadrantRune(mask byte) rune {
	return quadrantRunes[mask&0x0f]
}

// Render кодирует текст (уровень коррекции L, версия подбирается до 40) и упаковывает
// 2×2 модуля в одну ячейку. Светлые модули изображаются сегментами на тёмном фоне;
// quiet zone и дополнение правого/нижнего края до полной ячейки остаются светлыми.
func Render(text string) (string, error) {
	code, err := qr.Encode(text, qr.L)
	if err != nil {
		return "", ErrTooLarge
	}
	n := code.Size + 2*margin
	light := func(x, y int) bool {
		x, y = x-margin, y-margin
		return x < 0 || y < 0 || x >= code.Size || y >= code.Size || !code.Black(x, y)
	}
	var b strings.Builder
	for y := 0; y < n; y += 2 {
		for x := 0; x < n; x += 2 {
			var mask byte
			if light(x, y) {
				mask |= 1
			}
			if light(x+1, y) {
				mask |= 2
			}
			if light(x, y+1) {
				mask |= 4
			}
			if light(x+1, y+1) {
				mask |= 8
			}
			b.WriteRune(quadrantRune(mask))
		}
		b.WriteByte('\n')
	}
	return b.String(), nil
}
