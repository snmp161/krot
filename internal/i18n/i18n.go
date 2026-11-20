// Package i18n provides built-in message catalogs for all README languages (SPEC 4.7).
package i18n

import (
	"fmt"
	"strings"
	"sync/atomic"
)

type Language string

const (
	English            Language = "en_US"
	Russian            Language = "ru_RU"
	ChineseSimplified  Language = "zh_Hans"
	ChineseTraditional Language = "zh_Hant"
	Spanish            Language = "es_MX"
	Portuguese         Language = "pt_BR"
	Korean             Language = "ko_KR"
	Japanese           Language = "ja_JP"
	Arabic             Language = "ar"
	French             Language = "fr_FR"
	German             Language = "de_DE"
)

var messageCatalogs = map[Language]map[string]string{
	Russian:            russianMessages,
	ChineseSimplified:  chineseSimplifiedMessages,
	ChineseTraditional: chineseTraditionalMessages,
	Spanish:            spanishMessages,
	Portuguese:         portugueseMessages,
	Korean:             koreanMessages,
	Japanese:           japaneseMessages,
	Arabic:             arabicMessages,
	French:             frenchMessages,
	German:             germanMessages,
}

var current atomic.Value

func init() {
	current.Store(English)
}

func Current() Language {
	return current.Load().(Language)
}

func SetLanguage(language Language) {
	current.Store(language)
}

func Detect(getenv func(string) string) Language {
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		locale := strings.TrimSpace(getenv(key))
		if locale == "" {
			continue
		}
		return fromLocale(locale)
	}
	return English
}

func fromLocale(locale string) Language {
	normalized := strings.ToLower(strings.TrimSpace(locale))
	normalized = strings.ReplaceAll(normalized, "-", "_")
	if i := strings.IndexAny(normalized, ".@"); i >= 0 {
		normalized = normalized[:i]
	}
	parts := strings.Split(normalized, "_")
	if len(parts) == 0 {
		return English
	}
	if parts[0] == "zh" {
		for _, part := range parts[1:] {
			switch part {
			case "hant", "tw", "hk", "mo":
				return ChineseTraditional
			}
		}
		return ChineseSimplified
	}
	switch parts[0] {
	case "ru":
		return Russian
	case "es":
		return Spanish
	case "pt":
		return Portuguese
	case "ko":
		return Korean
	case "ja":
		return Japanese
	case "ar":
		return Arabic
	case "fr":
		return French
	case "de":
		return German
	default:
		return English
	}
}

func Configure(getenv func(string) string) Language {
	language := Detect(getenv)
	SetLanguage(language)
	return language
}

func Text(english string) string {
	if catalog := messageCatalogs[Current()]; catalog != nil {
		if translated, ok := catalog[english]; ok {
			return translated
		}
	}
	return english
}

func Sprintf(english string, args ...any) string {
	return fmt.Sprintf(Text(english), args...)
}

func Errorf(english string, args ...any) error {
	return fmt.Errorf(Text(english), args...)
}

type messageError struct {
	english string
}

func (e messageError) Error() string {
	return Text(e.english)
}

func NewError(english string) error {
	return messageError{english: english}
}
