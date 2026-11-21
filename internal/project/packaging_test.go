package project

import (
	"path/filepath"
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

// SPEC 8.4, 9.4 T47: policy gates проверяют реальные пакеты и полный lifecycle.
func TestPackageComplianceGates(t *testing.T) {
	for _, name := range []string{"scripts/package-preflight.sh", ".github/workflows/release.yml"} {
		text := readRepoFile(t, name)
		for _, want := range []string{"scripts/package-lint.sh", "scripts/package-lifecycle.sh"} {
			if !strings.Contains(text, want) {
				t.Errorf("%s misses package gate %q", name, want)
			}
		}
	}
	lint := readRepoFile(t, "scripts/package-lint.sh")
	for _, want := range []string{"lintian --fail-on error,warning", "rpmlint --strict", "packaging/rpmlintrc"} {
		if !strings.Contains(lint, want) {
			t.Errorf("package-lint.sh misses %q", want)
		}
	}
	lifecycle := readRepoFile(t, "scripts/package-lifecycle.sh")
	for _, want := range []string{"apt-get install", "apt-get install -y --reinstall", "apt-get purge", "dnf install", "dnf upgrade", "dnf reinstall", "dnf remove"} {
		if !strings.Contains(lifecycle, want) {
			t.Errorf("package-lifecycle.sh misses %q", want)
		}
	}

	pkg := readRepoFile(t, "packaging/nfpm.yaml")
	for _, want := range []string{"release: 1", "/usr/share/doc/krot/README.md", "/usr/share/lintian/overrides/krot"} {
		if !strings.Contains(pkg, want) {
			t.Errorf("nfpm.yaml misses distro compliance entry %q", want)
		}
	}
	changelog := readRepoFile(t, "packaging/changelog.yaml")
	for _, want := range []string{"distributions:", "stable", "urgency: low"} {
		if !strings.Contains(changelog, want) {
			t.Errorf("changelog.yaml misses Debian syntax %q", want)
		}
	}
}

// SPEC 4.4, 6.13 T47: документация не ограничивает Keenetic только WireGuard.
func TestKeeneticDocumentation(t *testing.T) {
	wrong := []string{
		"wg only", "только wg", "solo wg", "somente wg", "wg uniquement",
		"nur wg", "wg فقط", "仅限 wg", "僅限 wg",
	}
	readmes, err := filepath.Glob("../../README*.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range readmes {
		text := readRepoFile(t, filepath.Base(name))
		for _, phrase := range wrong {
			if strings.Contains(text, phrase) {
				t.Errorf("%s still contains incorrect Keenetic limitation %q", name, phrase)
			}
		}
	}
	integration := readRepoFile(t, "test/integration/integration_test.go")
	if strings.Contains(integration, "для AmneziaWG-клиентов не применим") {
		t.Error("Keenetic integration comment still contradicts AWG support")
	}
}
