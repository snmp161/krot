package awg

import (
	"slices"
	"strings"
)

// Param — параметр [Interface] AmneziaWG.
type Param struct {
	Key, Value string
}

// Params — набор параметров в порядке вывода.
type Params []Param

// Get возвращает значение ключа (без учёта регистра) и признак наличия.
func (p Params) Get(key string) (string, bool) {
	for _, kv := range p {
		if strings.EqualFold(kv.Key, key) {
			return kv.Value, true
		}
	}
	return "", false
}

// Has сообщает, задан ли ключ.
func (p Params) Has(key string) bool {
	_, ok := p.Get(key)
	return ok
}

// Канонические имена и порядок всех параметров AmneziaWG в [Interface].
// J1–J3 и Itime существовали в ранних сборках 1.5 и нужны только для копирования (SPEC 7.5).
var (
	keys10     = []string{"Jc", "Jmin", "Jmax", "S1", "S2", "H1", "H2", "H3", "H4"}
	keysS34    = []string{"S3", "S4"}
	keysI      = []string{"I1", "I2", "I3", "I4", "I5"}
	keysLegacy = []string{"J1", "J2", "J3", "Itime"}
	keysTiming = []string{"RekeyAfterTime", "RejectAfterTime", "RekeyTimeout", "KeepaliveTimeout", "MaxHandshakeAttempts"}
	keys30     = append([]string{"HeaderProtectionKey", "ContentPaddingAddition"}, keysTiming...)
	keys31     = []string{"RandomTrailers", "DisableCookies"}

	// Keys — все ключи AmneziaWG в каноническом порядке.
	Keys = slices.Concat(
		[]string{"Jc", "Jmin", "Jmax", "S1", "S2", "S3", "S4", "H1", "H2", "H3", "H4"},
		keysI, keysLegacy, keys30, keys31)
)

// CanonicalKey возвращает каноническое имя ключа AmneziaWG, если key им является.
func CanonicalKey(key string) (string, bool) {
	for _, k := range Keys {
		if strings.EqualFold(k, key) {
			return k, true
		}
	}
	return "", false
}

func hasAny(p Params, keys []string) (string, bool) {
	for _, k := range keys {
		if p.Has(k) {
			return k, true
		}
	}
	return "", false
}
