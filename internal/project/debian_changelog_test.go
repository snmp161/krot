package project

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/snmp161/krot/internal/execx"
)

func renderDebianChangelog(t *testing.T, changes string) (string, error) {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{"scripts", "packaging"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	source, err := os.ReadFile("../../scripts/debian-changelog.sh")
	if err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(root, "scripts", "debian-changelog.sh")
	if err := os.WriteFile(script, source, 0o600); err != nil {
		t.Fatal(err)
	}
	fixture := `- semver: 0.0.1
  date: 2026-01-02T03:04:05Z
  packager: Test Packager <test@example.com>
  changes:
` + changes
	if err := os.WriteFile(filepath.Join(root, "packaging", "changelog.yaml"), []byte(fixture), 0o600); err != nil {
		t.Fatal(err)
	}
	result, err := (execx.OS{}).Run("bash", script)
	if err != nil {
		return string(result.Stderr), err
	}
	file, err := os.Open(filepath.Join(root, "dist", "changelog.Debian.gz")) // #nosec G304 -- test fixture in t.TempDir
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	var out strings.Builder
	if _, err := io.Copy(&out, reader); err != nil {
		t.Fatal(err)
	}
	return out.String(), nil
}

// SPEC 9.4: однострочные YAML scalar в кавычках попадают в Debian changelog без синтаксических кавычек.
func TestDebianChangelogQuotedNotes(t *testing.T) {
	out, err := renderDebianChangelog(t, `    - commit: abc1234
      note: "Fixed parser: removed \"quotes\" and kept \\slash"
    - commit: def5678
      note: 'Fixed parser: it''s stable'
    - commit: 123abcd
      note: Plain note stays unchanged
`)
	if err != nil {
		t.Fatalf("render changelog: %v\n%s", err, out)
	}
	for _, want := range []string{
		`  * Fixed parser: removed "quotes" and kept \slash`,
		`  * Fixed parser: it's stable`,
		"  * Plain note stays unchanged",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("changelog misses %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, `* "Fixed`) || strings.Contains(out, `* 'Fixed`) {
		t.Fatalf("YAML quotes leaked into changelog:\n%s", out)
	}
}

// SPEC 9.4: незавершённая кавычка — ошибка, а не повреждённый changelog.
func TestDebianChangelogRejectsMalformedQuote(t *testing.T) {
	out, err := renderDebianChangelog(t, `    - commit: abc1234
      note: "Unterminated note
`)
	if err == nil {
		t.Fatalf("malformed quote unexpectedly succeeded:\n%s", out)
	}
}
