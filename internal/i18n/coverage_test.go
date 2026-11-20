package i18n

import (
	"errors"
	"testing"
)

// SPEC 4.7 T36: Configure сразу применяет выбранную локаль.
func TestConfigureAppliesLanguage(t *testing.T) {
	previous := Current()
	t.Cleanup(func() { SetLanguage(previous) })
	got := Configure(func(key string) string {
		if key == "LANG" {
			return "fr_FR.UTF-8"
		}
		return ""
	})
	if got != French || Current() != French {
		t.Fatalf("Configure() = %q, Current() = %q; want French", got, Current())
	}
}

// SPEC 4.7 T36: ошибки, созданные заранее, переводятся при обращении к Error.
func TestLocalizedErrors(t *testing.T) {
	previous := Current()
	t.Cleanup(func() { SetLanguage(previous) })
	SetLanguage(Russian)
	err := NewError("peer not found")
	if got := err.Error(); got != "peer не найден" {
		t.Fatalf("NewError().Error() = %q", got)
	}
	if got := Errorf("invalid --port %q: %v", "bad", errors.New("failure")).Error(); got != "недопустимое значение --port \"bad\": failure" {
		t.Fatalf("Errorf() = %q", got)
	}
}

// SPEC 4.7 T36: неизвестные язык и ключ используют безопасный fallback.
func TestPluralFallbacks(t *testing.T) {
	previous := Current()
	t.Cleanup(func() { SetLanguage(previous) })
	SetLanguage(Language("unknown"))
	if got := Plural(1, "client"); got != "client" {
		t.Fatalf("singular fallback = %q", got)
	}
	if got := Plural(2, "client"); got != "clients" {
		t.Fatalf("plural fallback = %q", got)
	}
	if got := Plural(1, "missing"); got != "missing" {
		t.Fatalf("unknown key fallback = %q", got)
	}
}
