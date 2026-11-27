package project

import (
	"strings"
	"testing"
)

// SPEC 9.5 T46: один детерминированный build path используется локально и в release.
func TestReproducibleBuildPipeline(t *testing.T) {
	makefile := readRepoFile(t, "Makefile")
	for _, want := range []string{"SOURCE_DATE_EPOCH", "git show -s --format=%ct", "-buildvcs=false", "repro-check:"} {
		if !strings.Contains(makefile, want) {
			t.Errorf("Makefile misses reproducibility requirement %q", want)
		}
	}

	build := readRepoFile(t, "scripts/build-release.sh")
	for _, want := range []string{
		"SOURCE_DATE_EPOCH", "git show -s --format=%ct", "-buildvcs=false",
		"scripts/compress-man.sh", "nfpm", "sha256sum -- LICENSE THIRD_PARTY_NOTICES krot-linux-*",
	} {
		if !strings.Contains(build, want) {
			t.Errorf("build-release.sh misses %q", want)
		}
	}

	release := readRepoFile(t, ".github/workflows/release.yml")
	if !strings.Contains(release, "scripts/build-release.sh") {
		t.Error("release does not use the shared reproducible build path")
	}
	if strings.Contains(release, "date -u +%Y-%m-%dT%H:%M:%SZ") {
		t.Error("release still embeds the current time")
	}
	if !strings.Contains(readRepoFile(t, ".github/workflows/ci.yml"), "make repro-check") {
		t.Error("CI lacks the independent double-build gate")
	}
	if !strings.Contains(readRepoFile(t, "packaging/nfpm.yaml"), "buildhost: reproducible.invalid") {
		t.Error("RPM build host is not deterministic")
	}

	check := readRepoFile(t, "scripts/reproducible-check.sh")
	for _, want := range []string{"git clone", "cmp", "TZ=UTC", "TZ=Pacific/Honolulu", "SHA256SUMS"} {
		if !strings.Contains(check, want) {
			t.Errorf("reproducible-check.sh misses %q", want)
		}
	}
}

// SPEC 9.2, 9.5: completion генерируется host-toolchain, а не release-бинарем чужой архитектуры.
func TestCompletionUsesHostToolchain(t *testing.T) {
	build := readRepoFile(t, "scripts/build-release.sh")
	if strings.Contains(build, "dist/krot-linux-amd64 --completion") {
		t.Error("build-release.sh executes the amd64 release binary for completion")
	}
	for _, want := range []string{
		"go run ./cmd/krot --completion bash",
		"go run ./cmd/krot --completion zsh",
	} {
		if !strings.Contains(build, want) {
			t.Errorf("build-release.sh misses host completion command %q", want)
		}
	}
}

// SPEC 9.5 T46: вычисление timestamp не маскирует ошибку git через export (SC2155).
func TestSourceDateEpochExportDoesNotMaskErrors(t *testing.T) {
	for _, name := range []string{".github/workflows/ci.yml", ".github/workflows/release.yml"} {
		if strings.Contains(readRepoFile(t, name), `export SOURCE_DATE_EPOCH="$(`) {
			t.Errorf("%s combines export and command substitution", name)
		}
	}
}
