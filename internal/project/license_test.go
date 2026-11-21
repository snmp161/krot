package project

import (
	"path/filepath"
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

// SPEC 9.2 T41: все бинарные дистрибутивы несут обязательные third-party notices.
func TestThirdPartyLicenseCompliance(t *testing.T) {
	for _, name := range []string{"THIRD_PARTY_NOTICES", "packaging/copyright"} {
		text := readRepoFile(t, name)
		for _, want := range []string{
			"rsc.io/qr", "Copyright (c) 2009 The Go Authors",
			"Redistribution and use in source and binary forms",
			"Go runtime and standard library", "Copyright 2009 The Go Authors",
		} {
			if !strings.Contains(text, want) {
				t.Errorf("%s misses %q", name, want)
			}
		}
	}

	pkg := readRepoFile(t, "packaging/nfpm.yaml")
	for _, want := range []string{
		"/usr/share/doc/krot/copyright",
		"/usr/share/doc/krot/THIRD_PARTY_NOTICES",
		"type: license",
		"packager: rpm",
		"packager: deb",
	} {
		if !strings.Contains(pkg, want) {
			t.Errorf("nfpm.yaml misses %q", want)
		}
	}

	release := readRepoFile(t, ".github/workflows/release.yml")
	if !strings.Contains(release, "cp LICENSE THIRD_PARTY_NOTICES dist/") {
		t.Error("standalone binaries are not accompanied by license assets")
	}

	agents := readRepoFile(t, "AGENTS.md")
	for _, want := range []string{"third-party component", "license obligations", "THIRD_PARTY_NOTICES"} {
		if !strings.Contains(agents, want) {
			t.Errorf("AGENTS.md misses license rule %q", want)
		}
	}

	readmes, err := filepath.Glob("../../README*.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range readmes {
		text := readRepoFile(t, filepath.Base(name))
		if !strings.Contains(text, `wget "$BASE/LICENSE" "$BASE/THIRD_PARTY_NOTICES"`) {
			t.Errorf("%s does not fetch license materials with standalone binaries", name)
		}
	}
}
