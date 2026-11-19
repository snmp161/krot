package project

import (
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
