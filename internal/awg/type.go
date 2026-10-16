// Package awg содержит знания о протоколах WireGuard/AmneziaWG: типы, таблицу версий,
// определение версии конфига, валидацию и генерацию параметров обфускации (SPEC 3, 7).
package awg

// Type — значение --type (SPEC 3).
type Type string

const (
	TypeWG   Type = "wg"
	TypeAWG  Type = "awg"
	TypeAWG2 Type = "awg2"
	TypeAWG3 Type = "awg3"
)

// Types — все допустимые типы в порядке из SPEC 3.
var Types = []Type{TypeWG, TypeAWG, TypeAWG2, TypeAWG3}

// ParseType разбирает значение --type.
func ParseType(s string) (Type, bool) {
	for _, t := range Types {
		if string(t) == s {
			return t, true
		}
	}
	return "", false
}

// IsAmnezia сообщает, относится ли тип к AmneziaWG (awg*).
func (t Type) IsAmnezia() bool {
	return t == TypeAWG || t == TypeAWG2 || t == TypeAWG3
}
