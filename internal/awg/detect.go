package awg

import "strings"

// Detect определяет тип и версию существующего интерфейса (SPEC 3.2).
// amnezia — конфиг лежит в каталоге AmneziaWG; meta — значение krot:type ("" — нет).
func Detect(amnezia bool, meta string, p Params) (Type, string) {
	if !amnezia {
		return TypeWG, "-"
	}
	switch Type(meta) {
	case TypeAWG:
		if p.Has("I1") {
			return TypeAWG, "1.5"
		}
		return TypeAWG, "1.0"
	case TypeAWG2:
		return TypeAWG2, "2.0"
	case TypeAWG3:
		return TypeAWG3, "3.1"
	}
	if _, ok := hasAny(p, keys31); ok {
		return TypeAWG3, "3.1"
	}
	if _, ok := hasAny(p, keys30); ok {
		return TypeAWG3, "3.0"
	}
	if _, ok := hasAny(p, keysS34); ok {
		return TypeAWG2, "2.0"
	}
	for _, k := range hKeys {
		if v, ok := p.Get(k); ok && strings.Contains(v, "-") {
			return TypeAWG2, "2.0"
		}
	}
	if _, ok := hasAny(p, keysI); ok {
		return TypeAWG, "1.5"
	}
	if _, ok := hasAny(p, keys10); ok {
		return TypeAWG, "1.0"
	}
	return TypeAWG, "compat"
}
