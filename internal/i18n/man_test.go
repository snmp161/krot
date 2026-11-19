package i18n

import (
	"os"
	"strings"
	"testing"
)

// SPEC 4.7: английская и русская man-страницы описывают один интерфейс.
func TestManPages(t *testing.T) {
	pages := map[string][]string{
		"krot.1":    {".SH NAME", ".SH SYNOPSIS", ".SH DESCRIPTION", ".SH ACTIONS", ".SH OPTIONS", ".SH ENVIRONMENT", ".SH \"EXIT STATUS\"", ".SH FILES", ".SH EXAMPLES"},
		"krot.ru.1": {".SH ИМЯ", ".SH СИНТАКСИС", ".SH ОПИСАНИЕ", ".SH ДЕЙСТВИЯ", ".SH ПАРАМЕТРЫ", ".SH ОКРУЖЕНИЕ", ".SH \"КОД ВЫХОДА\"", ".SH ФАЙЛЫ", ".SH ПРИМЕРЫ"},
	}
	flags := []string{
		"list", "create", "delete", "up", "down", "add-client", "delete-client",
		"lock-client", "unlock-client", "list-client", "doctor", "udp-listener", "udp-checker",
		"help", "version", "type", "interface", "ip", "port", "endpoint", "address",
		"mtu", "dns", "i1", "mimic", "id", "allowed-ips", "psk", "qr", "amnezia-key",
		"keenetic", "json", "yes", "timeout", "count", "echo-size", "workers", "verbose",
	}
	for name, sections := range pages {
		data, err := os.ReadFile("../../packaging/" + name)
		if err != nil {
			t.Errorf("read %s: %v", name, err)
			continue
		}
		text := string(data)
		if !strings.HasPrefix(text, ".TH KROT 1") {
			t.Errorf("%s: missing man title", name)
		}
		for _, section := range sections {
			if !strings.Contains(text, section) {
				t.Errorf("%s: missing section %q", name, section)
			}
		}
		plainText := strings.ReplaceAll(text, `\-`, "-")
		for _, flag := range flags {
			if !strings.Contains(plainText, "--"+flag) {
				t.Errorf("%s: missing --%s", name, flag)
			}
		}
	}
}

// SPEC 4.7, 9.2: nfpm устанавливает обе man-страницы в системные каталоги.
func TestManPagesPackaging(t *testing.T) {
	data, err := os.ReadFile("../../packaging/nfpm.yaml")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		"src: ./packaging/krot.1",
		"dst: /usr/share/man/man1/krot.1",
		"src: ./packaging/krot.ru.1",
		"dst: /usr/share/man/ru/man1/krot.1",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("nfpm.yaml misses %q", want)
		}
	}
}
