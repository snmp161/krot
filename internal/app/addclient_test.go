package app

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/sovchinnikov/krot/internal/awg"
	"github.com/sovchinnikov/krot/internal/conf"
	"github.com/sovchinnikov/krot/internal/errs"
	"github.com/sovchinnikov/krot/internal/execx"
	"github.com/sovchinnikov/krot/internal/keys"
)

// assertOnlyPeerOps — SPEC 6.10, 8.2: клиентские операции не вызывают ничего, кроме
// `wg|awg show ...` и точечных `wg|awg set <if> peer ...`. Этот тест нельзя ослаблять.
func assertOnlyPeerOps(t *testing.T, e *env, tool, iface string) {
	t.Helper()
	for _, c := range e.fake.Calls {
		ok := c.Name == tool && len(c.Args) > 0 &&
			(c.Args[0] == "show" || (len(c.Args) >= 4 && c.Args[0] == "set" && c.Args[1] == iface && c.Args[2] == "peer"))
		if !ok {
			t.Fatalf("SPEC 6.10 violated: client operation called %q (all calls: %q)", c.String(), e.fake.CommandLines())
		}
	}
}

func setupAWG0(t *testing.T, e *env, up bool) string {
	t.Helper()
	path := e.awgConf("awg0")
	e.write(path, fixture(t, "awg0_two_clients.conf"))
	if up {
		e.fake.On("awg show interfaces", "awg0\n")
	} else {
		e.fake.On("awg show interfaces", "")
	}
	return path
}

// splitOutput отделяет клиентский конфиг от QR-кода.
func clientConfig(t *testing.T, out string) *conf.Config {
	t.Helper()
	if i := strings.Index(out, "\n\n\n"); i >= 0 {
		out = out[:i+1]
	}
	return conf.Parse([]byte(out))
}

// checkClient проверяет клиентский конфиг по SPEC 6.5 и соответствие peer'у на сервере.
func checkClient(t *testing.T, server *conf.Config, client *conf.Config, id string, wantAddrs []string,
	endpoint, allowed string, psk bool) {
	t.Helper()
	sifc, _ := server.Interface()
	cifc, ok := client.Interface()
	if !ok {
		t.Fatal("client config has no [Interface]")
	}
	cpriv, err := keys.ParseKey(cifc.Get("PrivateKey"))
	if err != nil {
		t.Fatalf("client PrivateKey: %v", err)
	}
	cpub, _ := cpriv.Public()
	if got := cifc.List("Address"); !slices.Equal(got, wantAddrs) {
		t.Fatalf("client Address = %q; want %q", got, wantAddrs)
	}
	if m := sifc.Get("MTU"); cifc.Get("MTU") != m {
		t.Fatalf("client MTU = %q; want %q", cifc.Get("MTU"), m)
	}
	if dns := sifc.Meta["dns"]; dns != "" && strings.Join(cifc.List("DNS"), ",") != dns {
		t.Fatalf("client DNS = %q; want %q", cifc.List("DNS"), dns)
	}
	// SPEC 7.5: все AWG-параметры сервера — без изменений.
	for _, kv := range sifc.Keys {
		if k, ok := awg.CanonicalKey(kv.Key); ok && cifc.Get(k) != kv.Value {
			t.Fatalf("client %s = %q; want %q", k, cifc.Get(k), kv.Value)
		}
	}
	peers := client.Peers()
	if len(peers) != 1 {
		t.Fatalf("client config has %d peers", len(peers))
	}
	cp := peers[0]
	spriv, _ := keys.ParseKey(sifc.Get("PrivateKey"))
	spub, _ := spriv.Public()
	if cp.PublicKey() != spub.String() || cp.Get("Endpoint") != endpoint || strings.Join(cp.AllowedIPs(), ",") != allowed ||
		cp.Get("PersistentKeepalive") != "25" {
		t.Fatalf("client [Peer] = %+v", cp.Keys)
	}
	sp, ok := server.PeerByID(id)
	if !ok {
		t.Fatalf("server has no peer %s", id)
	}
	if sp.PublicKey() != cpub.String() || !slices.Equal(sp.AllowedIPs(), wantAddrs) || sp.Created != "2026-10-17T03:00:00Z" || sp.Locked {
		t.Fatalf("server peer = %+v", sp)
	}
	if psk != (sp.PresharedKey() != "") || cp.PresharedKey() != sp.PresharedKey() {
		t.Fatalf("PresharedKey: client %q server %q (want psk=%v)", cp.PresharedKey(), sp.PresharedKey(), psk)
	}
}

// SPEC 6.5: интерфейс поднят — peer пишется в конфиг и применяется вживую через `awg set`.
func TestAddClientLive(t *testing.T) {
	e := newEnv(t)
	path := setupAWG0(t, e, true)
	var pskFile string
	e.fake.Handler = func(c execx.Call) *execx.Response {
		if c.Name != "awg" || len(c.Args) < 2 || c.Args[0] != "set" {
			return nil
		}
		// PSK передаётся через временный файл 0600 (SPEC 6.5 п.5).
		i := slices.Index(c.Args, "preshared-key")
		if i < 0 {
			t.Errorf("no preshared-key in %q", c.String())
			return nil
		}
		pskFile = c.Args[i+1]
		st, err := os.Stat(pskFile)
		if err != nil || st.Mode().Perm() != 0o600 {
			t.Errorf("psk file %s: %v %v", pskFile, st, err)
			return nil
		}
		b, _ := os.ReadFile(pskFile)
		cfg := conf.Parse([]byte(e.read(path)))
		if p, ok := cfg.PeerByID("tablet"); !ok || strings.TrimSpace(string(b)) != p.PresharedKey() {
			t.Errorf("psk file content %q does not match config", b)
		}
		return nil
	}
	e.mustRun(0, "--add-client", "--interface", "awg0", "--id", "tablet", "--allowed-ips", "0.0.0.0/0,::/0", "--psk")
	server := conf.Parse([]byte(e.read(path)))
	checkClient(t, server, clientConfig(t, e.stdout.String()), "tablet",
		[]string{"10.8.0.4/32", "fd00:8::3/128"}, "vpn.example.com:51820", "0.0.0.0/0,::/0", true)

	sp, _ := server.PeerByID("tablet")
	want := "awg set awg0 peer " + sp.PublicKey() + " allowed-ips 10.8.0.4/32,fd00:8::3/128 preshared-key " + pskFile
	if !e.called(want) {
		t.Fatalf("live call missing: want %q, calls %q", want, e.fake.CommandLines())
	}
	if pskFile == "" || e.exists(pskFile) {
		t.Fatalf("psk temp file %q not removed", pskFile)
	}
	assertOnlyPeerOps(t, e, "awg", "awg0")
	golden(t, "add_client_awg2.golden", e.stdout.String())
	golden(t, "add_client_awg2_server.golden", e.read(path))
}

// SPEC 6.5: интерфейс не поднят — только запись в конфиг.
func TestAddClientDown(t *testing.T) {
	e := newEnv(t)
	path := setupAWG0(t, e, false)
	e.mustRun(0, "--add-client", "--interface", "awg0", "--id", "tablet", "--allowed-ips", "10.8.0.0/24")
	if e.calledPrefix("awg set") {
		t.Fatalf("live call on a down interface: %q", e.fake.CommandLines())
	}
	server := conf.Parse([]byte(e.read(path)))
	checkClient(t, server, clientConfig(t, e.stdout.String()), "tablet",
		[]string{"10.8.0.4/32", "fd00:8::3/128"}, "vpn.example.com:51820", "10.8.0.0/24", false)
	assertOnlyPeerOps(t, e, "awg", "awg0")
}

// SPEC 6.5, 7.5: клиентский конфиг для каждого типа содержит все AWG-параметры сервера.
func TestAddClientTypes(t *testing.T) {
	for _, typ := range []string{"wg", "awg", "awg_i1", "awg2", "awg3"} {
		t.Run(typ, func(t *testing.T) {
			e := newEnv(t)
			name := "awg0"
			path := e.awgConf(name)
			tool := "awg"
			if typ == "wg" {
				name, path, tool = "wg0", e.wgConf("wg0"), "wg"
			}
			e.write(path, fixture(t, "create_"+typ+".golden"))
			e.fake.On(tool+" show interfaces", name+"\n")
			e.mustRun(0, "--add-client", "--interface", name, "--id", "c1", "--allowed-ips", "0.0.0.0/0")
			server := conf.Parse([]byte(e.read(path)))
			ifc, _ := server.Interface()
			addr := "10.8.0.2/32"
			if typ == "wg" {
				addr = "10.9.0.2/32"
			}
			want := []string{addr}
			if len(ifc.List("Address")) == 2 {
				want = append(want, "fd00:8::2/128")
			}
			checkClient(t, server, clientConfig(t, e.stdout.String()), "c1", want, "vpn.example.com:51820", "0.0.0.0/0", false)
			assertOnlyPeerOps(t, e, tool, name)
			golden(t, "add_client_"+typ+".golden", e.stdout.String())
		})
	}
}

// SPEC 6.5 п.1: интерфейс существует (4), --id уникален с учётом заблокированных (5).
func TestAddClientIDConflicts(t *testing.T) {
	e := newEnv(t)
	e.mustRun(errs.NotFound, "--add-client", "--interface", "awg0", "--id", "x", "--allowed-ips", "0.0.0.0/0")
	path := setupAWG0(t, e, true)
	orig := e.read(path)
	for _, id := range []string{"phone-sergei", "laptop-old"} {
		e.mustRun(errs.Conflict, "--add-client", "--interface", "awg0", "--id", id, "--allowed-ips", "0.0.0.0/0")
		if e.read(path) != orig || e.calledPrefix("awg set") {
			t.Fatalf("%s: config changed or live call made", id)
		}
	}
}

// SPEC 6.5 п.2: адреса, заданные вручную.
func TestAddClientManualIP(t *testing.T) {
	e := newEnv(t)
	path := setupAWG0(t, e, false)
	e.mustRun(0, "--add-client", "--interface", "awg0", "--id", "m1", "--allowed-ips", "0.0.0.0/0", "--ip", "10.8.0.10/32")
	if p, _ := conf.Parse([]byte(e.read(path))).PeerByID("m1"); !slices.Equal(p.AllowedIPs(), []string{"10.8.0.10/32"}) {
		t.Fatalf("AllowedIPs = %q", p.AllowedIPs())
	}
	e.mustRun(0, "--add-client", "--interface", "awg0", "--id", "m2", "--allowed-ips", "0.0.0.0/0", "--ip", "10.8.0.11/32,fd00:8::11/128")
	if p, _ := conf.Parse([]byte(e.read(path))).PeerByID("m2"); !slices.Equal(p.AllowedIPs(), []string{"10.8.0.11/32", "fd00:8::11/128"}) {
		t.Fatalf("AllowedIPs = %q", p.AllowedIPs())
	}
	before := e.read(path)
	for _, ip := range []string{
		"10.8.0.2/32",   // занят активным клиентом
		"10.8.0.3/32",   // занят заблокированным клиентом
		"10.8.0.10/32",  // занят только что добавленным
		"fd00:8::2/128", // занят
		"10.8.0.1/32",   // адрес сервера
		"10.8.0.0/32",   // адрес сети
		"10.8.0.255/32", // broadcast
		"10.9.0.5/32",   // вне подсети
		"fd00:9::5/128", // вне подсети
		"fd00:8::/128",  // адрес сети IPv6
	} {
		e.mustRun(errs.Conflict, "--add-client", "--interface", "awg0", "--id", "x", "--allowed-ips", "0.0.0.0/0", "--ip", ip)
	}
	e.mustRun(errs.Invalid, "--add-client", "--interface", "awg0", "--id", "x", "--allowed-ips", "0.0.0.0/0", "--ip", "10.8.0.20/24")
	if e.read(path) != before {
		t.Fatal("config changed by rejected add-client")
	}
}

// SPEC 6.5 п.2: свободных адресов нет — код 5.
func TestAddClientExhausted(t *testing.T) {
	e := newEnv(t)
	path := e.wgConf("wg0")
	e.write(path, strings.Replace(fixture(t, "wg0_empty.conf"), "10.9.0.1/24", "10.9.0.1/30", 1))
	e.mustRun(0, "--add-client", "--interface", "wg0", "--id", "a", "--allowed-ips", "0.0.0.0/0")
	e.mustRun(errs.Conflict, "--add-client", "--interface", "wg0", "--id", "b", "--allowed-ips", "0.0.0.0/0")
}

// SPEC 6.5 п.6: живое применение не удалось — конфиг откатывается, код 1.
func TestAddClientRollback(t *testing.T) {
	e := newEnv(t)
	path := setupAWG0(t, e, true)
	orig := e.read(path)
	e.fake.Fail("awg set", 1, "Unable to modify interface: Protocol not supported")
	e.mustRun(errs.Internal, "--add-client", "--interface", "awg0", "--id", "tablet", "--allowed-ips", "0.0.0.0/0", "--psk")
	if e.read(path) != orig {
		t.Fatal("config not rolled back")
	}
	if strings.Contains(e.stdout.String(), "PrivateKey") {
		t.Fatal("client config printed although the client was not added")
	}
	assertOnlyPeerOps(t, e, "awg", "awg0")
}

// SPEC 4.3, 5.3, 6.5: чужой конфиг без krot:endpoint — нужен --endpoint (иначе 3);
// --endpoint переопределяет endpoint только для этого клиента.
func TestAddClientEndpoint(t *testing.T) {
	e := newEnv(t)
	path := e.wgConf("office")
	e.write(path, fixture(t, "foreign_wg.conf"))
	e.mustRun(errs.Invalid, "--add-client", "--interface", "office", "--id", "n1", "--allowed-ips", "0.0.0.0/0")
	if !strings.Contains(e.stderr.String(), "--endpoint") || e.read(path) != fixture(t, "foreign_wg.conf") {
		t.Fatalf("stderr %q", e.stderr)
	}
	e.mustRun(0, "--add-client", "--interface", "office", "--id", "n1", "--allowed-ips", "0.0.0.0/0", "--endpoint", "198.51.100.7")
	server := conf.Parse([]byte(e.read(path)))
	checkClient(t, server, clientConfig(t, e.stdout.String()), "n1",
		[]string{"10.9.0.4/32", "fd00:9::3/128"}, "198.51.100.7:51821", "0.0.0.0/0", false)
	// Чужие строки сохранены, peer дописан в конец (SPEC 5.3).
	if !strings.HasPrefix(e.read(path), fixture(t, "foreign_wg.conf")) {
		t.Fatal("foreign lines changed")
	}
	if ifc, _ := server.Interface(); len(ifc.Meta) != 0 {
		t.Fatal("--endpoint must not be written to the server config")
	}

	e = newEnv(t)
	path = setupAWG0(t, e, false)
	e.mustRun(0, "--add-client", "--interface", "awg0", "--id", "v6", "--allowed-ips", "::/0", "--endpoint", "2001:db8::1")
	if p := clientConfig(t, e.stdout.String()).Peers()[0]; p.Get("Endpoint") != "[2001:db8::1]:51820" {
		t.Fatalf("Endpoint = %q", p.Get("Endpoint"))
	}
	if ifc, _ := conf.Parse([]byte(e.read(path))).Interface(); ifc.Meta["endpoint"] != "vpn.example.com" {
		t.Fatal("server endpoint changed")
	}
}

// SPEC 6.5 п.7: --qr — QR-код после конфига; не помещается — конфиг выводится, затем код 3.
func TestAddClientQR(t *testing.T) {
	e := newEnv(t)
	setupAWG0(t, e, false)
	e.mustRun(0, "--add-client", "--interface", "awg0", "--id", "q1", "--allowed-ips", "0.0.0.0/0", "--qr")
	out := e.stdout.String()
	if !strings.Contains(out, "PersistentKeepalive = 25\n\n") || !strings.Contains(out, "█") {
		t.Fatalf("no QR after config:\n%s", out)
	}

	e = newEnv(t)
	path := e.awgConf("awg0")
	big := strings.Replace(fixture(t, "create_awg_i1.golden"), "<b 0xf6ab3267fa><r 16>", "<b 0x"+strings.Repeat("ab", 1500)+">", 1)
	e.write(path, big)
	e.mustRun(errs.Invalid, "--add-client", "--interface", "awg0", "--id", "q2", "--allowed-ips", "0.0.0.0/0", "--qr")
	if !strings.Contains(e.stdout.String(), "[Peer]") || !strings.Contains(e.stderr.String(), "config too large for QR") {
		t.Fatalf("stdout %q\nstderr %q", e.stdout, e.stderr)
	}
	if _, ok := conf.Parse([]byte(e.read(path))).PeerByID("q2"); !ok {
		t.Fatal("client must stay added")
	}
}
