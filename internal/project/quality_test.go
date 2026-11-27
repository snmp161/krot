package project

import (
	"strings"
	"testing"
)

// SPEC 9.1 T36: CI использует общие с локальным запуском quality gates.
func TestQualityWorkflowGates(t *testing.T) {
	ci := readRepoFile(t, ".github/workflows/ci.yml")
	for _, want := range []string{
		"go mod tidy -diff",
		"go mod verify",
		"make test",
		"make vulncheck",
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

// SPEC 9.1 T36: локальный check включает coverage, govulncheck и bounded fuzzing.
func TestQualityMakeTargets(t *testing.T) {
	makefile := readRepoFile(t, "Makefile")
	for _, want := range []string{
		"mod-check:", "fuzz-smoke:", "vulncheck:", "shellcheck:", "actionlint:", "package-preflight:",
		"coverprofile=coverage.out", "go tool cover", "t >= 90.0", "govulncheck@v1.9.0",
		"check: fmt-check vet mod-check test vulncheck fuzz-smoke",
		"scripts/fuzz-smoke.sh",
	} {
		if !strings.Contains(makefile, want) {
			t.Errorf("Makefile misses %q", want)
		}
	}
}

// SPEC 9.1: fuzz-smoke обнаруживает targets автоматически, без ручного списка имён.
func TestFuzzSmokeDiscoversTargets(t *testing.T) {
	script := readRepoFile(t, "scripts/fuzz-smoke.sh")
	for _, want := range []string{"go list ./...", "-list '^Fuzz'", "-fuzz=", "-fuzztime=10s"} {
		if !strings.Contains(script, want) {
			t.Errorf("fuzz-smoke.sh misses %q", want)
		}
	}
	for _, hardcoded := range []string{"FuzzParse", "FuzzValidateCPS", "FuzzParseVersion", "FuzzParseDump", "FuzzConfigRoundTrip"} {
		if strings.Contains(script, hardcoded) {
			t.Errorf("fuzz-smoke.sh hardcodes target %q", hardcoded)
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
