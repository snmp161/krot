package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// SPEC 10 T31: примеры установки во всех README используют текущий релиз.
func TestReadmeInstallVersion(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "README*.md"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 11 {
		t.Fatalf("found %d README files, want 11", len(files))
	}
	for _, name := range files {
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		text := string(data)
		if got := strings.Count(text, "VERSION=0.7.1"); got != 1 {
			t.Errorf("%s: current installation version occurs %d times", name, got)
		}
		if strings.Contains(text, "VERSION=0.1.0") {
			t.Errorf("%s still uses the obsolete installation version", name)
		}
	}
}
