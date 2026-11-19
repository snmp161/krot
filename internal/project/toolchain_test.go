package project

import (
	"strings"
	"testing"
)

// SPEC 9 T30: локальные и CI-проверки используют Go из go.mod и один staticcheck.
func TestToolchainFromGoMod(t *testing.T) {
	if goMod := readRepoFile(t, "go.mod"); !strings.Contains(goMod, "go 1.27.2") {
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
