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
