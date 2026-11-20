package i18n

import (
	"regexp"
	"strings"
	"testing"
)

// SPEC 4.7: первая непустая переменная LC_ALL, LC_MESSAGES, LANG выбирает язык.
func TestDetectPrecedence(t *testing.T) {
	env := map[string]string{
		"LC_ALL":      "en_US.UTF-8",
		"LC_MESSAGES": "ru_RU.UTF-8",
		"LANG":        "ru_RU.UTF-8",
	}
	getenv := func(key string) string { return env[key] }
	if got := Detect(getenv); got != English {
		t.Fatalf("Detect() = %q, want %q", got, English)
	}
	env["LC_ALL"] = ""
	if got := Detect(getenv); got != Russian {
		t.Fatalf("LC_MESSAGES fallback = %q, want %q", got, Russian)
	}
	env["LC_MESSAGES"] = ""
	if got := Detect(getenv); got != Russian {
		t.Fatalf("LANG fallback = %q, want %q", got, Russian)
	}
}

// SPEC 4.7: распознаются русские локали, остальные значения дают английский.
func TestDetectLocales(t *testing.T) {
	for _, locale := range []string{"ru", "ru_RU", "ru_RU.UTF-8", "ru-RU", "ru@latin"} {
		env := map[string]string{"LANG": locale}
		if got := Detect(func(key string) string { return env[key] }); got != Russian {
			t.Errorf("LANG=%q: Detect() = %q, want %q", locale, got, Russian)
		}
	}
	for _, locale := range []string{"", "C", "POSIX", "en_US.UTF-8", "de_DE.UTF-8", "russian", "en_RU"} {
		env := map[string]string{"LANG": locale}
		if got := Detect(func(key string) string { return env[key] }); got != English {
			t.Errorf("LANG=%q: Detect() = %q, want %q", locale, got, English)
		}
	}
}

// SPEC 4.7: русские сообщения переводятся, отсутствующий перевод остаётся английским.
func TestRussianFormattingAndFallback(t *testing.T) {
	previous := Current()
	SetLanguage(Russian)
	t.Cleanup(func() { SetLanguage(previous) })

	if got := Sprintf("unknown flag --%s", "bogus"); got != "неизвестный флаг --bogus" {
		t.Fatalf("translated error = %q", got)
	}
	const untranslated = "message without a translation: %s"
	if got := Sprintf(untranslated, "value"); got != "message without a translation: value" {
		t.Fatalf("English fallback = %q", got)
	}
	if got := Plural(1, "client"); got != "клиент" {
		t.Errorf("Plural(1) = %q", got)
	}
	for _, n := range []int{2, 3, 4, 22, 134} {
		if got := Plural(n, "client"); got != "клиента" {
			t.Errorf("Plural(%d) = %q", n, got)
		}
	}
	for _, n := range []int{0, 5, 11, 12, 14, 20, 21, 25, 101, 111} {
		want := "клиентов"
		if n == 21 || n == 101 {
			want = "клиент"
		}
		if got := Plural(n, "client"); got != want {
			t.Errorf("Plural(%d) = %q, want %q", n, got, want)
		}
	}
}

// SPEC 4.7: каталог не содержит пустых переводов и сохраняет форматирующие глаголы.
func TestRussianCatalog(t *testing.T) {
	for english, russian := range russianMessages {
		if strings.TrimSpace(russian) == "" {
			t.Errorf("empty translation for %q", english)
		}
		if got, want := formatVerbs(russian), formatVerbs(english); got != want {
			t.Errorf("format verbs for %q = %q, want %q", english, got, want)
		}
	}
}

func formatVerbs(format string) string {
	re := regexp.MustCompile(`%[#0 +\-]*[0-9]*(?:\.[0-9]+)?[a-zA-Z]`)
	return strings.Join(re.FindAllString(format, -1), ",")
}
