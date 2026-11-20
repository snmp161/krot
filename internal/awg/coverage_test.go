package awg

import "testing"

// SPEC 3 T36: типы протоколов разбираются строго, AmneziaWG отличается от wg.
func TestTypeAndCanonicalKeyContracts(t *testing.T) {
	for _, typ := range Types {
		got, ok := ParseType(string(typ))
		if !ok || got != typ {
			t.Errorf("ParseType(%q) = %q, %v", typ, got, ok)
		}
		if got.IsAmnezia() != (typ != TypeWG) {
			t.Errorf("%s IsAmnezia() = %v", typ, got.IsAmnezia())
		}
	}
	if got, ok := ParseType("WG"); ok || got != "" {
		t.Fatalf("ParseType(WG) = %q, %v", got, ok)
	}
	if got, ok := CanonicalKey("headerprotectionkey"); !ok || got != "HeaderProtectionKey" {
		t.Fatalf("CanonicalKey() = %q, %v", got, ok)
	}
	if got, ok := CanonicalKey("NotAwg"); ok || got != "" {
		t.Fatalf("CanonicalKey(unknown) = %q, %v", got, ok)
	}
}

// SPEC 7.6 T36: алфавит динамических CPS-полей проверяется независимо.
func TestDynamicFieldAlphabets(t *testing.T) {
	if !isDigits([]byte("0123456789")) || isDigits([]byte("12a")) {
		t.Error("isDigits contract failed")
	}
	if !isLetters([]byte("abcXYZ")) || isLetters([]byte("abc1")) {
		t.Error("isLetters contract failed")
	}
}
