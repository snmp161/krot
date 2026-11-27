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

// jobBlock возвращает текст job'а workflow от "  <name>:" до следующего job'а верхнего уровня.
func jobBlock(t *testing.T, workflow, name string) string {
	t.Helper()
	start := strings.Index(workflow, "\n  "+name+":\n")
	if start < 0 {
		t.Fatalf("workflow has no job %q", name)
	}
	rest := workflow[start+1:]
	lines := strings.SplitAfter(rest, "\n")
	end := len(rest)
	offset := len(lines[0])
	for _, l := range lines[1:] {
		if strings.HasPrefix(l, "  ") && !strings.HasPrefix(l, "   ") && strings.HasSuffix(strings.TrimSpace(l), ":") {
			end = offset
			break
		}
		offset += len(l)
	}
	return rest[:end]
}

// SPEC 9.2 T43, T46: release notes, сформированные из curated записи, доходят до
// публикации — общая сборка очищает dist/, поэтому notes формируются после неё.
func TestReleaseNotesSurviveSharedBuild(t *testing.T) {
	build := jobBlock(t, readRepoFile(t, ".github/workflows/release.yml"), "build")
	shared := strings.Index(build, "scripts/build-release.sh")
	notes := strings.Index(build, `scripts/release-notes.sh "$GITHUB_REF_NAME"`)
	upload := strings.Index(build, "actions/upload-artifact")
	if shared < 0 || notes < 0 || upload < 0 {
		t.Fatalf("release build job misses shared build (%d), release notes (%d) or upload (%d)", shared, notes, upload)
	}
	if notes < shared {
		t.Error("release notes are generated before scripts/build-release.sh, which recreates dist/ and deletes dist/release-notes.md")
	}
	if notes > upload {
		t.Error("release notes are generated after dist/ is uploaded")
	}
	if !strings.Contains(readRepoFile(t, "scripts/build-release.sh"), "rm -rf dist\n") {
		t.Fatal("test assumption changed: scripts/build-release.sh no longer recreates dist/")
	}
}

// SPEC 9.2 T44, T43: проверки annotated tag и даты curated записи не зависят от того,
// сохранил ли shallow checkout объект тега: тег явно загружается из origin.
func TestReleaseFetchesAnnotatedTagObject(t *testing.T) {
	release := readRepoFile(t, ".github/workflows/release.yml")
	for job, check := range map[string]string{
		"validate": "git cat-file -t",
		"build":    `scripts/release-notes.sh "$GITHUB_REF_NAME"`,
	} {
		text := jobBlock(t, release, job)
		fetch := strings.Index(text, `git fetch --no-tags --force origin "refs/tags/`)
		use := strings.Index(text, check)
		if fetch < 0 || use < 0 || fetch > use {
			t.Errorf("%s job must fetch the annotated tag object before %q", job, check)
		}
	}
}

// SPEC 9.2 T51: curated запись v0.7.4 описывает изменения поведения и совместимости
// и указывает на текущую версию в шаблоне bug report.
func TestReleaseNotes074(t *testing.T) {
	changelog := readRepoFile(t, "CHANGELOG.md")
	// chglog начинает файл с пустой строки; первой записью должна быть 0.7.4.
	if !strings.HasPrefix(strings.TrimLeft(changelog, "\n"), "0.7.4\n") {
		t.Fatalf("CHANGELOG.md must start with the 0.7.4 entry:\n%.200s", changelog)
	}
	for _, want := range []string{"quadrant", "symlink", "--keenetic", "--udp-listener", "port_start", "attestation"} {
		if !strings.Contains(changelog, want) {
			t.Errorf("0.7.4 release notes miss %q", want)
		}
	}
	if !strings.Contains(readRepoFile(t, ".github/ISSUE_TEMPLATE/bug.yml"), "krot v0.7.4 (") {
		t.Error("bug report template does not show the current version")
	}
}
