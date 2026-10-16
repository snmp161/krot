package conf

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func read(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func assertBytes(t *testing.T, got, want []byte) {
	t.Helper()
	if !bytes.Equal(got, want) {
		t.Fatalf("output mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// SPEC 8.2: round-trip — прочитать и записать без изменений → байт-в-байт тот же файл,
// включая чужие конфиги с произвольными комментариями (SPEC 5.3).
func TestRoundTrip(t *testing.T) {
	for _, name := range []string{"spec_example.conf", "foreign_wg.conf", "bare_locked.conf",
		"spec_example.lock.golden", "foreign_wg.add.golden"} {
		b := read(t, name)
		assertBytes(t, Parse(b).Bytes(), b)

		noNL := bytes.TrimSuffix(b, []byte("\n"))
		assertBytes(t, Parse(noNL).Bytes(), noNL)

		crlf := bytes.ReplaceAll(b, []byte("\n"), []byte("\r\n"))
		assertBytes(t, Parse(crlf).Bytes(), crlf)
	}
	for _, s := range []string{"", "\n", "\n\n", "# just a comment", "[Interface]"} {
		assertBytes(t, Parse([]byte(s)).Bytes(), []byte(s))
	}
}

// SPEC 5.1: метаданные интерфейса — строки `# krot:` перед [Interface].
func TestInterface(t *testing.T) {
	c := Parse(read(t, "spec_example.conf"))
	ifc, ok := c.Interface()
	if !ok {
		t.Fatal("no [Interface]")
	}
	wantMeta := map[string]string{"type": "awg2", "endpoint": "vpn.example.com", "dns": "1.1.1.1,9.9.9.9"}
	if len(ifc.Meta) != len(wantMeta) {
		t.Fatalf("meta = %v", ifc.Meta)
	}
	for k, v := range wantMeta {
		if ifc.Meta[k] != v {
			t.Errorf("meta[%s] = %q; want %q", k, ifc.Meta[k], v)
		}
	}
	if got := ifc.Get("listenport"); got != "51820" {
		t.Errorf("ListenPort = %q", got)
	}
	if got := ifc.Get("H4"); got != "2001009901-2001534188" {
		t.Errorf("H4 = %q", got)
	}
	if got := ifc.List("Address"); !slices.Equal(got, []string{"10.8.0.1/24", "fd00:8::1/64"}) {
		t.Errorf("Address = %q", got)
	}
	var keys []string
	for _, kv := range ifc.Keys {
		keys = append(keys, kv.Key)
	}
	want := []string{"PrivateKey", "Address", "ListenPort", "MTU", "Jc", "Jmin", "Jmax",
		"S1", "S2", "S3", "S4", "H1", "H2", "H3", "H4"}
	if !slices.Equal(keys, want) {
		t.Errorf("keys = %q", keys)
	}
}

// SPEC 5.1, 5.3: peer'ы krot (активные и заблокированные) и чужие peer'ы.
func TestPeers(t *testing.T) {
	ps := Parse(read(t, "spec_example.conf")).Peers()
	if len(ps) != 2 {
		t.Fatalf("got %d peers", len(ps))
	}
	p := ps[0]
	if p.ID != "phone-sergei" || p.Created != "2026-10-17T01:30:00Z" || p.Locked || p.LockedAt != "" ||
		p.PublicKey() != "PHONEpublicKEYphonePUBLICkeyPHONEpublicKE=" ||
		p.PresharedKey() != "PHONEpskPHONEpskPHONEpskPHONEpskPHONEpskP=" ||
		!slices.Equal(p.AllowedIPs(), []string{"10.8.0.2/32", "fd00:8::2/128"}) {
		t.Errorf("peer 0 = %+v", p)
	}
	p = ps[1]
	if p.ID != "laptop-old" || p.Created != "2026-10-01T10:00:00Z" || !p.Locked ||
		p.LockedAt != "2026-10-16T12:00:00Z" || p.PublicKey() != "LAPTOPpublicKEYlaptopPUBLICkeyLAPTOPpubl=" ||
		p.PresharedKey() != "" || !slices.Equal(p.AllowedIPs(), []string{"10.8.0.3/32"}) {
		t.Errorf("peer 1 = %+v", p)
	}
	if _, ok := Parse(read(t, "spec_example.conf")).PeerByID("laptop-old"); !ok {
		t.Error("PeerByID(laptop-old) not found")
	}
	if _, ok := Parse(read(t, "spec_example.conf")).PeerByID("nope"); ok {
		t.Error("PeerByID(nope) found")
	}

	c := Parse(read(t, "foreign_wg.conf"))
	ifc, _ := c.Interface()
	if len(ifc.Meta) != 0 {
		t.Errorf("foreign meta = %v", ifc.Meta)
	}
	if got := ifc.List("Address"); !slices.Equal(got, []string{"10.9.0.1/24", "fd00:9::1/64"}) {
		t.Errorf("foreign Address = %q", got)
	}
	ps = c.Peers()
	if len(ps) != 2 {
		t.Fatalf("foreign: got %d peers", len(ps))
	}
	if ps[0].ID != "" || ps[0].PublicKey() != "xTIBA5rboUvnH4htodjb6e697QjLERt1NAB4mZqp8Dg=" ||
		!slices.Equal(ps[0].AllowedIPs(), []string{"10.9.0.2/32", "fd00:9::2/128"}) {
		t.Errorf("foreign peer 0 = %+v", ps[0])
	}
	if ps[1].ID != "" || ps[1].PublicKey() != "TrMvSoP4jYQlY6RIzBgbssQqY3vxI2Pi+y71lOWWXX0=" ||
		!slices.Equal(ps[1].AllowedIPs(), []string{"10.9.0.3/32", "192.168.50.0/24"}) ||
		ps[1].Get("PersistentKeepalive") != "25" || ps[1].Get("endpoint") != "203.0.113.7:51820" {
		t.Errorf("foreign peer 1 = %+v", ps[1])
	}
	if _, ok := c.PeerByID(""); ok {
		t.Error("PeerByID(\"\") must not match foreign peers")
	}

	ps = Parse(read(t, "bare_locked.conf")).Peers()
	if len(ps) != 2 || !ps[0].Locked || !slices.Equal(ps[0].AllowedIPs(), []string{"10.0.0.2/32"}) ||
		ps[1].Locked || ps[1].PublicKey() != "BBB" {
		t.Errorf("bare locked peers = %+v", ps)
	}
}

func newTablet() NewPeer {
	return NewPeer{
		ID:         "tablet",
		Created:    "2026-10-17T03:00:00Z",
		PublicKey:  "TABLETpublicKEYtabletPUBLICkeyTABLETpubl=",
		AllowedIPs: []string{"10.8.0.4/32", "fd00:8::4/128"},
	}
}

// SPEC 8.2: добавление, удаление, блокировка, разблокировка peer'ов на golden-файлах.
func TestOperations(t *testing.T) {
	orig := read(t, "spec_example.conf")

	c := Parse(orig)
	c.AddPeer(newTablet())
	assertBytes(t, c.Bytes(), read(t, "spec_example.add.golden"))
	if p, ok := c.PeerByID("tablet"); !ok || p.PublicKey() != newTablet().PublicKey {
		t.Error("added peer not visible after AddPeer")
	}

	c = Parse(orig)
	if err := c.RemovePeer("phone-sergei"); err != nil {
		t.Fatal(err)
	}
	assertBytes(t, c.Bytes(), read(t, "spec_example.delete.golden"))

	c = Parse(orig)
	if err := c.LockPeer("phone-sergei", "2026-10-17T02:00:00Z"); err != nil {
		t.Fatal(err)
	}
	assertBytes(t, c.Bytes(), read(t, "spec_example.lock.golden"))

	c = Parse(orig)
	if err := c.UnlockPeer("laptop-old"); err != nil {
		t.Fatal(err)
	}
	assertBytes(t, c.Bytes(), read(t, "spec_example.unlock.golden"))

	// Удаление заблокированного клиента допустимо (SPEC 6.6).
	c = Parse(orig)
	if err := c.RemovePeer("laptop-old"); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.PeerByID("laptop-old"); ok || len(c.Peers()) != 1 {
		t.Error("locked peer not removed")
	}
	if strings.Contains(string(c.Bytes()), "laptop") {
		t.Errorf("leftovers of removed peer:\n%s", c.Bytes())
	}

	c = Parse(read(t, "bare_locked.conf"))
	if err := c.UnlockPeer("a"); err != nil {
		t.Fatal(err)
	}
	assertBytes(t, c.Bytes(), read(t, "bare_locked.unlock.golden"))
}

// Обратимость: add→delete и lock→unlock возвращают исходный файл байт-в-байт.
func TestReversible(t *testing.T) {
	for _, name := range []string{"spec_example.conf", "foreign_wg.conf"} {
		for _, crlf := range []bool{false, true} {
			orig := read(t, name)
			if crlf {
				orig = bytes.ReplaceAll(orig, []byte("\n"), []byte("\r\n"))
			}
			c := Parse(orig)
			c.AddPeer(newTablet())
			if crlf && !bytes.Contains(c.Bytes(), []byte("[Peer]\r\nPublicKey = TABLET")) {
				t.Errorf("%s: added lines must use CRLF:\n%q", name, c.Bytes())
			}
			if err := c.LockPeer("tablet", "2026-10-17T04:00:00Z"); err != nil {
				t.Fatal(err)
			}
			if p, _ := c.PeerByID("tablet"); !p.Locked || p.PublicKey() != newTablet().PublicKey {
				t.Errorf("%s: lock is not visible: %+v", name, p)
			}
			if err := c.UnlockPeer("tablet"); err != nil {
				t.Fatal(err)
			}
			if err := c.RemovePeer("tablet"); err != nil {
				t.Fatal(err)
			}
			assertBytes(t, c.Bytes(), orig)
		}
	}
	orig := read(t, "spec_example.conf")
	c := Parse(orig)
	if err := c.LockPeer("phone-sergei", "2026-10-17T02:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if err := c.UnlockPeer("phone-sergei"); err != nil {
		t.Fatal(err)
	}
	assertBytes(t, c.Bytes(), orig)
}

// SPEC 5.3: при записи все чужие строки, комментарии и порядок сохраняются.
func TestForeignPreserved(t *testing.T) {
	c := Parse(read(t, "foreign_wg.conf"))
	c.AddPeer(NewPeer{
		ID:           "new-client",
		Created:      "2026-10-17T03:00:00Z",
		PublicKey:    "NEWpublicKEYnewPUBLICkeyNEWpublicKEYnewPU=",
		PresharedKey: "NEWpskNEWpskNEWpskNEWpskNEWpskNEWpskNEWp=",
		AllowedIPs:   []string{"10.9.0.4/32"},
	})
	assertBytes(t, c.Bytes(), read(t, "foreign_wg.add.golden"))
	if n := len(c.Peers()); n != 3 {
		t.Errorf("got %d peers", n)
	}
}

func TestOperationErrors(t *testing.T) {
	c := Parse(read(t, "spec_example.conf"))
	if err := c.RemovePeer("nope"); !errors.Is(err, ErrPeerNotFound) {
		t.Errorf("RemovePeer(nope) = %v", err)
	}
	if err := c.LockPeer("nope", "x"); !errors.Is(err, ErrPeerNotFound) {
		t.Errorf("LockPeer(nope) = %v", err)
	}
	if err := c.UnlockPeer("nope"); !errors.Is(err, ErrPeerNotFound) {
		t.Errorf("UnlockPeer(nope) = %v", err)
	}
	if err := c.LockPeer("laptop-old", "x"); !errors.Is(err, ErrAlreadyLocked) {
		t.Errorf("LockPeer(locked) = %v", err)
	}
	if err := c.UnlockPeer("phone-sergei"); !errors.Is(err, ErrNotLocked) {
		t.Errorf("UnlockPeer(active) = %v", err)
	}
	assertBytes(t, c.Bytes(), read(t, "spec_example.conf"))
}

// SPEC 5.1: формат конфига, создаваемого --create (закомментированные хуки — всегда).
func TestRenderServer(t *testing.T) {
	awg2 := Server{
		Meta:       []KV{{"type", "awg2"}, {"endpoint", "vpn.example.com"}, {"dns", "1.1.1.1,9.9.9.9"}},
		PrivateKey: "SERVERprivateKEYserverPRIVATEkeySERVERpriv=",
		Addresses:  []string{"10.8.0.1/24", "fd00:8::1/64"},
		ListenPort: 51820,
		MTU:        1376,
		Params: []KV{{"Jc", "6"}, {"Jmin", "64"}, {"Jmax", "512"}, {"S1", "43"}, {"S2", "117"},
			{"S3", "31"}, {"S4", "12"}, {"H1", "194847201-195371488"}, {"H2", "905128442-905652729"},
			{"H3", "1503002211-1503526498"}, {"H4", "2001009901-2001534188"}},
	}
	got := RenderServer(awg2)
	assertBytes(t, got, read(t, "render_awg2.golden"))
	ifc, _ := Parse(got).Interface()
	if ifc.Meta["type"] != "awg2" || ifc.Get("MTU") != "1376" {
		t.Errorf("rendered config does not parse back: %+v", ifc)
	}

	wg := Server{
		Meta:       []KV{{"type", "wg"}, {"endpoint", "203.0.113.1"}},
		PrivateKey: "SERVERprivateKEYserverPRIVATEkeySERVERpriv=",
		Addresses:  []string{"10.9.0.1/24"},
		ListenPort: 51821,
		MTU:        1420,
	}
	assertBytes(t, RenderServer(wg), read(t, "render_wg.golden"))
}
