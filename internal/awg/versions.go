package awg

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Version — версия компонента вида X.Y.Z (Z у AmneziaWG — дата YYYYMMDD).
type Version struct {
	Major, Minor, Patch int
}

var versionRe = regexp.MustCompile(`^v?(\d+)\.(\d+)\.(\d+)(?:-[0-9A-Za-z.-]+)?$`)

// ParseVersion разбирает "1.0.20250901", "v3.1.20260812", "v3.0.20260805-3-gabc-dirty".
// Суффиксы после '-' (релизные и git describe) игнорируются.
func ParseVersion(s string) (Version, error) {
	m := versionRe.FindStringSubmatch(s)
	if m == nil {
		return Version{}, fmt.Errorf("unrecognized version %q", s)
	}
	var v Version
	var err error
	for i, p := range []*int{&v.Major, &v.Minor, &v.Patch} {
		if *p, err = strconv.Atoi(m[i+1]); err != nil {
			return Version{}, fmt.Errorf("unrecognized version %q", s)
		}
	}
	return v, nil
}

func mustParseVersion(s string) Version {
	v, err := ParseVersion(s)
	if err != nil {
		panic(err)
	}
	return v
}

// Compare возвращает -1, 0 или 1.
func (v Version) Compare(o Version) int {
	for _, d := range [3]int{v.Major - o.Major, v.Minor - o.Minor, v.Patch - o.Patch} {
		if d < 0 {
			return -1
		}
		if d > 0 {
			return 1
		}
	}
	return 0
}

func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

func (v Version) isZero() bool { return v == Version{} }

func firstLine(s string) string {
	s = strings.TrimLeft(s, "\r\n")
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

// ParseToolsOutput разбирает `awg --version` / `wg --version`:
// "amneziawg-tools v3.1.20260812 - https://amnezia.org",
// "wireguard-tools v1.0.20210914 - https://git.zx2c4.com/wireguard-tools/".
func ParseToolsOutput(out string) (Version, error) {
	f := strings.Fields(firstLine(out))
	if len(f) < 2 || !strings.HasSuffix(f[0], "-tools") {
		return Version{}, fmt.Errorf("unrecognized tools version output %q", firstLine(out))
	}
	return ParseVersion(f[1])
}

// ParseGoOutput разбирает `amneziawg-go --version`: первая строка "amneziawg-go v0.2.15"
// (сборка make) или "amneziawg-go 0.0.20250522" (сборка go build).
func ParseGoOutput(out string) (Version, error) {
	f := strings.Fields(firstLine(out))
	if len(f) < 2 || (f[0] != "amneziawg-go" && f[0] != "wireguard-go") {
		return Version{}, fmt.Errorf("unrecognized amneziawg-go version output %q", firstLine(out))
	}
	return ParseVersion(f[1])
}

// ParseModuleOutput разбирает /sys/module/amneziawg/version или `modinfo -F version amneziawg`.
func ParseModuleOutput(out string) (Version, error) {
	return ParseVersion(strings.TrimSpace(out))
}

var genlVersionRe = regexp.MustCompile(`\bVersion:\s*0x([0-9a-fA-F]+)\b`)

// ParseGenlOutput извлекает версию netlink-семейства из `genl ctrl get name <family>`.
func ParseGenlOutput(out string) (int, error) {
	m := genlVersionRe.FindStringSubmatch(out)
	if m == nil {
		return 0, errors.New("no family version in genl output")
	}
	n, err := strconv.ParseInt(m[1], 16, 32)
	if err != nil {
		return 0, err
	}
	return int(n), nil
}

// Requirement — минимальные версии компонентов AmneziaWG для типа (SPEC 3.1).
type Requirement struct {
	Label  string  // как тип называется в сообщении: "awg3", "awg with --i1"
	Gen    string  // поколение протокола: "1.0", "1.5", "2.0", "3.1"
	Tools  Version // amneziawg-tools (`awg --version`)
	Module Version // строка версии модуля; нулевая — не проверяется
	Genl   int     // минимальная версия netlink-семейства amneziawg
	Go     Version // amneziawg-go
}

// Таблица SPEC 3.1 «тип → минимальная версия» — единственное место, где она хранится.
// Значения взяты из реальных релизов upstream (задача T0).
var (
	reqAWG = Requirement{Label: "awg", Gen: "1.0",
		Tools: mustParseVersion("1.0.20240213"), Genl: 1, Go: mustParseVersion("0.1.0")}
	reqAWGI1 = Requirement{Label: "awg with --i1", Gen: "1.5",
		Tools: mustParseVersion("1.0.20250901"), Module: mustParseVersion("1.0.20251009"), Genl: 2, Go: mustParseVersion("0.2.13")}
	reqAWG2 = Requirement{Label: "awg2", Gen: "2.0",
		Tools: mustParseVersion("1.0.20250901"), Module: mustParseVersion("1.0.20251009"), Genl: 2, Go: mustParseVersion("0.2.15")}
	reqAWG3 = Requirement{Label: "awg3", Gen: "3.1",
		Tools: mustParseVersion("3.1.20260812"), Module: mustParseVersion("3.1.20260812"), Genl: 3, Go: mustParseVersion("3.1.20260812")}
)

// RequirementFor возвращает требования для типа awg*; i1 — задан ли --i1 (awg 1.5).
func RequirementFor(t Type, i1 bool) Requirement {
	switch t {
	case TypeAWG2:
		return reqAWG2
	case TypeAWG3:
		return reqAWG3
	}
	if i1 {
		return reqAWGI1
	}
	return reqAWG
}

// Поколение по версии компонента: первая строка, чей минимум не выше версии.
type genStep struct {
	min Version
	gen string
}

var (
	toolsGens = []genStep{
		{mustParseVersion("3.1.20260812"), "3.1"},
		{mustParseVersion("3.0.20260730"), "3.0"},
		{mustParseVersion("1.0.20250901"), "2.0"},
	}
	moduleGens = []genStep{
		{mustParseVersion("3.1.20260812"), "3.1"},
		{mustParseVersion("3.0.20260730"), "3.0"},
		{mustParseVersion("1.0.20251009"), "2.0"},
	}
	goGens = []genStep{
		{mustParseVersion("3.1.20260812"), "3.1"},
		{mustParseVersion("3.0.0"), "3.0"},
		{mustParseVersion("0.2.15"), "2.0"},
		{mustParseVersion("0.2.13"), "1.5"},
	}
)

func generation(steps []genStep, v Version) string {
	for _, s := range steps {
		if v.Compare(s.min) >= 0 {
			return s.gen
		}
	}
	return "1.0"
}

func genlGeneration(n int) string {
	switch {
	case n >= 3:
		return "3.x"
	case n == 2:
		return "2.0"
	}
	return "1.0"
}

// Версии-«заглушки», которые сообщают некоторые способы сборки (SPEC 3.1).
var (
	toolsPlaceholder   = mustParseVersion("1.0.20210914")
	modulePlaceholders = []Version{mustParseVersion("1.0.0"), mustParseVersion("1.0.20220627")}
	// Первый релиз amneziawg-tools с бинарём `awg`: всё, что так называется, умеет 1.0.
	firstAWGTools = mustParseVersion("1.0.20240213")
	firstAWGGo    = mustParseVersion("0.1.0")
)

func isModulePlaceholder(v Version) bool {
	for _, p := range modulePlaceholders {
		if v == p {
			return true
		}
	}
	return false
}

// Installed — что найдено в системе (сырые выводы команд).
type Installed struct {
	Tools         string // вывод `awg --version`
	ModuleLoaded  bool   // модуль загружен (/sys/module/amneziawg существует)
	ModulePresent bool   // модуль загружен или установлен (modinfo нашёл его)
	Genl          string // вывод `genl ctrl get name amneziawg`; "" — genl недоступен или ошибка
	Module        string // /sys/module/amneziawg/version или `modinfo -F version amneziawg`
	GoPresent     bool   // amneziawg-go есть в PATH
	Go            string // вывод `amneziawg-go --version`
}

// CheckInstalled проверяет установленную AmneziaWG по требованию (SPEC 3.1).
// Ошибка — надёжно распознанная версия ниже требуемой либо нет ни модуля, ни amneziawg-go.
// Нераспознанные версии и «заглушки» дают предупреждения.
func CheckInstalled(req Requirement, in Installed) (warnings []string, err error) {
	warn := func(format string, a ...any) {
		warnings = append(warnings, fmt.Sprintf(format, a...)+fmt.Sprintf("; assuming it supports %s", req.Label))
	}
	tooOld := func(gen, detail string) error {
		return fmt.Errorf("%s requires AmneziaWG >= %s, installed: %s (%s)", req.Label, req.Gen, gen, detail)
	}

	// 1. amneziawg-tools.
	if v, perr := ParseToolsOutput(in.Tools); perr != nil {
		warn("cannot determine amneziawg-tools version: %v", perr)
	} else if v == toolsPlaceholder {
		if req.Tools.Compare(firstAWGTools) > 0 {
			warn("cannot determine amneziawg-tools version: it reports placeholder %s", v)
		}
	} else if v.Compare(req.Tools) < 0 {
		return warnings, tooOld(generation(toolsGens, v), "amneziawg-tools "+v.String())
	}

	// 2. Модуль ядра или 3. amneziawg-go.
	checkModuleString := func() error {
		if req.Module.isZero() {
			return nil
		}
		v, perr := ParseModuleOutput(in.Module)
		switch {
		case perr != nil:
			warn("cannot determine amneziawg kernel module version: %v", perr)
		case isModulePlaceholder(v):
			warn("cannot determine amneziawg kernel module version: it reports placeholder %s", v)
		case v.Compare(req.Module) < 0:
			return tooOld(generation(moduleGens, v), "amneziawg kernel module "+v.String())
		}
		return nil
	}
	switch {
	case in.ModuleLoaded:
		if in.Genl == "" {
			warn("cannot query amneziawg netlink family (genl unavailable)")
			return warnings, checkModuleString()
		}
		n, perr := ParseGenlOutput(in.Genl)
		if perr != nil {
			warn("cannot determine amneziawg netlink family version: %v", perr)
			return warnings, checkModuleString()
		}
		if n < req.Genl {
			return warnings, tooOld(genlGeneration(n), fmt.Sprintf("amneziawg kernel module, netlink family version %d", n))
		}
		// Версия семейства 3 не различает 3.0 и 3.1 — тогда уточняем по строке версии.
		if req.Genl >= 3 {
			return warnings, checkModuleString()
		}
		return warnings, nil
	case in.ModulePresent:
		return warnings, checkModuleString()
	case in.GoPresent:
		v, perr := ParseGoOutput(in.Go)
		switch {
		case perr != nil:
			warn("cannot determine amneziawg-go version: %v", perr)
		case v.Major == 0 && v.Minor == 0:
			if req.Go.Compare(firstAWGGo) > 0 {
				warn("cannot determine amneziawg-go version: it reports placeholder %s", v)
			}
		case v.Compare(req.Go) < 0:
			return warnings, tooOld(generation(goGens, v), "amneziawg-go "+v.String())
		}
		return warnings, nil
	}
	return warnings, errors.New("AmneziaWG kernel module (amneziawg) or userspace amneziawg-go is required")
}
