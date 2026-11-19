package app

import (
	"strings"
	"testing"

	"github.com/snmp161/krot/internal/errs"
	"github.com/snmp161/krot/internal/i18n"
)

func useRussian(t *testing.T) {
	t.Helper()
	previous := i18n.Current()
	i18n.SetLanguage(i18n.Russian)
	t.Cleanup(func() { i18n.SetLanguage(previous) })
}

// SPEC 4.7: справка, версия, ошибки и подсказка выводятся на русском.
func TestRussianHelpVersionAndErrors(t *testing.T) {
	useRussian(t)
	e := newEnv(t)
	e.mustRun(0, "--help")
	for _, want := range []string{
		"krot — управление туннелями WireGuard и AmneziaWG",
		"Действия (ровно одно):",
		"Коды выхода:",
	} {
		if !strings.Contains(e.stdout.String(), want) {
			t.Errorf("Russian help misses %q:\n%s", want, e.stdout)
		}
	}

	e.mustRun(0, "--version")
	if got, want := e.stdout.String(), "krot v1.2.3 (коммит abc1234, собран 2026-10-17T00:00:00Z)\n"; got != want {
		t.Errorf("Russian version = %q, want %q", got, want)
	}

	e.mustRun(errs.Usage, "--bogus")
	for _, want := range []string{
		"krot: неизвестный флаг --bogus",
		"Попробуйте 'krot --help' для получения дополнительной информации.",
	} {
		if !strings.Contains(e.stderr.String(), want) {
			t.Errorf("Russian error misses %q: %q", want, e.stderr.String())
		}
	}

	e.mustRun(errs.Invalid, "--list", "--port", "invalid")
	if !strings.Contains(e.stderr.String(), "недопустимое значение --port") {
		t.Errorf("Russian validation error: %q", e.stderr.String())
	}
}

// SPEC 4.7: таблица локализуется, а пустой JSON остаётся неизменным.
func TestRussianListAndJSON(t *testing.T) {
	useRussian(t)
	e := newEnv(t)
	e.mustRun(0, "--list")
	if !strings.Contains(e.stdout.String(), "ИНТЕРФЕЙС") || !strings.Contains(e.stdout.String(), "СОСТОЯНИЕ") {
		t.Fatalf("Russian table:\n%s", e.stdout)
	}
	e.mustRun(0, "--list", "--json")
	if got := e.stdout.String(); got != "[]\n" {
		t.Fatalf("Russian JSON = %q, want unchanged empty array", got)
	}
}

// SPEC 4.7: подтверждение удаления учитывает русские формы множественного числа.
func TestRussianDeleteConfirmation(t *testing.T) {
	useRussian(t)
	e := newEnv(t)
	e.write(e.awgConf("awg0"), fixture(t, "awg0_two_clients.conf"))
	e.tty = true
	e.stdin = "n\n"
	e.mustRun(errs.Cancelled, "--delete", "--interface", "awg0")
	if !strings.Contains(e.stderr.String(), "В интерфейсе awg0: 2 клиента. Удалить? [y/N]") {
		t.Fatalf("Russian confirmation: %q", e.stderr.String())
	}
	if !strings.Contains(e.stderr.String(), "Удаление интерфейса awg0 отменено") {
		t.Fatalf("Russian cancellation: %q", e.stderr.String())
	}
}

// SPEC 4.7: английский остаётся языком по умолчанию.
func TestEnglishFallbackOutput(t *testing.T) {
	previous := i18n.Current()
	i18n.SetLanguage(i18n.English)
	t.Cleanup(func() { i18n.SetLanguage(previous) })
	e := newEnv(t)
	e.mustRun(0, "--help")
	if !strings.HasPrefix(e.stdout.String(), "krot — manage WireGuard and AmneziaWG tunnels\n") {
		t.Fatalf("English help changed:\n%s", e.stdout)
	}
	e.mustRun(0, "--list")
	if !strings.HasPrefix(e.stdout.String(), "INTERFACE") {
		t.Fatalf("English table changed:\n%s", e.stdout)
	}
}
