package project

import (
	"path/filepath"
	"strings"
	"testing"
)

// SPEC 9 T30: локальные и CI-проверки используют Go из go.mod и один staticcheck.
func TestToolchainFromGoMod(t *testing.T) {
	if goMod := readRepoFile(t, "go.mod"); !strings.Contains(goMod, "go 1.27.4") {
		t.Fatal("go.mod does not contain the expected pinned Go version")
	}
	makefile := readRepoFile(t, "Makefile")
	for _, want := range []string{"GO_VERSION", "export GOTOOLCHAIN := go$(GO_VERSION)", "GOFMT"} {
		if !strings.Contains(makefile, want) {
			t.Errorf("Makefile misses %q for go.mod toolchain selection", want)
		}
	}
	for _, name := range []string{"Makefile", ".github/workflows/ci.yml"} {
		text := readRepoFile(t, name)
		if strings.Contains(text, "GOTOOLCHAIN=local") {
			t.Errorf("%s overrides the go.mod toolchain", name)
		}
		if !strings.Contains(text, "honnef.co/go/tools/cmd/staticcheck@v0.8.2") {
			t.Errorf("%s does not pin staticcheck v0.8.2", name)
		}
	}
}

// SPEC 9.1: форматирование в CI и инструкциях выполняется через toolchain из go.mod.
func TestPinnedGofmtEntryPoint(t *testing.T) {
	ci := readRepoFile(t, ".github/workflows/ci.yml")
	if !strings.Contains(ci, "make fmt-check") {
		t.Error("ci.yml must use make fmt-check")
	}
	if strings.Contains(ci, "gofmt -l .") {
		t.Error("ci.yml still calls gofmt from PATH")
	}
	for _, name := range []string{"AGENTS.md", "CONTRIBUTING.md"} {
		text := readRepoFile(t, name)
		if !strings.Contains(text, "make fmt-check") {
			t.Errorf("%s does not recommend make fmt-check", name)
		}
		if strings.Contains(text, "gofmt -l .") {
			t.Errorf("%s still recommends gofmt from PATH", name)
		}
	}
}

// SPEC 10 T40: актуальный Go закреплён во всех toolchain references без исключений.
func TestCurrentGoVersionPinnedEverywhere(t *testing.T) {
	files := []string{"go.mod", "scripts/collect-versions.sh", "test/integration/Dockerfile"}
	readmes, err := filepath.Glob("../../README*.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range readmes {
		files = append(files, filepath.Base(name))
	}
	for _, name := range files {
		text := readRepoFile(t, name)
		if !strings.Contains(text, "1.27.4") {
			t.Errorf("%s does not reference Go 1.27.4", name)
		}
		if strings.Contains(text, "1.27.2") {
			t.Errorf("%s still references obsolete Go 1.27.2", name)
		}
	}
}
