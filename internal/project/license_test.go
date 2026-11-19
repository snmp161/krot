package project

import (
	"strings"
	"testing"
)

// SPEC 10 T32: копирайт заменяет шаблон в приложении Apache, а не стоит перед лицензией.
func TestLicenseCopyright(t *testing.T) {
	license := readRepoFile(t, "LICENSE")
	const notice = "Copyright 2026 Sergey Ovchinnikov"
	if strings.HasPrefix(license, notice) {
		t.Fatal("copyright notice must not precede the Apache License text")
	}
	if !strings.Contains(license, "\n   "+notice+"\n") {
		t.Fatalf("LICENSE appendix does not contain %q", notice)
	}
	if strings.Contains(license, "Copyright [yyyy] [name of copyright owner]") {
		t.Fatal("LICENSE appendix still contains the copyright placeholder")
	}
	if strings.Count(license, notice) != 1 {
		t.Fatal("LICENSE must contain exactly one owner copyright notice")
	}
}
