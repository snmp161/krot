package awg

import (
	"time"
)

// Профили мимикрии (SPEC 7.6). Каждый генератор возвращает пакеты цепочки I1–Ik:
// конкретные байты с отмеченными переменными полями — модуль AmneziaWG пересчитывает
// их перед каждой отправкой рукопожатия.

func init() {
	mimicProfiles["dns"] = genDNSChain
	mimicProfiles["stun"] = genSTUNChain
	mimicProfiles["ntp"] = genNTPChain
	mimicProfiles["ssdp"] = genSSDPChain
	mimicProfiles["dtls"] = genDTLSChain
}

// hostname — правдоподобное имя хоста из букв (метки DNS/SNI: буквы всегда допустимы).
func hostname(g *gen) string {
	letters := func(n int) string {
		const abc = "abcdefghijklmnopqrstuvwxyz"
		s := make([]byte, n)
		for i := range s {
			s[i] = abc[g.int(0, len(abc)-1)]
		}
		return string(s)
	}
	tld := [...]string{"com", "net", "org", "io"}
	return letters(g.int(5, 9)) + "." + letters(g.int(3, 6)) + "." + tld[g.int(0, len(tld)-1)]
}

// genDNSChain — три запроса по одному имени: типы A/AAAA/HTTPS в случайном
// порядке (резолвер спрашивает разное, а не повторяет подряд), ID запроса —
// переменное поле (RFC 5452), OPT с размером UDP 1232.
func genDNSChain(g *gen, _ time.Time) ([]packet, error) {
	qtypes := []uint16{1, 28, 65}          // A, AAAA, HTTPS
	for i := len(qtypes) - 1; i > 0; i-- { // Fisher–Yates
		j := g.int(0, i)
		qtypes[i], qtypes[j] = qtypes[j], qtypes[i]
	}
	name := hostname(g)
	labels := splitLabels(name)
	var pkts []packet
	for _, qt := range qtypes {
		var p packet
		g.bytes(&p, 2)
		p.mark(tagRand, 2) // ID запроса — случаен в каждом запросе (RFC 5452)
		u16(&p, 0x0100)    // RD
		u16(&p, 1)         // QDCOUNT
		u16(&p, 0)         // ANCOUNT
		u16(&p, 0)         // NSCOUNT
		u16(&p, 1)         // ARCOUNT
		for _, label := range labels {
			p.buf = append(p.buf, byte(len(label)))
			p.buf = append(p.buf, label...)
		}
		p.buf = append(p.buf, 0) // корень
		u16(&p, qt)
		u16(&p, 1) // IN
		// OPT (RFC 6891): корень, тип 41, класс = размер UDP, TTL 0, RDLEN 0.
		p.buf = append(p.buf, 0)
		u16(&p, 41)
		u16(&p, 1232)
		u32(&p, 0)
		u16(&p, 0)
		pkts = append(pkts, p)
	}
	return pkts, nil
}

func splitLabels(name string) []string {
	var out []string
	start := 0
	for i := 0; i < len(name); i++ {
		if name[i] == '.' {
			out = append(out, name[start:i])
			start = i + 1
		}
	}
	return append(out, name[start:])
}

// genSTUNChain — Binding Request без FINGERPRINT: CRC32 атрибута покрыл бы всё
// сообщение, и пакет был бы обречён на повторы байт-в-байт (SPEC 7.6). Без него
// запрос остаётся корректным (RFC 5389: FINGERPRINT — RECOMMENDED, не REQUIRED),
// а transaction ID, PRIORITY и ICE-CONTROLLING пересчитываются при каждой отправке.
func genSTUNChain(g *gen, _ time.Time) ([]packet, error) {
	var p packet
	u16(&p, 0x0001) // Binding Request
	lenOff := len(p.buf)
	u16(&p, 0) // LENGTH — заполняется после сборки атрибутов
	u32(&p, 0x2112A442)
	g.bytes(&p, 12)
	p.mark(tagRand, 12)                    // transaction ID
	stunAttr(&p, 0x0006, func(v *packet) { // USERNAME "<8 букв>:<host>"
		g.letters(v, 8)
		v.mark(tagChars, 8)
		v.buf = append(v.buf, ':')
		v.buf = append(v.buf, hostname(g)...)
	})
	stunAttr(&p, 0x0024, func(v *packet) { // PRIORITY (RFC 8445)
		g.bytes(v, 4)
		v.mark(tagRand, 4)
	})
	stunAttr(&p, 0x802A, func(v *packet) { // ICE-CONTROLLING
		g.bytes(v, 8)
		v.mark(tagRand, 8)
	})
	setU16(p.buf[lenOff:], uint16(len(p.buf)-20))
	return []packet{p}, nil
}

// stunAttr добавляет атрибут STUN: тип, длина, значение (сборка build), паддинг до 4.
func stunAttr(p *packet, typ uint16, build func(v *packet)) {
	var v packet
	build(&v)
	u16(p, typ)
	u16(p, uint16(len(v.buf)))
	valOff := len(p.buf)
	p.buf = append(p.buf, v.buf...)
	for _, d := range v.dyn {
		p.dyn = append(p.dyn, dynField{d.kind, valOff + d.off, d.length})
	}
	for range (4 - len(v.buf)%4) % 4 {
		p.buf = append(p.buf, 0)
	}
}

func setU16(b []byte, v uint16) { b[0], b[1] = byte(v>>8), byte(v) }

// ntpEpoch — сдвиг Unix → NTP (эра 1900).
const ntpEpoch = 2208988800

// genNTPChain — запрос клиента без синхронизации (stratum 0, refid "INIT") —
// снимок sntp при старте. Переменные — дробные части меток: у живых клиентов
// они случайны. Секунды фиксируются при генерации: тег <t> даёт Unix-time, а
// NTP живёт в эре 1900 — известное ограничение (SPEC 7.6).
func genNTPChain(g *gen, now time.Time) ([]packet, error) {
	var p packet
	p.buf = append(p.buf, 0x23, 0x00, 0x06, 0xEC) // LI=0, VN=4, Mode=3; stratum; poll; precision
	u32(&p, 0x00000100)                           // root delay
	u32(&p, 0x00000100)                           // root dispersion
	p.buf = append(p.buf, "INIT"...)
	sec := now.Unix()
	u32(&p, uint32(sec-int64(g.int(30, 900)))+ntpEpoch) // Reference Timestamp
	g.bytes(&p, 4)
	p.mark(tagRand, 4) // дробная часть Reference
	for range 16 {     // Origin + Receive = 0
		p.buf = append(p.buf, 0)
	}
	u32(&p, uint32(sec)+ntpEpoch) // Transmit Timestamp
	g.bytes(&p, 4)
	p.mark(tagRand, 4) // дробная часть Transmit
	return []packet{p}, nil
}

// Пулы правдоподобных значений SSDP; различаются между клиентами.
var ssdpTargets = []string{
	"urn:schemas-upnp-org:service:WANIPConnection:1",
	"urn:schemas-upnp-org:service:WANPPPConnection:1",
	"urn:schemas-upnp-org:device:InternetGatewayDevice:1",
	"upnp:rootdevice",
	"ssdp:all",
}

var ssdpUserAgents = []string{
	"macOS/14.7.6 UPnP/1.1 ControlPoint/1.0",
	"Ubuntu/24.04 UPnP/1.1 MiniUPnPc/2.2.5",
	"Windows NT/10.0 UPnP/1.1 SSDPClient/1.0",
	"Google Chrome/130.0 UPnP/1.1",
}

// genSSDPChain — M-SEARCH целиком статичен: реальные устройства повторяют его
// дословно (SPEC 7.6). Между клиентами различаются ST, MX и User-Agent.
func genSSDPChain(g *gen, _ time.Time) ([]packet, error) {
	s := "M-SEARCH * HTTP/1.1\r\n" +
		"HOST: 239.255.255.250:1900\r\n" +
		"MAN: \"ssdp:discover\"\r\n" +
		"ST: " + ssdpTargets[g.int(0, len(ssdpTargets)-1)] + "\r\n" +
		"MX: " + itoa(g.int(1, 5)) + "\r\n" +
		"USER-AGENT: " + ssdpUserAgents[g.int(0, len(ssdpUserAgents)-1)] + "\r\n" +
		"ACCEPT-LANGUAGE: en-US,en;q=0.9\r\n" +
		"\r\n"
	return []packet{{buf: []byte(s)}}, nil
}

// genDTLSChain — ClientHello первого полёта DTLS 1.2 (WebRTC-стиль): ничем не
// подписан и не зашифрован, поэтому client_random и session_id — переменные поля.
func genDTLSChain(g *gen, _ time.Time) ([]packet, error) {
	host := hostname(g)
	var body packet
	u16(&body, 0xFEFD) // client_version: DTLS 1.2
	g.bytes(&body, 32)
	body.mark(tagRand, 32) // client_random
	body.buf = append(body.buf, 32)
	g.bytes(&body, 32)
	body.mark(tagRand, 32)         // session_id
	body.buf = append(body.buf, 0) // cookie: пустой (нет HelloVerifyRequest)
	suites := []uint16{0xC02B, 0xC02F, 0xCCA9, 0xC02C, 0x009C, 0x009D}
	u16(&body, uint16(2*len(suites)))
	for _, s := range suites {
		u16(&body, s)
	}
	body.buf = append(body.buf, 1, 0) // compression: null

	var ext packet
	extU16(&ext, 0x0000, func(v *packet) { // server_name
		var list packet
		list.buf = append(list.buf, 0) // тип: hostname
		u16(&list, uint16(len(host)))
		list.buf = append(list.buf, host...)
		u16(v, uint16(len(list.buf)))
		v.buf = append(v.buf, list.buf...)
	})
	extU16(&ext, 0x000A, func(v *packet) { // supported_groups
		u16(v, 6)
		u16(v, 0x001D) // x25519
		u16(v, 0x0017) // secp256r1
		u16(v, 0x0018) // secp384r1
	})
	extU16(&ext, 0x000D, func(v *packet) { // signature_algorithms
		algs := []uint16{0x0403, 0x0503, 0x0804, 0x0401, 0x0501, 0x0201}
		u16(v, uint16(2*len(algs)))
		for _, a := range algs {
			u16(v, a)
		}
	})
	extU16(&ext, 0x002F, func(v *packet) { // use_srtp (RFC 5764)
		u16(v, 2)                // длина списка профилей
		u16(v, 0x0001)           // SRTP_AES128_CM_SHA1_80
		v.buf = append(v.buf, 0) // MKI length
	})
	extU16(&ext, 0x0017, func(v *packet) {}) // extended_master_secret
	u16(&body, uint16(len(ext.buf)))
	body.buf = append(body.buf, ext.buf...)

	// Record layer + handshake header; поля body сдвигаются на их длину.
	var p packet
	p.buf = append(p.buf, 0x16, 0xFE, 0xFD) // handshake, DTLS 1.2
	u16(&p, 0)                              // epoch
	p.buf = append(p.buf, 0, 0, 0, 0, 0, 0) // sequence_number первого полёта
	u16(&p, uint16(12+len(body.buf)))       // длина handshake-сообщения
	p.buf = append(p.buf, 0x01)             // ClientHello
	u24(&p, uint32(len(body.buf)))
	u16(&p, 0) // message_seq
	u24(&p, 0) // fragment_offset
	u24(&p, uint32(len(body.buf)))
	p.buf = append(p.buf, body.buf...)
	const shift = 13 + 12 // record header + handshake header
	for _, d := range body.dyn {
		p.dyn = append(p.dyn, dynField{d.kind, shift + d.off, d.length})
	}
	return []packet{p}, nil
}

// extU16 добавляет расширение: тип, длина, значение (сборка build).
func extU16(p *packet, typ uint16, build func(v *packet)) {
	var v packet
	build(&v)
	u16(p, typ)
	u16(p, uint16(len(v.buf)))
	p.buf = append(p.buf, v.buf...)
}
