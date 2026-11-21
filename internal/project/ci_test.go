package project

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// SPEC 9 T29: Actions, инструменты, timeouts и concurrency закреплены в workflows.
func TestReproducibleWorkflows(t *testing.T) {
	uses := regexp.MustCompile(`(?m)^\s*(?:-\s+)?uses:\s*(\S+)`)
	commit := regexp.MustCompile(`^[0-9a-f]{40}$`)
	for _, name := range []string{".github/workflows/ci.yml", ".github/workflows/release.yml"} {
		text := readRepoFile(t, name)
		matches := uses.FindAllStringSubmatch(text, -1)
		if len(matches) == 0 {
			t.Fatalf("%s: no actions found", name)
		}
		for _, match := range matches {
			action := match[1]
			if strings.HasPrefix(action, "./") {
				continue
			}
			_, ref, ok := strings.Cut(action, "@")
			if !ok || !commit.MatchString(ref) {
				t.Errorf("%s: action %q is not pinned to a full commit SHA", name, action)
			}
		}
		if strings.Contains(text, "@latest") {
			t.Errorf("%s: contains an unpinned Go tool", name)
		}
		if got, want := strings.Count(text, "timeout-minutes:"), strings.Count(text, "runs-on:"); got != want {
			t.Errorf("%s: %d timeouts for %d runner jobs", name, got, want)
		}
	}
	ci := readRepoFile(t, ".github/workflows/ci.yml")
	for _, want := range []string{"concurrency:", "cancel-in-progress: true"} {
		if !strings.Contains(ci, want) {
			t.Errorf("ci.yml misses %q", want)
		}
	}
}

// SPEC 9.1 T29: теги AmneziaWG сверяются с неизменяемыми коммитами и ключами кэша.
func TestAmneziaSourceRevisions(t *testing.T) {
	ci := readRepoFile(t, ".github/workflows/ci.yml")
	for _, want := range []string{
		"AWG_TOOLS_TAG: v3.1.20260812",
		"AWG_TOOLS_REV: ee0f0a9aa34ff0a0da4b3433b9512781cfe02843",
		"AWG_GO_TAG: v3.1.20260828",
		"AWG_GO_REV: b5928efb6ca19f0153958460c3d141f04abc5c2e",
		"${{ env.AWG_TOOLS_REV }}",
		"${{ env.AWG_GO_REV }}",
	} {
		if !strings.Contains(ci, want) {
			t.Errorf("ci.yml misses %q", want)
		}
	}
	if got := strings.Count(ci, "git rev-parse HEAD"); got < 2 {
		t.Errorf("ci.yml verifies only %d AmneziaWG revisions", got)
	}
}

// SPEC 9.2, 9.3 T44: релиз допускается только от проверенного annotated тега из main.
func TestReleaseTagPolicy(t *testing.T) {
	release := readRepoFile(t, ".github/workflows/release.yml")
	for _, want := range []string{
		"validate:",
		"needs: validate",
		"git cat-file -t",
		"merge-base --is-ancestor",
		"tagged_commit",
		"semver=",
		"--draft",
		"--draft=false",
	} {
		if !strings.Contains(release, want) {
			t.Errorf("release.yml misses tag/release enforcement %q", want)
		}
	}
}

// SPEC 9.2 T45: подписанное provenance и его проверка являются обязательным gate.
func TestArtifactProvenance(t *testing.T) {
	release := readRepoFile(t, ".github/workflows/release.yml")
	for _, want := range []string{
		"actions/attest-build-provenance@4d101475d8b20a2381f78447822ac1eab6504dd8",
		"id-token: write",
		"attestations: write",
		"attestations: read",
		"dist/krot-linux-*",
		"dist/*.deb",
		"dist/*.rpm",
		"dist/SHA256SUMS",
		"gh attestation verify",
	} {
		if !strings.Contains(release, want) {
			t.Errorf("release.yml misses provenance requirement %q", want)
		}
	}
	verify := strings.Index(release, "gh attestation verify")
	publish := strings.Index(release, "gh release create")
	if verify < 0 || publish < 0 || verify > publish {
		t.Error("artifact attestations must be verified before release creation")
	}
	if strings.Contains(release, "sha256sum -- * > SHA256SUMS") {
		t.Error("checksum manifest is still generated after packaging instead of before attestation")
	}
	if !strings.Contains(readRepoFile(t, "scripts/build-release.sh"), "sha256sum -- LICENSE THIRD_PARTY_NOTICES krot-linux-*") {
		t.Error("shared build does not generate the checksum manifest before attestation")
	}

	readmes, err := filepath.Glob("../../README*.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range readmes {
		if !strings.Contains(readRepoFile(t, filepath.Base(name)), "gh attestation verify SHA256SUMS --repo snmp161/krot") {
			t.Errorf("%s does not document cryptographic attestation verification", name)
		}
	}
}
