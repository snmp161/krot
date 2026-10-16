package awg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func golden(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "versions", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func mustVersion(t *testing.T, s string) Version {
	t.Helper()
	v, err := ParseVersion(s)
	if err != nil {
		t.Fatalf("ParseVersion(%q): %v", s, err)
	}
	return v
}

// SPEC 3.1: разбор версий с суффиксами git describe и релизными суффиксами.
func TestParseVersion(t *testing.T) {
	cases := []struct {
		in   string
		want Version
	}{
		{"1.0.20250901", Version{1, 0, 20250901}},
		{"v3.1.20260812", Version{3, 1, 20260812}},
		{"1.0.20260618-2", Version{1, 0, 20260618}},
		{"v3.0.20260805-3-gabcdef1-dirty", Version{3, 0, 20260805}},
		{"0.2.15", Version{0, 2, 15}},
		{"3.0.20260731-04", Version{3, 0, 20260731}},
	}
	for _, c := range cases {
		got, err := ParseVersion(c.in)
		if err != nil || got != c.want {
			t.Errorf("ParseVersion(%q) = %v, %v; want %v", c.in, got, err, c.want)
		}
	}
	for _, bad := range []string{"", "garbage", "1.0", "1.x.3", "v", "1.0.-1"} {
		if _, err := ParseVersion(bad); err == nil {
			t.Errorf("ParseVersion(%q): expected error", bad)
		}
	}
}

func TestVersionCompare(t *testing.T) {
	a := Version{1, 0, 20250901}
	b := Version{3, 1, 20260812}
	if a.Compare(b) >= 0 || b.Compare(a) <= 0 || a.Compare(a) != 0 {
		t.Fatal("Compare is wrong")
	}
	if (Version{0, 2, 15}).Compare(Version{0, 2, 13}) <= 0 {
		t.Fatal("Compare patch is wrong")
	}
	if got := (Version{3, 1, 20260812}).String(); got != "3.1.20260812" {
		t.Fatalf("String() = %q", got)
	}
}

// SPEC 3.1 шаг 1: `awg --version` (golden-выводы реальных сборок).
func TestParseToolsOutput(t *testing.T) {
	cases := map[string]string{
		"awg-tools-v1.0.20240213-git.txt":     "1.0.20210914",
		"awg-tools-v1.0.20250901-git.txt":     "1.0.20210914",
		"awg-tools-v1.0.20250901-tarball.txt": "1.0.20210914",
		"awg-tools-v3.1.20260812-git.txt":     "3.1.20260812",
		"wg-tools-host.txt":                   "1.0.20210914",
	}
	for file, want := range cases {
		got, err := ParseToolsOutput(golden(t, file))
		if err != nil || got != mustVersion(t, want) {
			t.Errorf("%s: got %v, %v; want %s", file, got, err, want)
		}
	}
	for _, bad := range []string{"", "awg: command not found", "amneziawg-tools - https://amnezia.org"} {
		if _, err := ParseToolsOutput(bad); err == nil {
			t.Errorf("ParseToolsOutput(%q): expected error", bad)
		}
	}
}

// SPEC 3.1 шаг 3: `amneziawg-go --version`.
func TestParseGoOutput(t *testing.T) {
	cases := map[string]string{
		"amneziawg-go-v0.2.15-make.txt":          "0.2.15",
		"amneziawg-go-v3.1.20260828-make.txt":    "3.1.20260828",
		"amneziawg-go-v3.1.20260828-gobuild.txt": "0.0.20250522",
	}
	for file, want := range cases {
		got, err := ParseGoOutput(golden(t, file))
		if err != nil || got != mustVersion(t, want) {
			t.Errorf("%s: got %v, %v; want %s", file, got, err, want)
		}
	}
	if _, err := ParseGoOutput("something else\n"); err == nil {
		t.Error("expected error for unknown output")
	}
}

// SPEC 3.1 шаг 2: /sys/module/amneziawg/version и modinfo -F version.
func TestParseModuleOutput(t *testing.T) {
	cases := map[string]string{
		"module-1.0.0.txt":           "1.0.0",
		"module-1.0.20220627.txt":    "1.0.20220627",
		"module-1.0.20251009.txt":    "1.0.20251009",
		"module-3.0.20260805.txt":    "3.0.20260805",
		"module-3.1.20260812.txt":    "3.1.20260812",
		"modinfo-wireguard-host.txt": "1.0.0",
	}
	for file, want := range cases {
		got, err := ParseModuleOutput(golden(t, file))
		if err != nil || got != mustVersion(t, want) {
			t.Errorf("%s: got %v, %v; want %s", file, got, err, want)
		}
	}
	if _, err := ParseModuleOutput("modinfo: ERROR: Module amneziawg not found.\n"); err == nil {
		t.Error("expected error")
	}
}

// SPEC 3.1 шаг 2: версия netlink-семейства из `genl ctrl get name amneziawg`.
func TestParseGenlOutput(t *testing.T) {
	cases := map[string]int{
		"genl-amneziawg-v1.txt": 1,
		"genl-amneziawg-v2.txt": 2,
		"genl-amneziawg-v3.txt": 3,
		"genl-nlctrl-host.txt":  2,
	}
	for file, want := range cases {
		got, err := ParseGenlOutput(golden(t, file))
		if err != nil || got != want {
			t.Errorf("%s: got %d, %v; want %d", file, got, err, want)
		}
	}
	if _, err := ParseGenlOutput(golden(t, "genl-amneziawg-missing-host.txt")); err == nil {
		t.Error("expected error for missing family")
	}
}

// SPEC 3.1: таблица «тип → минимальная версия» хранится в одном месте.
func TestRequirementTable(t *testing.T) {
	cases := []struct {
		typ    Type
		i1     bool
		gen    string
		tools  string
		module string
		genl   int
		gov    string
	}{
		{TypeAWG, false, "1.0", "1.0.20240213", "", 1, "0.1.0"},
		{TypeAWG, true, "1.5", "1.0.20250901", "1.0.20251009", 2, "0.2.13"},
		{TypeAWG2, false, "2.0", "1.0.20250901", "1.0.20251009", 2, "0.2.15"},
		{TypeAWG2, true, "2.0", "1.0.20250901", "1.0.20251009", 2, "0.2.15"},
		{TypeAWG3, false, "3.1", "3.1.20260812", "3.1.20260812", 3, "3.1.20260812"},
		{TypeAWG3, true, "3.1", "3.1.20260812", "3.1.20260812", 3, "3.1.20260812"},
	}
	for _, c := range cases {
		r := RequirementFor(c.typ, c.i1)
		var module Version
		if c.module != "" {
			module = mustVersion(t, c.module)
		}
		if r.Gen != c.gen || r.Tools != mustVersion(t, c.tools) || r.Module != module ||
			r.Genl != c.genl || r.Go != mustVersion(t, c.gov) {
			t.Errorf("RequirementFor(%s, %v) = %+v", c.typ, c.i1, r)
		}
	}
}

func TestCheckInstalled(t *testing.T) {
	tools31 := golden(t, "awg-tools-v3.1.20260812-git.txt")
	toolsStub := golden(t, "awg-tools-v1.0.20250901-tarball.txt")
	toolsOld := golden(t, "awg-tools-v1.0.20240213-git.txt")
	loaded := func(genl, module string) Installed {
		return Installed{Tools: tools31, ModuleLoaded: true, ModulePresent: true, Genl: genl, Module: module}
	}
	cases := []struct {
		name     string
		typ      Type
		i1       bool
		in       Installed
		wantErr  string // подстрока; "" — без ошибки
		warnings int
	}{
		{"awg3 all 3.1", TypeAWG3, false,
			loaded(golden(t, "genl-amneziawg-v3.txt"), "3.1.20260812\n"), "", 0},
		{"awg3 genl 2 -> 2.0", TypeAWG3, false,
			Installed{Tools: toolsStub, ModuleLoaded: true, ModulePresent: true, Genl: golden(t, "genl-amneziawg-v2.txt"), Module: "1.0.20251009\n"},
			"awg3 requires AmneziaWG >= 3.1, installed: 2.0", 1},
		{"awg3 module string 3.0", TypeAWG3, false,
			loaded(golden(t, "genl-amneziawg-v3.txt"), "3.0.20260805\n"),
			"awg3 requires AmneziaWG >= 3.1, installed: 3.0", 0},
		{"awg3 module placeholder warns", TypeAWG3, false,
			loaded(golden(t, "genl-amneziawg-v3.txt"), "1.0.0\n"), "", 1},
		{"awg3 tools 2.0 reliable", TypeAWG3, false,
			Installed{Tools: "amneziawg-tools v1.0.20260618-2 - https://amnezia.org\n", ModuleLoaded: true, ModulePresent: true,
				Genl: golden(t, "genl-amneziawg-v3.txt"), Module: "3.1.20260812\n"},
			"awg3 requires AmneziaWG >= 3.1, installed: 2.0", 0},
		{"awg2 tools placeholder, genl 2", TypeAWG2, false,
			Installed{Tools: toolsStub, ModuleLoaded: true, ModulePresent: true, Genl: golden(t, "genl-amneziawg-v2.txt"), Module: "1.0.20220627\n"},
			"", 1},
		{"awg2 genl 1", TypeAWG2, false,
			loaded(golden(t, "genl-amneziawg-v1.txt"), "1.0.20220627\n"),
			"awg2 requires AmneziaWG >= 2.0, installed: 1.0", 0},
		{"awg2 genl unavailable, module ok", TypeAWG2, false,
			loaded("", "1.0.20251009\n"), "", 1},
		{"awg2 genl unavailable, module placeholder", TypeAWG2, false,
			loaded("", "1.0.20220627\n"), "", 2},
		{"awg2 modinfo old placeholder", TypeAWG2, false,
			Installed{Tools: tools31, ModulePresent: true, Module: "1.0.20220627\n"}, "", 1},
		{"awg2 modinfo ok", TypeAWG2, false,
			Installed{Tools: tools31, ModulePresent: true, Module: "1.0.20251009\n"}, "", 0},
		{"awg2 go ok", TypeAWG2, false,
			Installed{Tools: tools31, GoPresent: true, Go: golden(t, "amneziawg-go-v0.2.15-make.txt")}, "", 0},
		{"awg2 go placeholder", TypeAWG2, false,
			Installed{Tools: tools31, GoPresent: true, Go: golden(t, "amneziawg-go-v3.1.20260828-gobuild.txt")}, "", 1},
		{"awg2 go 1.5", TypeAWG2, false,
			Installed{Tools: tools31, GoPresent: true, Go: "amneziawg-go v0.2.13\n"},
			"awg2 requires AmneziaWG >= 2.0, installed: 1.5", 0},
		{"awg+i1 go 0.2.13", TypeAWG, true,
			Installed{Tools: tools31, GoPresent: true, Go: "amneziawg-go v0.2.13\n"}, "", 0},
		{"awg+i1 go 0.2.12", TypeAWG, true,
			Installed{Tools: tools31, GoPresent: true, Go: "amneziawg-go v0.2.12\n"},
			"awg with --i1 requires AmneziaWG >= 1.5, installed: 1.0", 0},
		{"awg 1.0 placeholders are fine", TypeAWG, false,
			Installed{Tools: toolsOld, GoPresent: true, Go: golden(t, "amneziawg-go-v3.1.20260828-gobuild.txt")}, "", 0},
		{"no module and no amneziawg-go", TypeAWG2, false,
			Installed{Tools: tools31}, "amneziawg-go", 0},
		{"tools output unparsable", TypeAWG2, false,
			Installed{Tools: "weird\n", GoPresent: true, Go: "amneziawg-go v0.2.15\n"}, "", 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			warnings, err := CheckInstalled(RequirementFor(c.typ, c.i1), c.in)
			if c.wantErr == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if c.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), c.wantErr) {
					t.Fatalf("error = %v; want containing %q", err, c.wantErr)
				}
				return
			}
			if len(warnings) != c.warnings {
				t.Fatalf("warnings = %q; want %d", warnings, c.warnings)
			}
		})
	}
}
