package awg

import (
	"encoding/binary"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
	"time"
)

// SPEC 4.4, 6.12: набор профилей мимикрии фиксирован.
func TestMimicProfiles(t *testing.T) {
	want := []string{"dns", "stun", "ntp", "ssdp", "dtls"}
	if got := MimicProfiles(); !slices.Equal(got, want) {
		t.Fatalf("MimicProfiles() = %v; want %v", got, want)
	}
}

// SPEC 6.12, 7.6: для каждого профиля цепочка генерируется, проходит V-05,
// укладывается в V-06; длины цепочек по политике 7.6.
func TestGenerateMimicChainAllProfiles(t *testing.T) {
	wantLen := map[string]int{"dns": 3, "stun": 1, "ntp": 1, "ssdp": 1, "dtls": 1}
	for _, p := range MimicProfiles() {
		t.Run(p, func(t *testing.T) {
			chain, err := GenerateMimicChain(p, seeded(42), time.Unix(1_700_000_000, 0))
			if err != nil {
				t.Fatalf("GenerateMimicChain(%s): %v", p, err)
			}
			if len(chain) != wantLen[p] {
				t.Fatalf("%s: chain length = %d; want %d", p, len(chain), wantLen[p])
			}
			total := 0
			for _, cps := range chain {
				if err := ValidateCPS(cps); err != nil {
					t.Fatalf("%s: generated CPS fails V-05: %v\n%s", p, err, cps)
				}
				total += len(cps)
			}
			if total > MaxITotal {
				t.Fatalf("%s: chain is %d chars; want ≤ %d (V-06)", p, total, MaxITotal)
			}
		})
	}
}

// SPEC 7.6: генерация детерминирована при фиксированном seed (AGENTS.md).
func TestGenerateMimicChainDeterministic(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	for _, p := range MimicProfiles() {
		a, err := GenerateMimicChain(p, seeded(7), now)
		if err != nil {
			t.Fatal(err)
		}
		b, err := GenerateMimicChain(p, seeded(7), now)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(a, b) {
			t.Fatalf("%s: same seed gave different chains", p)
		}
		c, err := GenerateMimicChain(p, seeded(8), now)
		if err != nil {
			t.Fatal(err)
		}
		if slices.Equal(a, c) {
			t.Fatalf("%s: different seeds gave the same chain", p)
		}
	}
}

// SPEC 6.12: неизвестный профиль — ошибка (в CLI она же даёт код 3).
func TestGenerateMimicChainUnknown(t *testing.T) {
	if _, err := GenerateMimicChain("quic", seeded(1), time.Now()); err == nil {
		t.Fatal("quic: want error, got nil")
	}
	if _, err := GenerateMimicChain("bogus", seeded(1), time.Now()); err == nil {
		t.Fatal("bogus: want error, got nil")
	}
}

// instantiate заполняет шаблон CPS конкретными байтами (для проверок структуры).
// Порядок тегов сохраняется; значения — детерминированные по seed.
func instantiate(t *testing.T, cps string, seed uint64) []byte {
	t.Helper()
	var out []byte
	rest := cps
	rng := rand.New(rand.NewPCG(seed, 0))
	for len(rest) > 0 {
		if rest[0] != '<' {
			t.Fatalf("bad CPS %q", cps)
		}
		end := strings.IndexByte(rest, '>')
		if end < 0 {
			t.Fatalf("bad CPS %q", cps)
		}
		parts := strings.Fields(rest[1:end])
		rest = rest[end+1:]
		switch parts[0] {
		case "b":
			h := strings.TrimPrefix(parts[1], "0x")
			for i := 0; i < len(h); i += 2 {
				var v byte
				for _, d := range h[i : i+2] {
					v <<= 4
					switch {
					case d >= '0' && d <= '9':
						v |= byte(d - '0')
					case d >= 'a' && d <= 'f':
						v |= byte(d - 'a' + 10)
					default:
						t.Fatalf("bad hex %q", h)
					}
				}
				out = append(out, v)
			}
		case "t":
			out = binary.BigEndian.AppendUint32(out, 0x66554433)
		case "r":
			n := atoiOrDie(t, parts[1])
			buf := make([]byte, n)
			rng.Read(buf)
			out = append(out, buf...)
		case "rc":
			const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
			n := atoiOrDie(t, parts[1])
			for range n {
				out = append(out, letters[rng.IntN(len(letters))])
			}
		case "rd":
			n := atoiOrDie(t, parts[1])
			for range n {
				out = append(out, byte('0'+rng.IntN(10)))
			}
		default:
			t.Fatalf("unknown tag %q", parts[0])
		}
	}
	return out
}

func atoiOrDie(t *testing.T, s string) int {
	t.Helper()
	n := 0
	for _, d := range s {
		if d < '0' || d > '9' {
			t.Fatalf("bad number %q", s)
		}
		n = n*10 + int(d-'0')
	}
	return n
}

// dynRegions вытаскивает из шаблона смещения и длины динамических полей.
func dynRegions(t *testing.T, cps string) (length int, regions []struct{ off, n int }) {
	t.Helper()
	for len(cps) > 0 {
		end := strings.IndexByte(cps, '>')
		parts := strings.Fields(cps[1:end])
		cps = cps[end+1:]
		var n int
		switch parts[0] {
		case "b":
			n = len(strings.TrimPrefix(parts[1], "0x")) / 2
		case "t":
			n = 4
		default:
			n = atoiOrDie(t, parts[1])
			regions = append(regions, struct{ off, n int }{length, n})
		}
		length += n
	}
	return length, regions
}

// SPEC 7.6 п.2: каждое динамическое поле встречается в пакете ровно один раз:
// две копии поля получили бы разные значения тегов на отправке.
func TestMimicDynamicFieldsUnique(t *testing.T) {
	for _, p := range MimicProfiles() {
		t.Run(p, func(t *testing.T) {
			chain, err := GenerateMimicChain(p, seeded(11), time.Unix(1_700_000_000, 0))
			if err != nil {
				t.Fatal(err)
			}
			for i, cps := range chain {
				total, regions := dynRegions(t, cps)
				pkt := instantiate(t, cps, uint64(i+1))
				if len(pkt) != total {
					t.Fatalf("packet %d: len %d; want %d", i, len(pkt), total)
				}
				for _, r := range regions {
					if r.off+r.n > len(pkt) {
						t.Fatalf("region %+v beyond packet %d (%d bytes)", r, i, len(pkt))
					}
					val := pkt[r.off : r.off+r.n]
					if got := strings.Count(string(pkt), string(val)); got != 1 {
						t.Fatalf("%s packet %d: dynamic value %x occurs %d times; want 1 (SPEC 7.6 п.2)",
							p, i, val, got)
					}
				}
			}
		})
	}
}

// SPEC 7.6: DNS — три запроса (A/AAAA/HTTPS в случайном порядке, без повторов
// подряд), ID запроса динамический, OPT с размером UDP 1232.
func TestMimicDNS(t *testing.T) {
	chain, err := GenerateMimicChain("dns", seeded(3), time.Unix(1_700_000_000, 0))
	if err != nil {
		t.Fatal(err)
	}
	if len(chain) != 3 {
		t.Fatalf("chain length = %d; want 3", len(chain))
	}
	types := map[uint16]bool{}
	for i, cps := range chain {
		pkt := instantiate(t, cps, uint64(i))
		if len(pkt) < 30 {
			t.Fatalf("packet %d too short: %d", i, len(pkt))
		}
		if binary.BigEndian.Uint16(pkt[2:4]) != 0x0100 { // RD
			t.Fatalf("packet %d flags = %#x; want 0x0100", i, binary.BigEndian.Uint16(pkt[2:4]))
		}
		if qd := binary.BigEndian.Uint16(pkt[4:6]); qd != 1 {
			t.Fatalf("packet %d QDCOUNT = %d; want 1", i, qd)
		}
		if an, ns := binary.BigEndian.Uint16(pkt[6:8]), binary.BigEndian.Uint16(pkt[8:10]); an != 0 || ns != 0 {
			t.Fatalf("packet %d AN/NS = %d/%d; want 0/0", i, an, ns)
		}
		if ar := binary.BigEndian.Uint16(pkt[10:12]); ar != 1 {
			t.Fatalf("packet %d ARCOUNT = %d; want 1 (OPT)", i, ar)
		}
		// QNAME: последовательность меток, заканчивается 0x00.
		j := 12
		for pkt[j] != 0 {
			j += 1 + int(pkt[j])
		}
		if j+5 > len(pkt) {
			t.Fatalf("packet %d: truncated question", i)
		}
		qt := binary.BigEndian.Uint16(pkt[j+1 : j+3])
		if qt != 1 && qt != 28 && qt != 65 {
			t.Fatalf("packet %d qtype = %d; want 1 (A), 28 (AAAA) or 65 (HTTPS)", i, qt)
		}
		types[qt] = true
		if cl := binary.BigEndian.Uint16(pkt[j+3 : j+5]); cl != 1 {
			t.Fatalf("packet %d qclass = %d; want 1 (IN)", i, cl)
		}
		// OPT: имя-корень, тип 41, класс = размер UDP.
		rest := pkt[j+5:]
		if len(rest) != 11 || rest[0] != 0 || binary.BigEndian.Uint16(rest[1:3]) != 41 {
			t.Fatalf("packet %d: bad OPT record %x", i, rest)
		}
		if udp := binary.BigEndian.Uint16(rest[3:5]); udp != 1232 {
			t.Fatalf("packet %d OPT UDP size = %d; want 1232", i, udp)
		}
	}
	for _, qt := range []uint16{1, 28, 65} {
		if !types[qt] {
			t.Fatalf("qtype %d not used in chain", qt)
		}
	}
	if !strings.Contains(chain[0], "<r 2>") {
		t.Fatalf("query ID must be dynamic:\n%s", chain[0])
	}
}

// SPEC 7.6: STUN — Binding Request с динамическими transaction ID, USERNAME и
// ICE-CONTROLLING; без FINGERPRINT (иначе CRC32 сделала бы пакет статичным).
func TestMimicSTUN(t *testing.T) {
	chain, err := GenerateMimicChain("stun", seeded(5), time.Unix(1_700_000_000, 0))
	if err != nil {
		t.Fatal(err)
	}
	pkt := instantiate(t, chain[0], 1)
	if len(pkt) < 20 {
		t.Fatalf("packet too short: %d", len(pkt))
	}
	if mt := binary.BigEndian.Uint16(pkt[0:2]); mt != 0x0001 {
		t.Fatalf("message type = %#x; want 0x0001 (Binding Request)", mt)
	}
	ml := int(binary.BigEndian.Uint16(pkt[2:4]))
	if ml != len(pkt)-20 || ml%4 != 0 {
		t.Fatalf("message length = %d; want %d, кратное 4", ml, len(pkt)-20)
	}
	if cookie := binary.BigEndian.Uint32(pkt[4:8]); cookie != 0x2112A442 {
		t.Fatalf("magic cookie = %#x; want 0x2112A442", cookie)
	}
	var seen map[uint16]bool = map[uint16]bool{}
	for off := 20; off+4 <= len(pkt); {
		at := binary.BigEndian.Uint16(pkt[off : off+2])
		al := int(binary.BigEndian.Uint16(pkt[off+2 : off+4]))
		seen[at] = true
		off += 4 + al + (4-al%4)%4
	}
	if off != len(pkt) {
		t.Fatalf("attrs end at %d; packet len %d", off, len(pkt))
	}
	for _, at := range []uint16{0x0006, 0x0024, 0x802A} { // USERNAME, PRIORITY, ICE-CONTROLLING
		if !seen[at] {
			t.Fatalf("attr %#x missing (have %v)", at, seen)
		}
	}
	if seen[0x8028] {
		t.Fatal("FINGERPRINT present; packet would be frozen by CRC32 (SPEC 7.6)")
	}
	// USERNAME — "<8 hex>:<host>": hex-часть динамическая, буквенная (<rc>).
	if !strings.Contains(chain[0], "<rc 8>") {
		t.Fatalf("USERNAME token must be dynamic:\n%s", chain[0])
	}
	if !strings.Contains(chain[0], "<r 12>") || !strings.Contains(chain[0], "<r 8>") {
		t.Fatalf("transaction ID / ICE-CONTROLLING must be dynamic:\n%s", chain[0])
	}
}

// SPEC 7.6: NTP — 48 байт, клиент без синхронизации (stratum 0, "INIT"),
// динамические дробные части меток, reference раньше transmit на 30–900 с.
func TestMimicNTP(t *testing.T) {
	chain, err := GenerateMimicChain("ntp", seeded(9), time.Unix(1_700_000_000, 0))
	if err != nil {
		t.Fatal(err)
	}
	pkt := instantiate(t, chain[0], 1)
	if len(pkt) != 48 {
		t.Fatalf("len = %d; want 48", len(pkt))
	}
	if pkt[0] != 0x23 { // LI=0, VN=4, Mode=3 (client)
		t.Fatalf("first byte = %#x; want 0x23", pkt[0])
	}
	if pkt[1] != 0 || pkt[2] != 6 || pkt[3] != 0xEC {
		t.Fatalf("stratum/poll/precision = %d/%d/%#x; want 0/6/0xEC", pkt[1], pkt[2], pkt[3])
	}
	if string(pkt[12:16]) != "INIT" {
		t.Fatalf("refid = %q; want INIT", pkt[12:16])
	}
	ref := binary.BigEndian.Uint32(pkt[16:20])
	tx := binary.BigEndian.Uint32(pkt[40:44])
	if d := tx - ref; d < 30 || d > 900 {
		t.Fatalf("transmit - reference = %d s; want 30..900", d)
	}
	if !strings.Contains(chain[0], "<r 4>") {
		t.Fatalf("timestamp fractions must be dynamic:\n%s", chain[0])
	}
}

// SPEC 7.6: SSDP — статичный M-SEARCH (реальные устройства повторяют его
// дословно), переменные части меняются между клиентами.
func TestMimicSSDP(t *testing.T) {
	a, err := GenerateMimicChain("ssdp", seeded(13), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	b, err := GenerateMimicChain("ssdp", seeded(14), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	pkt := instantiate(t, a[0], 1)
	s := string(pkt)
	for _, want := range []string{
		"M-SEARCH * HTTP/1.1\r\n",
		"HOST: 239.255.255.250:1900\r\n",
		"MAN: \"ssdp:discover\"\r\n",
		"ST: urn:",
		"USER-AGENT: ",
		"ACCEPT-LANGUAGE: en-US,en;q=0.9\r\n",
		"\r\n\r\n",
	} {
		if !strings.Contains(s, want) {
			t.Fatalf("SSDP packet lacks %q:\n%s", want, s)
		}
	}
	if !strings.HasSuffix(s, "\r\n\r\n") {
		t.Fatalf("packet must end with empty line:\n%q", s)
	}
	if mx := s[strings.Index(s, "MX: ")+4:]; mx[0] < '1' || mx[0] > '5' {
		t.Fatalf("MX = %q; want 1..5", mx[:1])
	}
	if !strings.Contains(a[0], "<b 0x") || strings.Contains(a[0], "<r ") {
		t.Fatalf("SSDP template must be fully static:\n%s", a[0])
	}
	if a[0] == b[0] {
		t.Fatal("different seeds gave the same SSDP packet")
	}
}

// SPEC 7.6: DTLS — ClientHello первого полёта: record и handshake заголовки
// согласованы, client_random и session_id динамические.
func TestMimicDTLS(t *testing.T) {
	chain, err := GenerateMimicChain("dtls", seeded(17), time.Unix(1_700_000_000, 0))
	if err != nil {
		t.Fatal(err)
	}
	cps := chain[0]
	pkt := instantiate(t, cps, 1)
	if len(pkt) < 13+25 {
		t.Fatalf("packet too short: %d", len(pkt))
	}
	if pkt[0] != 0x16 || pkt[1] != 0xFE || pkt[2] != 0xFD {
		t.Fatalf("record header = % x; want 16 fe fd (handshake, DTLS 1.2)", pkt[:3])
	}
	if binary.BigEndian.Uint16(pkt[3:5]) != 0 { // epoch
		t.Fatal("epoch must be 0 in the first flight")
	}
	recLen := int(binary.BigEndian.Uint16(pkt[11:13]))
	if recLen != len(pkt)-13 {
		t.Fatalf("record length = %d; want %d", recLen, len(pkt)-13)
	}
	hs := pkt[13:]
	if hs[0] != 0x01 {
		t.Fatalf("handshake type = %#x; want 1 (ClientHello)", hs[0])
	}
	bodyLen := int(hs[1])<<16 | int(hs[2])<<8 | int(hs[3])
	if bodyLen+25 != len(hs) {
		t.Fatalf("handshake length = %d; want %d", bodyLen, len(hs)-25)
	}
	if binary.BigEndian.Uint16(hs[4:6]) != 0 || hs[6] != 0 || hs[7] != 0 || hs[8] != 0 {
		t.Fatalf("message_seq/frag_offset must be 0: % x", hs[4:9])
	}
	fragLen := int(hs[9])<<16 | int(hs[10])<<8 | int(hs[11])
	if fragLen != bodyLen {
		t.Fatalf("fragment length = %d; want %d (single fragment)", fragLen, bodyLen)
	}
	body := hs[12:]
	if body[0] != 0xFE || body[1] != 0xFD {
		t.Fatalf("client_version = % x; want fe fd", body[:2])
	}
	if sidLen := body[34]; sidLen != 32 {
		t.Fatalf("session_id length = %d; want 32", sidLen)
	}
	n := strings.Count(cps, "<r 32>")
	if n != 2 {
		t.Fatalf("client_random and session_id must be dynamic: %d <r 32> tags\n%s", n, cps)
	}
}

// SPEC 6.12: цепочка записывается как I1..Ik — валидные ключи AmneziaWG.
func TestMimicChainKeys(t *testing.T) {
	chain, err := GenerateMimicChain("dns", seeded(21), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	for i, cps := range chain {
		if cps == "" {
			t.Fatalf("chain[%d] empty", i)
		}
		if len(cps) > maxCPSLen*4 { // каждый пакет умеренного размера
			t.Fatalf("chain[%d] too long: %d", i, len(cps))
		}
	}
}
