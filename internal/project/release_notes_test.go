package project

import (
	"strings"
	"testing"
)

// SPEC 9.2 T43: curated notes управляют репозиторным, GitHub и пакетными changelog.
func TestCuratedReleaseNotes(t *testing.T) {
	changelog := readRepoFile(t, "CHANGELOG.md")
	for _, want := range []string{
		"0.7.3", "2026-11-20", "fb2fe73", "cb28408", "1b31e6a",
		"Added application and manual page localization",
		"Fixed tunnel deletion confirmation",
	} {
		if !strings.Contains(changelog, want) {
			t.Errorf("CHANGELOG.md misses %q", want)
		}
	}
	if strings.Contains(changelog, "()") {
		t.Error("CHANGELOG.md contains empty commit references")
	}

	for _, name := range []string{"AGENTS.md", "CONTRIBUTING.md"} {
		text := readRepoFile(t, name)
		for _, want := range []string{"packaging/changelog.yaml", "CHANGELOG.md", "release notes"} {
			if !strings.Contains(text, want) {
				t.Errorf("%s misses release-notes rule %q", name, want)
			}
		}
	}

	release := readRepoFile(t, ".github/workflows/release.yml")
	for _, want := range []string{"chglog@v0.7.6", "scripts/release-notes.sh", "--notes-file"} {
		if !strings.Contains(release, want) {
			t.Errorf("release.yml misses %q", want)
		}
	}
	if strings.Contains(release, "--generate-notes") {
		t.Error("release.yml must not replace curated notes with generated commit notes")
	}

	script := readRepoFile(t, "scripts/release-notes.sh")
	for _, want := range []string{"packaging/changelog.yaml", "CHANGELOG.md", "dist/release-notes.md", "taggerdate"} {
		if !strings.Contains(script, want) {
			t.Errorf("release-notes.sh misses validation %q", want)
		}
	}
	if !strings.Contains(readRepoFile(t, ".github/workflows/ci.yml"), "make changelog-check") {
		t.Error("CI does not check that CHANGELOG.md is current")
	}
	for _, want := range []string{"changelog:", "changelog-check:", "chglog@v0.7.6"} {
		if !strings.Contains(readRepoFile(t, "Makefile"), want) {
			t.Errorf("Makefile misses %q", want)
		}
	}
}
