// Package i18n provides the built-in en_US and ru_RU message catalogs (SPEC 4.7).
package i18n

import (
	"fmt"
	"strings"
	"sync/atomic"
)

type Language string

const (
	English Language = "en_US"
	Russian Language = "ru_RU"
)

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
		if locale == "ru" || (strings.HasPrefix(locale, "ru") && len(locale) > 2 && strings.ContainsRune("_-.@", rune(locale[2]))) {
			return Russian
		}
		return English
	}
	return English
}

func Configure(getenv func(string) string) Language {
	language := Detect(getenv)
	SetLanguage(language)
	return language
}

func Text(english string) string {
	if Current() == Russian {
		if russian, ok := russianMessages[english]; ok {
			return russian
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

func Plural(n int, englishOne, englishMany, russianOne, russianFew, russianMany string) string {
	if Current() != Russian {
		if n == 1 {
			return englishOne
		}
		return englishMany
	}
	if n < 0 {
		n = -n
	}
	switch {
	case n%100 >= 11 && n%100 <= 14:
		return russianMany
	case n%10 == 1:
		return russianOne
	case n%10 >= 2 && n%10 <= 4:
		return russianFew
	default:
		return russianMany
	}
}
