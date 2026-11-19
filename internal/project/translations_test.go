package project

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

type readmeLanguage struct {
	name   string
	native string
}

var readmeLanguages = []readmeLanguage{
	{"README.md", "English"},
	{"README.ru.md", "Русский"},
	{"README.zh.md", "简体中文"},
	{"README.zh-Hant.md", "繁體中文"},
	{"README.es.md", "Español"},
	{"README.pt.md", "Português"},
	{"README.ko.md", "한국어"},
	{"README.ja.md", "日本語"},
	{"README.ar.md", "العربية"},
	{"README.fr.md", "Français"},
	{"README.de.md", "Deutsch"},
}

// SPEC 10 T33, T34: все README связывают все доступные языковые версии.
func TestReadmeLanguageLinks(t *testing.T) {
	for _, current := range readmeLanguages {
		text := readRepoFile(t, current.name)
		for _, target := range readmeLanguages {
			want := "[" + target.native + "](" + target.name + ")"
			if current.name == target.name {
				want = "**" + target.native + "**"
			}
			if !strings.Contains(text, want) {
				t.Errorf("%s misses %q", current.name, want)
			}
		}
	}
}

// SPEC 10 T33, T34: переводы сохраняют техническую структуру английского README.
func TestFrenchGermanReadmeStructure(t *testing.T) {
	english := readRepoFile(t, "README.md")
	for _, name := range []string{"README.fr.md", "README.de.md"} {
		translated := readRepoFile(t, name)
		if got, want := headingLevels(translated), headingLevels(english); got != want {
			t.Errorf("%s heading structure differs: got %q, want %q", name, got, want)
		}
		if got, want := tableShapes(translated), tableShapes(english); got != want {
			t.Errorf("%s table structure differs", name)
		}
		if got, want := inlineCode(english), inlineCode(translated); got != want {
			t.Errorf("%s inline code differs", name)
		}
		if got, want := technicalCodeLines(english), technicalCodeLines(translated); got != want {
			t.Errorf("%s command or configuration content differs", name)
		}
	}
}

// SPEC 10 T33, T34: локальные ссылки во всех README указывают на файлы.
func TestReadmeLocalLinks(t *testing.T) {
	link := regexp.MustCompile(`\]\(([^)]+)\)`)
	for _, language := range readmeLanguages {
		text := readRepoFile(t, language.name)
		for _, match := range link.FindAllStringSubmatch(text, -1) {
			target := strings.Split(match[1], "#")[0]
			if target == "" || strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") {
				continue
			}
			if _, err := os.Stat(filepath.Join("..", "..", target)); err != nil {
				t.Errorf("%s: broken local link %q: %v", language.name, target, err)
			}
		}
	}
}

func headingLevels(text string) string {
	var out strings.Builder
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "#") {
			level := len(line) - len(strings.TrimLeft(line, "#"))
			out.WriteByte(byte('0' + level))
		}
	}
	return out.String()
}

func tableShapes(text string) string {
	var out strings.Builder
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "|") {
			out.WriteString(strings.Repeat("|", strings.Count(line, "|")))
			out.WriteByte('\n')
		}
	}
	return out.String()
}

func inlineCode(text string) string {
	re := regexp.MustCompile("`[^`]+`")
	return strings.Join(re.FindAllString(text, -1), "\n")
}

func technicalCodeLines(text string) string {
	var out []string
	language := ""
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			if language == "" {
				language = strings.TrimPrefix(trimmed, "```")
			} else {
				language = ""
			}
			continue
		}
		if language != "sh" && language != "ini" {
			continue
		}
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.Contains(trimmed, "#") {
			continue
		}
		out = append(out, trimmed)
	}
	return strings.Join(out, "\n")
}
