package project

import (
	"strings"
	"testing"
)

// SPEC 9.3 T39: PR template сохраняет процесс SPEC/TDD, проверки и безопасность.
func TestPullRequestTemplate(t *testing.T) {
	text := readRepoFile(t, ".github/pull_request_template.md")
	for _, want := range []string{
		"## Summary",
		"docs/SPEC.md",
		"Task: T",
		"make check",
		"make integration",
		"make package-preflight",
		"private keys",
		"README",
		"golden",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("PR template misses %q", want)
		}
	}
}

// SPEC 9.3 T39: issue form собирает диагностику и требует удалить секреты.
func TestBugIssueTemplate(t *testing.T) {
	text := readRepoFile(t, ".github/ISSUE_TEMPLATE/bug.yml")
	for _, want := range []string{
		"name: Bug report",
		"labels:",
		"\"bug\"",
		"krot --version",
		"Operating system",
		"Architecture",
		"Tunnel type",
		"Steps to reproduce",
		"Expected behavior",
		"Actual behavior",
		"private keys",
		"required: true",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("bug issue template misses %q", want)
		}
	}
}
