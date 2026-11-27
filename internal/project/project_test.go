package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readRepoFile(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(data)
}

// SPEC 10 T28: руководство описывает полный и безопасный процесс разработки.
func TestContributingGuide(t *testing.T) {
	guide := readRepoFile(t, "CONTRIBUTING.md")
	for _, want := range []string{
		"docs/SPEC.md",
		"CODE_OF_CONDUCT.md",
		"SECURITY.md",
		"Apache-2.0",
		"make check",
		"go test -race ./...",
		"make integration",
		"Conventional Commits",
		"Expected: failing",
		"crypto/rand",
		"CGO_ENABLED=0",
		"private keys",
		"man",
		"README",
	} {
		if !strings.Contains(guide, want) {
			t.Errorf("CONTRIBUTING.md misses %q", want)
		}
	}
}

// SPEC 4.7, 6.10, 9.2: руководство для людей не мягче правил для агентов (AGENTS.md):
// сторонние лицензии, коммиты без падающих тестов, незаметность для клиентов и
// curated release notes.
func TestContributingFollowsAgents(t *testing.T) {
	guide := readRepoFile(t, "CONTRIBUTING.md")
	for _, want := range []string{
		"## Third-party licenses",
		"THIRD_PARTY_NOTICES",
		"packaging/copyright",
		"a permissive license does not remove attribution obligations",
		"never commit with failing tests",
		"do not combine several tasks in one commit",
		"(SPEC 6.10)",
		"fake executor",
		"disabled, skipped, or weakened",
		"automatically generated commit list",
		"UTC date must match the intended tag",
	} {
		if !strings.Contains(guide, want) {
			t.Errorf("CONTRIBUTING.md misses AGENTS.md rule %q", want)
		}
	}
}
