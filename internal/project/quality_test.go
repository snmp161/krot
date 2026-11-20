package project

import (
	"strings"
	"testing"
)

// SPEC 9.1 T36: CI содержит module, fuzz, script/workflow и package gates.
func TestQualityWorkflowGates(t *testing.T) {
	ci := readRepoFile(t, ".github/workflows/ci.yml")
	for _, want := range []string{
		"go mod tidy -diff",
		"go mod verify",
		"go test -race -count=1 -shuffle=on -coverprofile=coverage.out ./...",
		"shellcheck",
		"actionlint",
		"make fuzz-smoke",
		"nfpm",
		"ubuntu:24.04",
		"almalinux:9",
	} {
		if !strings.Contains(ci, want) {
			t.Errorf("ci.yml misses %q", want)
		}
	}
}

// SPEC 9.1 T36: основные quality gates доступны для локального запуска.
func TestQualityMakeTargets(t *testing.T) {
	makefile := readRepoFile(t, "Makefile")
	for _, want := range []string{
		"mod-check:", "fuzz-smoke:", "shellcheck:", "actionlint:", "package-preflight:",
		"-fuzztime=10s", "FuzzParse", "FuzzValidateCPS", "FuzzParseVersion", "FuzzParseDump", "FuzzConfigRoundTrip",
	} {
		if !strings.Contains(makefile, want) {
			t.Errorf("Makefile misses %q", want)
		}
	}
}

// SPEC 8.3 T36: listener/checker проверяются в отдельных netns, включая DROP.
func TestUDPIntegrationCoveragePresent(t *testing.T) {
	integration := readRepoFile(t, "test/integration/integration_test.go")
	for _, want := range []string{"func TestUDPPortDetect", "iptables", `"filtered"`, `"ok"`} {
		if !strings.Contains(integration, want) {
			t.Errorf("integration_test.go misses %q", want)
		}
	}
}
