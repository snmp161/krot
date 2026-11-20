package project

import (
	"strings"
	"testing"
)

// SPEC 4.7, 8.4 T27: package smoke покрывает каждый язык и man-страницу.
func TestAllLocalizationSmoke(t *testing.T) {
	smoke := readRepoFile(t, "scripts/smoke.sh")
	for _, want := range []string{
		"ru_RU.UTF-8", "zh_CN.UTF-8", "zh_TW.UTF-8", "es_MX.UTF-8",
		"pt_BR.UTF-8", "ko_KR.UTF-8", "ja_JP.UTF-8", "ar_SA.UTF-8",
		"fr_FR.UTF-8", "de_DE.UTF-8",
		"/usr/share/man/man1/krot.1",
		"/usr/share/man/ru/man1/krot.1",
		"/usr/share/man/zh_CN/man1/krot.1",
		"/usr/share/man/zh_TW/man1/krot.1",
		"/usr/share/man/es/man1/krot.1",
		"/usr/share/man/pt_BR/man1/krot.1",
		"/usr/share/man/ko/man1/krot.1",
		"/usr/share/man/ja/man1/krot.1",
		"/usr/share/man/ar/man1/krot.1",
		"/usr/share/man/fr/man1/krot.1",
		"/usr/share/man/de/man1/krot.1",
	} {
		if !strings.Contains(smoke, want) {
			t.Errorf("smoke.sh misses %q", want)
		}
	}
}
