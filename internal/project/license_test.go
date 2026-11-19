package project

import (
	"strings"
	"testing"
)

// SPEC 10 T32: файл лицензии содержит явный копирайт владельца проекта.
func TestLicenseCopyright(t *testing.T) {
	license := readRepoFile(t, "LICENSE")
	const notice = "Copyright 2026 Sergey Ovchinnikov"
	if !strings.HasPrefix(license, notice+"\n\n") {
		t.Fatalf("LICENSE does not start with %q", notice)
	}
	if strings.Count(license, notice) != 1 {
		t.Fatal("LICENSE must contain exactly one owner copyright notice")
	}
}
