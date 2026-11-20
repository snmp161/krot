package app

import (
	"strings"
	"testing"

	"github.com/snmp161/krot/internal/errs"
	"github.com/snmp161/krot/internal/i18n"
)

// SPEC 4.7 T27: справка, ошибки и таблицы переводятся на каждый язык README,
// а JSON остаётся неизменным.
func TestAllApplicationLanguages(t *testing.T) {
	previous := i18n.Current()
	t.Cleanup(func() { i18n.SetLanguage(previous) })

	i18n.SetLanguage(i18n.English)
	englishEnv := newEnv(t)
	englishEnv.mustRun(0, "--help")
	englishHelp := englishEnv.stdout.String()

	languages := []i18n.Language{
		i18n.English,
		i18n.Russian,
		i18n.ChineseSimplified,
		i18n.ChineseTraditional,
		i18n.Spanish,
		i18n.Portuguese,
		i18n.Korean,
		i18n.Japanese,
		i18n.Arabic,
		i18n.French,
		i18n.German,
	}
	for _, language := range languages {
		t.Run(string(language), func(t *testing.T) {
			i18n.SetLanguage(language)
			e := newEnv(t)
			e.mustRun(0, "--help")
			for _, want := range []string{"krot", "--add-client", "--udp-checker"} {
				if !strings.Contains(e.stdout.String(), want) {
					t.Errorf("help misses %q", want)
				}
			}
			if language != i18n.English && e.stdout.String() == englishHelp {
				t.Error("translated help is identical to English")
			}

			e.mustRun(errs.Usage, "--bogus")
			if !strings.Contains(e.stderr.String(), "--bogus") || !strings.Contains(e.stderr.String(), "krot --help") {
				t.Errorf("localized usage error loses technical values: %q", e.stderr.String())
			}
			if language != i18n.English && strings.Contains(e.stderr.String(), "unknown flag") {
				t.Errorf("usage error is not translated: %q", e.stderr.String())
			}

			e.mustRun(0, "--list", "--json")
			if got := e.stdout.String(); got != "[]\n" {
				t.Errorf("localized JSON = %q, want unchanged empty array", got)
			}
		})
	}
}
