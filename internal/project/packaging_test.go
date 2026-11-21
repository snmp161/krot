package project

import (
	"strings"
	"testing"
)

// SPEC 9.4 T42: оба формата используют один полный текст и curated changelog.
func TestDistributionPackageMetadata(t *testing.T) {
	description := readRepoFile(t, "packaging/description.txt")
	for _, want := range []string{
		"WireGuard and AmneziaWG",
		"without restarting or recreating",
		"single source of truth",
		"QR codes",
		"KeeneticOS",
		"UDP diagnostic",
		"does not configure NAT",
	} {
		if !strings.Contains(description, want) {
			t.Errorf("package description misses %q", want)
		}
	}

	pkg := readRepoFile(t, "packaging/nfpm.yaml")
	for _, want := range []string{
		"description: ${DESCRIPTION}",
		`changelog: "./packaging/changelog.yaml"`,
	} {
		if !strings.Contains(pkg, want) {
			t.Errorf("nfpm.yaml misses %q", want)
		}
	}

	changelog := readRepoFile(t, "packaging/changelog.yaml")
	for _, want := range []string{"semver: 0.7.3", "packager: Sergey Ovchinnikov", "localization", "deletion confirmation"} {
		if !strings.Contains(changelog, want) {
			t.Errorf("package changelog misses %q", want)
		}
	}

	for _, name := range []string{"scripts/package-preflight.sh", ".github/workflows/release.yml"} {
		text := readRepoFile(t, name)
		for _, want := range []string{"scripts/build-release.sh", `DESCRIPTION="$(cat packaging/description.txt)"`} {
			if !strings.Contains(text, want) {
				t.Errorf("%s misses %q", name, want)
			}
		}
	}

	smoke := readRepoFile(t, "scripts/smoke.sh")
	for _, want := range []string{"gzip -t", "dpkg-query", "rpm -q", "changelog.Debian.gz", "THIRD_PARTY_NOTICES"} {
		if !strings.Contains(smoke, want) {
			t.Errorf("smoke.sh misses package metadata check %q", want)
		}
	}
	if strings.Contains(readRepoFile(t, "docs/BACKLOG.md"), "Расширенное описание пакетов") {
		t.Error("completed package description work remains in the backlog")
	}
}
