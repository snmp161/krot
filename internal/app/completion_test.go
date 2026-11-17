package app

import (
	"strings"
	"testing"
)

// SPEC 4.6: completion работает без root и не обращается к окружению.
func TestCompletionWithoutRoot(t *testing.T) {
	e := newEnv(t)
	e.app.Euid = func() int { return 1000 }
	for _, shell := range []string{"bash", "zsh"} {
		e.mustRun(0, "--completion", shell)
		marker := "complete -F _krot krot"
		if shell == "zsh" {
			marker = "#compdef krot"
		}
		if !strings.Contains(e.stdout.String(), marker) {
			t.Errorf("%s output:\n%s", shell, e.stdout)
		}
		if e.stderr.String() != "" {
			t.Errorf("%s stderr: %q", shell, e.stderr)
		}
	}
	if len(e.fake.Calls) != 0 {
		t.Fatalf("unexpected calls: %q", e.fake.CommandLines())
	}
}
