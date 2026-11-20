package i18n

import (
	"strings"
	"testing"
)

var supportedLanguages = []Language{
	English,
	Russian,
	ChineseSimplified,
	ChineseTraditional,
	Spanish,
	Portuguese,
	Korean,
	Japanese,
	Arabic,
	French,
	German,
}

// SPEC 4.7 T27: все языки README выбираются через стандартные locale-переменные.
func TestDetectAllLanguages(t *testing.T) {
	tests := []struct {
		locale string
		want   Language
	}{
		{"en_US.UTF-8", English},
		{"ru_RU.UTF-8", Russian},
		{"RU-ru@latin", Russian},
		{"zh", ChineseSimplified},
		{"zh_CN.UTF-8", ChineseSimplified},
		{"zh-SG", ChineseSimplified},
		{"zh_Hans", ChineseSimplified},
		{"zh_TW.UTF-8", ChineseTraditional},
		{"zh-HK", ChineseTraditional},
		{"zh_MO@ Traditional", ChineseTraditional},
		{"zh_Hant", ChineseTraditional},
		{"es", Spanish},
		{"es_MX.UTF-8", Spanish},
		{"es-ES", Spanish},
		{"pt_BR.UTF-8", Portuguese},
		{"pt-PT", Portuguese},
		{"ko_KR.UTF-8", Korean},
		{"ja_JP.UTF-8", Japanese},
		{"ar", Arabic},
		{"ar_SA.UTF-8", Arabic},
		{"fr_FR.UTF-8", French},
		{"de-DE", German},
		{"C", English},
		{"POSIX", English},
		{"xx_XX", English},
		{"russian", English},
	}
	for _, tt := range tests {
		t.Run(tt.locale, func(t *testing.T) {
			got := Detect(func(key string) string {
				if key == "LANG" {
					return tt.locale
				}
				return ""
			})
			if got != tt.want {
				t.Fatalf("LANG=%q: Detect() = %q, want %q", tt.locale, got, tt.want)
			}
		})
	}
}

// SPEC 4.7 T27: переводы не пусты и сохраняют форматирование исходных сообщений.
func TestAllMessageCatalogFormatting(t *testing.T) {
	for _, language := range supportedLanguages {
		if language == English {
			continue
		}
		catalog := messageCatalogs[language]
		if len(catalog) == 0 {
			t.Errorf("%s: catalog is missing or empty", language)
			continue
		}
		for english, translated := range catalog {
			if strings.TrimSpace(translated) == "" {
				t.Errorf("%s: empty translation for %q", language, english)
			}
			if got, want := formatVerbs(translated), formatVerbs(english); got != want {
				t.Errorf("%s: format verbs for %q = %q, want %q", language, english, got, want)
			}
			if got, want := strings.Count(translated, "\n"), strings.Count(english, "\n"); got != want {
				t.Errorf("%s: newline count for %q = %d, want %d", language, english, got, want)
			}
			if got, want := strings.Count(translated, "\t"), strings.Count(english, "\t"); got != want {
				t.Errorf("%s: tab count for %q = %d, want %d", language, english, got, want)
			}
		}
	}
}

// SPEC 4.7 T27: подтверждение использует формы множественного числа языка.
func TestLanguagePlurals(t *testing.T) {
	previous := Current()
	t.Cleanup(func() { SetLanguage(previous) })
	tests := []struct {
		language Language
		cases    map[int]string
	}{
		{English, map[int]string{0: "clients", 1: "client", 2: "clients"}},
		{Russian, map[int]string{0: "клиентов", 1: "клиент", 2: "клиента", 5: "клиентов", 11: "клиентов", 21: "клиент", 22: "клиента", 111: "клиентов"}},
		{ChineseSimplified, map[int]string{0: "客户端", 1: "客户端", 2: "客户端"}},
		{ChineseTraditional, map[int]string{0: "客戶端", 1: "客戶端", 2: "客戶端"}},
		{Spanish, map[int]string{0: "clientes", 1: "cliente", 2: "clientes"}},
		{Portuguese, map[int]string{0: "cliente", 1: "cliente", 2: "clientes"}},
		{Korean, map[int]string{0: "클라이언트", 1: "클라이언트", 2: "클라이언트"}},
		{Japanese, map[int]string{0: "クライアント", 1: "クライアント", 2: "クライアント"}},
		{Arabic, map[int]string{0: "عملاء", 1: "عميل", 2: "عميلان", 3: "عملاء", 10: "عملاء", 11: "عميلًا", 99: "عميلًا", 100: "عميل", 103: "عملاء", 111: "عميلًا"}},
		{French, map[int]string{0: "client", 1: "client", 2: "clients"}},
		{German, map[int]string{0: "Clients", 1: "Client", 2: "Clients"}},
	}
	for _, tt := range tests {
		t.Run(string(tt.language), func(t *testing.T) {
			SetLanguage(tt.language)
			for n, want := range tt.cases {
				if got := Plural(n, "client"); got != want {
					t.Errorf("Plural(%d, client) = %q, want %q", n, got, want)
				}
			}
		})
	}
}
