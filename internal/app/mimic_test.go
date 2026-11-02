package app

import (
	"strings"
	"testing"

	"github.com/snmp161/krot/internal/awg"
	"github.com/snmp161/krot/internal/conf"
)

// noIParams проверяет, что в [Interface] нет I-параметров.
func noIParams(t *testing.T, ifc conf.Interface, where string) {
	t.Helper()
	for _, k := range []string{"I1", "I2", "I3", "I4", "I5"} {
		if ifc.Get(k) != "" {
			t.Fatalf("%s: %s must be absent (SPEC 6.12)", where, k)
		}
	}
}

// SPEC 6.12: --create --mimic запоминает профиль в метаданных, серверу I-параметры не нужны.
func TestCreateMimic(t *testing.T) {
	e := newEnv(t)
	e.mustRun(0, createArgs("awg", "--mimic", "dns")...)
	c := conf.Parse([]byte(e.read(e.awgConf("awg0"))))
	ifc, ok := c.Interface()
	if !ok {
		t.Fatal("no [Interface]")
	}
	if ifc.Meta["mimic"] != "dns" {
		t.Fatalf("krot:mimic = %q; want dns", ifc.Meta["mimic"])
	}
	noIParams(t, ifc, "server config")
}

// SPEC 6.12, 7.6: --add-client генерирует цепочку I1–Ik профиля по умолчанию;
// серверный конфиг не меняется.
func TestAddClientMimicChain(t *testing.T) {
	e := newEnv(t)
	e.mustRun(0, createArgs("awg", "--mimic", "dns")...)
	e.mustRun(0, "--add-client", "--interface", "awg0", "--id", "phone", "--allowed-ips", "0.0.0.0/0")

	ifc, ok := clientConfig(t, e.stdout.String()).Interface()
	if !ok {
		t.Fatal("client config has no [Interface]")
	}
	total := 0
	vals := map[string]string{}
	for _, k := range []string{"I1", "I2", "I3"} {
		v := ifc.Get(k)
		if v == "" {
			t.Fatalf("client config: %s missing (dns chain is 3 packets, SPEC 7.6)", k)
		}
		if err := awg.ValidateCPS(v); err != nil {
			t.Fatalf("client %s fails V-05: %v", k, err)
		}
		vals[k] = v
		total += len(v)
	}
	if ifc.Get("I4") != "" {
		t.Fatal("dns chain is 3 packets; I4 must be absent")
	}
	if total > awg.MaxITotal {
		t.Fatalf("chain is %d chars; want ≤ %d (V-06)", total, awg.MaxITotal)
	}
	// Пакеты цепочки различны (разные типы запросов).
	if vals["I1"] == vals["I2"] || vals["I2"] == vals["I3"] || vals["I1"] == vals["I3"] {
		t.Fatal("dns chain packets must differ")
	}

	s := conf.Parse([]byte(e.read(e.awgConf("awg0"))))
	sifc, _ := s.Interface()
	noIParams(t, sifc, "server config")
}

// SPEC 6.12: --add-client --mimic переопределяет профиль интерфейса.
func TestAddClientMimicOverride(t *testing.T) {
	e := newEnv(t)
	e.mustRun(0, createArgs("awg", "--mimic", "dns")...)
	e.mustRun(0, "--add-client", "--interface", "awg0", "--id", "a", "--allowed-ips", "0.0.0.0/0", "--mimic", "stun")

	ifc, _ := clientConfig(t, e.stdout.String()).Interface()
	v := ifc.Get("I1")
	if v == "" || awg.ValidateCPS(v) != nil {
		t.Fatalf("client I1 missing or invalid: %q", v)
	}
	if ifc.Get("I2") != "" {
		t.Fatal("stun chain is 1 packet; I2 must be absent (SPEC 7.6)")
	}
}

// SPEC 6.12, 7.5: без --mimic I-параметры сервера копируются как есть;
// --add-client --mimic заменяет их сгенерированной цепочкой.
func TestAddClientMimicOverridesStaticI1(t *testing.T) {
	e := newEnv(t)
	e.mustRun(0, createArgs("awg", "--i1", "<r 32>")...)

	e.mustRun(0, "--add-client", "--interface", "awg0", "--id", "a", "--allowed-ips", "0.0.0.0/0")
	ifc, _ := clientConfig(t, e.stdout.String()).Interface()
	if v := ifc.Get("I1"); v != "<r 32>" {
		t.Fatalf("without --mimic I1 must be copied verbatim; got %q", v)
	}

	e.mustRun(0, "--add-client", "--interface", "awg0", "--id", "b", "--allowed-ips", "0.0.0.0/0", "--mimic", "ntp")
	ifc, _ = clientConfig(t, e.stdout.String()).Interface()
	v := ifc.Get("I1")
	if v == "" {
		t.Fatal("--mimic ntp: client I1 missing")
	}
	if v == "<r 32>" {
		t.Fatal("--mimic must replace the copied I1 with a generated chain")
	}
	if err := awg.ValidateCPS(v); err != nil {
		t.Fatalf("generated I1 fails V-05: %v", err)
	}
	if ifc.Get("I2") != "" {
		t.Fatal("ntp chain is 1 packet; I2 must be absent")
	}
}

// SPEC 6.12: уникальность цепочек — два клиента одного интерфейса получают разные I1.
func TestAddClientMimicUniquePerClient(t *testing.T) {
	e := newEnv(t)
	e.mustRun(0, createArgs("awg", "--mimic", "dns")...)
	e.mustRun(0, "--add-client", "--interface", "awg0", "--id", "a", "--allowed-ips", "0.0.0.0/0")
	aifc, _ := clientConfig(t, e.stdout.String()).Interface()
	a1 := aifc.Get("I1")
	e.mustRun(0, "--add-client", "--interface", "awg0", "--id", "b", "--allowed-ips", "0.0.0.0/0")
	bifc, _ := clientConfig(t, e.stdout.String()).Interface()
	b1 := bifc.Get("I1")
	if a1 == "" || a1 == b1 {
		t.Fatal("clients of one interface must get different chains (SPEC 6.12)")
	}
}

// SPEC 6.12, 5.3: чужой конфиг с krot:mimic тоже получает сгенерированную цепочку.
func TestAddClientMimicForeignConfig(t *testing.T) {
	e := newEnv(t)
	path := e.awgConf("awg0")
	confText := strings.ReplaceAll(fixture(t, "awg0_two_clients.conf"), "# krot:endpoint=", "# krot:mimic=ssdp\n# krot:endpoint=")
	e.write(path, confText)
	e.fake.On("awg show interfaces", "awg0\n")
	e.mustRun(0, "--add-client", "--interface", "awg0", "--id", "new", "--allowed-ips", "0.0.0.0/0")
	ifc, _ := clientConfig(t, e.stdout.String()).Interface()
	v := ifc.Get("I1")
	if v == "" || awg.ValidateCPS(v) != nil {
		t.Fatalf("foreign config with krot:mimic: client I1 missing or invalid: %q", v)
	}
	if ifc.Get("I2") != "" {
		t.Fatal("ssdp chain is 1 packet; I2 must be absent")
	}
}
