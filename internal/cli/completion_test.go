package cli

import (
	"slices"
	"strings"
	"testing"

	"github.com/snmp161/krot/internal/errs"
)

// SPEC 4.6: --completion — отдельный режим, несовместимый с другими флагами.
func TestCompletionMode(t *testing.T) {
	for _, args := range [][]string{{"--completion", "bash"}, {"--completion=zsh"}} {
		c, err := Parse(args)
		if err != nil {
			t.Fatalf("%q: %v", args, err)
		}
		want := "bash"
		if args[0] == "--completion=zsh" {
			want = "zsh"
		}
		if c.Completion != want {
			t.Errorf("%q: completion = %q; want %q", args, c.Completion, want)
		}
	}

	for _, args := range [][]string{
		{"--completion"},
		{"--completion", "bash", "--list"},
		{"--completion=zsh", "--version"},
	} {
		if got := code(parseErr(args)); got != errs.Usage {
			t.Errorf("%q: code %d; want %d", args, got, errs.Usage)
		}
	}
	if got := code(parseErr([]string{"--completion", "fish"})); got != errs.Invalid {
		t.Errorf("unsupported shell: code %d; want %d", got, errs.Invalid)
	}

	c, err := Parse([]string{"--completion", "bash", "--help"})
	if err != nil || !c.Help {
		t.Errorf("--help must retain priority: %+v, %v", c, err)
	}
}

// SPEC 4.6: варианты автодополнения строятся из матрицы 4.3.
func TestCompletionOptions(t *testing.T) {
	initial := completionOptions("")
	for _, action := range allActions {
		if !slices.Contains(initial, "--"+string(action)) {
			t.Errorf("initial options do not contain --%s", action)
		}
	}
	for _, option := range []string{"--help", "--version", "--completion"} {
		if !slices.Contains(initial, option) {
			t.Errorf("initial options do not contain %s", option)
		}
	}

	for _, action := range allActions {
		options := completionOptions(action)
		for _, p := range paramTable {
			got := slices.Contains(options, "--"+p.name)
			want := p.matrix[action] != forbidden
			if got != want {
				t.Errorf("--%s --%s: present %v; want %v", action, p.name, got, want)
			}
		}
		if !slices.Contains(options, "--help") {
			t.Errorf("--%s options do not contain --help", action)
		}
		for _, other := range allActions {
			if slices.Contains(options, "--"+string(other)) {
				t.Errorf("--%s options contain another action --%s", action, other)
			}
		}
	}
}

// SPEC 4.6: генерируются оба поддерживаемых формата скриптов.
func TestCompletionScripts(t *testing.T) {
	for _, shell := range []string{"bash", "zsh"} {
		script, err := CompletionScript(shell)
		if err != nil {
			t.Fatalf("%s: %v", shell, err)
		}
		marker := "complete -F _krot krot"
		if shell == "zsh" {
			marker = "#compdef krot"
		}
		if !strings.Contains(script, marker) || !strings.HasSuffix(script, "\n") {
			t.Errorf("%s script is incomplete:\n%s", shell, script)
		}
		for _, action := range allActions {
			if !strings.Contains(script, "--"+string(action)) {
				t.Errorf("%s script does not contain --%s", shell, action)
			}
		}
		for _, p := range paramTable {
			if !strings.Contains(script, "--"+p.name) {
				t.Errorf("%s script does not contain --%s", shell, p.name)
			}
		}
	}
	if _, err := CompletionScript("fish"); err == nil {
		t.Error("unsupported shell did not return an error")
	}
}
