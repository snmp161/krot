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
