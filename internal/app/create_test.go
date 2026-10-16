package app

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/sovchinnikov/krot/internal/awg"
	"github.com/sovchinnikov/krot/internal/conf"
	"github.com/sovchinnikov/krot/internal/errs"
	"github.com/sovchinnikov/krot/internal/keys"
)

func createArgs(typ string, extra ...string) []string {
	args := []string{"--create", "--type", typ, "--ip", "10.8.0.1/24", "--port", "51820", "--endpoint", "vpn.example.com"}
	return append(args, extra...)
}

// checkCreated проверяет созданный конфиг структурно (SPEC 5.1, 6.2, 7).
func checkCreated(t *testing.T, e *env, path string, typ awg.Type, mtu int) *conf.Config {
	t.Helper()
	st, err := os.Stat(path)
	if err != nil {
		t.Fatalf("config not written: %v", err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("config mode = %v; want 0600", st.Mode().Perm())
	}
	c := conf.Parse([]byte(e.read(path)))
	ifc, ok := c.Interface()
	if !ok {
		t.Fatal("no [Interface]")
	}
	if ifc.Meta["type"] != string(typ) || ifc.Meta["endpoint"] != "vpn.example.com" {
		t.Fatalf("meta = %v", ifc.Meta)
	}
	if ifc.Get("ListenPort") != "51820" || ifc.Get("MTU") != strconv.Itoa(mtu) {
		t.Fatalf("ListenPort/MTU = %q/%q", ifc.Get("ListenPort"), ifc.Get("MTU"))
	}
	priv, err := keys.ParseKey(ifc.Get("PrivateKey"))
	if err != nil {
		t.Fatalf("PrivateKey: %v", err)
	}
	if _, err := priv.Public(); err != nil {
		t.Fatal(err)
	}
	var p awg.Params
	for _, kv := range ifc.Keys {
		if k, ok := awg.CanonicalKey(kv.Key); ok {
			p = append(p, awg.Param{Key: k, Value: kv.Value})
		}
	}
	if err := awg.Validate(typ, p, mtu); err != nil {
		t.Fatalf("generated params invalid: %v", err)
	}
	for _, hook := range []string{"# PreUp =\n", "# PostUp =\n", "# PreDown =\n", "# PostDown =\n"} {
		if !strings.Contains(e.read(path), hook) {
			t.Fatalf("missing %q", hook)
		}
	}
	if len(c.Peers()) != 0 {
		t.Fatal("new interface must have no peers")
	}
	return c
}

// SPEC 6.2: создание wg-туннеля.
func TestCreateWG(t *testing.T) {
	e := newEnv(t)
	e.mustRun(0, "--create", "--type", "wg", "--ip", "10.9.0.1/24", "--port", "51820", "--endpoint", "vpn.example.com")
	path := e.wgConf("wg0")
	checkCreated(t, e, path, awg.TypeWG, 1420)
	golden(t, "create_wg.golden", e.read(path))
	if !e.called("ip link show dev wg0") || !e.called("systemctl enable --now wg-quick@wg0") {
		t.Fatalf("calls = %q", e.fake.CommandLines())
	}
	out := e.stdout.String()
	for _, s := range []string{"wg0", "wg", "10.9.0.1/24", "51820", "vpn.example.com"} {
		if !strings.Contains(out, s) {
			t.Errorf("summary misses %q:\n%s", s, out)
		}
	}
}

// SPEC 6.2, 7: создание awg-туннелей всех типов; MTU по умолчанию; каталог amnezia создаётся.
func TestCreateAWG(t *testing.T) {
	cases := []struct {
		name   string
		typ    awg.Type
		args   []string
		mtu    int
		golden string
		meta   map[string]string
		addrs  []string
	}{
		{"awg", awg.TypeAWG, createArgs("awg"), 1376, "create_awg.golden", nil, []string{"10.8.0.1/24"}},
		{"awg+i1", awg.TypeAWG, createArgs("awg", "--i1", "<b 0xf6ab3267fa><r 16>"), 1376, "create_awg_i1.golden", nil, []string{"10.8.0.1/24"}},
		{"awg2", awg.TypeAWG2, []string{"--create", "--type", "awg2", "--ip", "10.8.0.1/24,fd00:8::1/64", "--port", "51820",
			"--endpoint", "vpn.example.com", "--dns", "1.1.1.1,9.9.9.9"}, 1376, "create_awg2.golden",
			map[string]string{"dns": "1.1.1.1,9.9.9.9"}, []string{"10.8.0.1/24", "fd00:8::1/64"}},
		{"awg3", awg.TypeAWG3, createArgs("awg3"), 1280, "create_awg3.golden", nil, []string{"10.8.0.1/24"}},
		{"awg3 custom mtu", awg.TypeAWG3, createArgs("awg3", "--mtu", "1400", "--interface", "tun-x"), 1400, "", nil, []string{"10.8.0.1/24"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := newEnv(t)
			e.mustRun(0, c.args...)
			name := "awg0"
			if c.golden == "" {
				name = "tun-x"
			}
			path := e.awgConf(name)
			cfg := checkCreated(t, e, path, c.typ, c.mtu)
			ifc, _ := cfg.Interface()
			for k, v := range c.meta {
				if ifc.Meta[k] != v {
					t.Errorf("meta %s = %q; want %q", k, ifc.Meta[k], v)
				}
			}
			if got := ifc.List("Address"); strings.Join(got, ",") != strings.Join(c.addrs, ",") {
				t.Errorf("Address = %q", got)
			}
			if c.golden != "" {
				golden(t, c.golden, e.read(path))
			}
			st, _ := os.Stat(e.app.Paths.AWGDir)
			if st.Mode().Perm() != 0o700 {
				t.Errorf("amnezia dir mode = %v", st.Mode().Perm())
			}
			if !e.called("systemctl enable --now awg-quick@" + name) {
				t.Fatalf("calls = %q", e.fake.CommandLines())
			}
			if strings.Contains(e.stderr.String(), "warning") {
				t.Errorf("unexpected warnings: %s", e.stderr)
			}
		})
	}
}

// SPEC 6.2 п.2: первое свободное имя wg0, wg1… / awg0, awg1… (конфиг в любом каталоге или живой интерфейс).
func TestCreateAutoName(t *testing.T) {
	e := newEnv(t)
	e.write(e.wgConf("wg0"), fixture(t, "wg0_empty.conf"))
	e.write(e.awgConf("wg1"), fixture(t, "wg0_empty.conf"))
	e.fake.On("ip link show dev wg2", "5: wg2: <POINTOPOINT,NOARP,UP,LOWER_UP> mtu 1420\n")
	e.mustRun(0, "--create", "--type", "wg", "--ip", "10.10.0.1/24", "--port", "51830", "--endpoint", "vpn.example.com")
	if !e.exists(e.wgConf("wg3")) {
		t.Fatalf("wg3 not created; calls %q", e.fake.CommandLines())
	}

	e.write(e.awgConf("awg0"), fixture(t, "awg0_two_clients.conf"))
	e.mustRun(0, "--create", "--type", "awg3", "--ip", "10.11.0.1/24", "--port", "51831", "--endpoint", "vpn.example.com")
	if !e.exists(e.awgConf("awg1")) {
		t.Fatal("awg1 not created")
	}
}

// SPEC 6.2 п.3: конфликты — код 5, ничего не создаётся.
func TestCreateConflicts(t *testing.T) {
	cases := []struct {
		name  string
		setup func(e *env)
		args  []string
	}{
		{"config in wg dir", func(e *env) { e.write(e.wgConf("wg0"), fixture(t, "wg0_empty.conf")) },
			createArgs("wg", "--interface", "wg0")},
		{"config in amnezia dir", func(e *env) { e.write(e.awgConf("wg0"), fixture(t, "wg0_empty.conf")) },
			createArgs("wg", "--interface", "wg0")},
		{"link exists", func(e *env) { e.fake.On("ip link show dev wg5", "7: wg5: <UP>\n") },
			createArgs("wg", "--interface", "wg5")},
		{"port in another config", func(e *env) {
			e.write(e.wgConf("other"), strings.Replace(fixture(t, "wg0_empty.conf"), "51821", "51820", 1))
		}, createArgs("awg2")},
		{"port in foreign config", func(e *env) {
			e.write(e.awgConf("foreign"), strings.Replace(fixture(t, "foreign_wg.conf"), "51821", "51820", 1))
		}, createArgs("wg")},
		{"port listened (udp)", func(e *env) {
			e.write(e.app.Paths.ProcNet+"/udp", udpHdr+
				"  123: 00000000:CA6C 00000000:0000 07 00000000:00000000 00:00000000 00000000     0        0 4242 2 0000000000000000 0\n")
		}, createArgs("wg")},
		{"port listened (udp6)", func(e *env) {
			e.write(e.app.Paths.ProcNet+"/udp6", udpHdr+
				"  7: 00000000000000000000000000000000:CA6C 00000000000000000000000000000000:0000 07 00000000:00000000 00:00000000 00000000     0        0 4243 2 0000000000000000 0\n")
		}, createArgs("wg")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := newEnv(t)
			c.setup(e)
			e.mustRun(errs.Conflict, c.args...)
			if e.calledPrefix("systemctl") {
				t.Fatalf("systemctl called: %q", e.fake.CommandLines())
			}
			if e.exists(e.wgConf("wg0")) && c.name != "config in wg dir" {
				t.Fatal("config written despite the conflict")
			}
		})
	}
	// Порт, не совпадающий с занятым, — не конфликт.
	e := newEnv(t)
	e.write(e.app.Paths.ProcNet+"/udp", udpHdr+
		"  1: 00000000:CA6D 00000000:0000 07 00000000:00000000 00:00000000 00000000     0        0 1 2 0 0\n")
	e.mustRun(0, createArgs("wg")...)
}

// SPEC 2.2, 3.1: окружение — код 6 с названием пакета.
func TestCreateEnvironment(t *testing.T) {
	cases := []struct {
		name  string
		typ   string
		setup func(e *env)
		want  string
	}{
		{"no wg", "wg", func(e *env) { e.fake.Missing = map[string]bool{"wg": true} }, "wireguard-tools"},
		{"no wg-quick", "wg", func(e *env) { e.fake.Missing = map[string]bool{"wg-quick": true} }, "wireguard-tools"},
		{"no systemctl", "wg", func(e *env) { e.fake.Missing = map[string]bool{"systemctl": true} }, "systemd"},
		{"no ip", "awg2", func(e *env) { e.fake.Missing = map[string]bool{"ip": true} }, "iproute2"},
		{"no awg", "awg2", func(e *env) { e.fake.Missing = map[string]bool{"awg": true} }, "amneziawg-tools"},
		{"no awg-quick", "awg", func(e *env) { e.fake.Missing = map[string]bool{"awg-quick": true} }, "amneziawg-tools"},
		{"no wireguard module", "wg", func(e *env) {
			os.RemoveAll(e.app.Paths.SysModule + "/wireguard")
			e.fake.Fail("modinfo -F version wireguard", 1, "modinfo: ERROR: Module wireguard not found.")
		}, "WireGuard kernel module"},
		{"awg3 on old tools", "awg3", func(e *env) {
			e.fake.On("awg --version", "amneziawg-tools v1.0.20260618 - https://amnezia.org\n")
		}, "awg3 requires AmneziaWG >= 3.1, installed: 2.0"},
		{"awg2 on old amneziawg-go", "awg2", func(e *env) {
			e.fake.On("amneziawg-go --version", "amneziawg-go v0.2.12\n")
		}, "awg2 requires AmneziaWG >= 2.0"},
		{"no module and no amneziawg-go", "awg", func(e *env) {
			e.fake.Missing = map[string]bool{"amneziawg-go": true}
		}, "amneziawg-go"},
		{"loaded module with old netlink family", "awg3", func(e *env) {
			e.write(e.app.Paths.SysModule+"/amneziawg/version", "1.0.20260611\n")
			e.fake.On("genl ctrl get name amneziawg", "\nName: amneziawg\n\tID: 0x22  Version: 0x2  header size: 0  max attribs: 33 \n")
		}, "awg3 requires AmneziaWG >= 3.1, installed: 2.0"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e := newEnv(t)
			c.setup(e)
			e.mustRun(errs.Env, createArgs(c.typ)...)
			if !strings.Contains(e.stderr.String(), c.want) {
				t.Fatalf("stderr = %q; want %q", e.stderr, c.want)
			}
			if e.calledPrefix("systemctl") || e.exists(e.wgConf("wg0")) || e.exists(e.awgConf("awg0")) {
				t.Fatal("changes made despite environment error")
			}
		})
	}
}

// SPEC 3.1: модуль ядра проверяется через genl и строку версии; заглушки — предупреждение.
func TestCreateModuleChecks(t *testing.T) {
	e := newEnv(t)
	e.write(e.app.Paths.SysModule+"/amneziawg/version", "3.1.20260812\n")
	e.fake.On("genl ctrl get name amneziawg", "\nName: amneziawg\n\tID: 0x22  Version: 0x3  header size: 0  max attribs: 33 \n")
	e.mustRun(0, createArgs("awg3")...)
	if strings.Contains(e.stderr.String(), "warning") || e.calledPrefix("amneziawg-go") {
		t.Fatalf("stderr %q, calls %q", e.stderr, e.fake.CommandLines())
	}

	e = newEnv(t)
	e.fake.On("awg --version", "amneziawg-tools v1.0.20210914 - https://amnezia.org\n")
	e.fake.On("modinfo -F version amneziawg", "1.0.0\n")
	e.mustRun(0, createArgs("awg2")...)
	if n := strings.Count(e.stderr.String(), "warning"); n != 2 {
		t.Fatalf("want 2 warnings, stderr:\n%s", e.stderr)
	}

	// Модуль wireguard не загружен, но установлен — это нормально.
	e = newEnv(t)
	os.RemoveAll(e.app.Paths.SysModule + "/wireguard")
	e.fake.On("modinfo -F version wireguard", "1.0.0\n")
	e.mustRun(0, createArgs("wg")...)
}

// SPEC 6.2: при ошибке systemctl конфиг удаляется, unit отключается.
func TestCreateRollback(t *testing.T) {
	e := newEnv(t)
	e.fake.Fail("systemctl enable --now wg-quick@wg0", 1, "Job for wg-quick@wg0.service failed.")
	e.mustRun(errs.Internal, createArgs("wg")...)
	if e.exists(e.wgConf("wg0")) {
		t.Fatal("config not removed on rollback")
	}
	if !e.called("systemctl disable --now wg-quick@wg0") {
		t.Fatalf("unit not disabled: %q", e.fake.CommandLines())
	}
	if !strings.Contains(e.stderr.String(), "Job for wg-quick@wg0.service failed.") {
		t.Fatalf("stderr = %q", e.stderr)
	}
}
