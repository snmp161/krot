package cli

import (
	"regexp"
	"strconv"
	"testing"
)

var usageTokenPattern = regexp.MustCompile(`--[a-z0-9-]+`)

// SPEC 4.7 T27: каждая локализованная справка сохраняет полный CLI-интерфейс.
func TestAllUsageTextsPreserveCLIStructure(t *testing.T) {
	want := usageTokens(usageText)
	for language, text := range localizedUsage {
		if got := usageTokens(text); got != want {
			t.Errorf("%s: CLI token structure differs\n got: %q\nwant: %q", language, got, want)
		}
	}
}

func usageTokens(text string) string {
	counts := map[string]int{}
	for _, token := range usageTokenPattern.FindAllString(text, -1) {
		counts[token]++
	}
	out := ""
	for _, token := range usageTokenPattern.FindAllString(usageText, -1) {
		if counts[token] == 0 {
			continue
		}
		out += token + "=" + strconv.Itoa(counts[token]) + "\n"
		delete(counts, token)
	}
	return out
}
