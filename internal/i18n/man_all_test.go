package i18n

import (
	"os"
	"strings"
	"testing"
)

var allManPages = []struct {
	file string
	dir  string
}{
	{"krot.1", "man1"},
	{"krot.ru.1", "ru/man1"},
	{"krot.zh.1", "zh_CN/man1"},
	{"krot.zh-Hant.1", "zh_TW/man1"},
	{"krot.es.1", "es/man1"},
	{"krot.pt.1", "pt_BR/man1"},
	{"krot.ko.1", "ko/man1"},
	{"krot.ja.1", "ja/man1"},
	{"krot.ar.1", "ar/man1"},
	{"krot.fr.1", "fr/man1"},
	{"krot.de.1", "de/man1"},
}

// SPEC 4.7 T27: все man-страницы имеют одинаковый технический состав.
func TestAllManPagesStructure(t *testing.T) {
	english, err := os.ReadFile("../../packaging/krot.1")
	if err != nil {
		t.Fatal(err)
	}
	englishSections := sectionCount(string(english))
	for _, page := range allManPages {
		t.Run(page.file, func(t *testing.T) {
			data, err := os.ReadFile("../../packaging/" + page.file)
			if err != nil {
				t.Fatal(err)
			}
			text := string(data)
			if !strings.HasPrefix(text, ".TH KROT 1") {
				t.Error("missing man title")
			}
			if got := sectionCount(text); got != englishSections {
				t.Errorf("section count = %d, want %d", got, englishSections)
			}
			plain := strings.ReplaceAll(text, `\-`, "-")
			for _, want := range []string{
				"--list", "--create", "--delete", "--add-client", "--delete-client",
				"--lock-client", "--unlock-client", "--list-client", "--doctor",
				"--udp-listener", "--udp-checker", "--allowed-ips", "--amnezia-key",
				"--keenetic", "--workers", "LC_ALL", "LC_MESSAGES", "LANG",
				"/etc/wireguard/*.conf", "/etc/amnezia/amneziawg/*.conf",
			} {
				if !strings.Contains(plain, want) {
					t.Errorf("missing %q", want)
				}
			}
		})
	}
}

// SPEC 4.7, 9.2 T27: nfpm устанавливает каждую локализованную страницу.
func TestAllManPagesPackaging(t *testing.T) {
	data, err := os.ReadFile("../../packaging/nfpm.yaml")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, page := range allManPages {
		for _, want := range []string{
			"src: ./packaging/" + page.file,
			"dst: /usr/share/man/" + page.dir + "/krot.1",
		} {
			if !strings.Contains(text, want) {
				t.Errorf("nfpm.yaml misses %q", want)
			}
		}
	}
}

func sectionCount(text string) int {
	count := 0
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, ".SH ") {
			count++
		}
	}
	return count
}
