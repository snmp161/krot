package awg

import (
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"
)

// segTag — тег CPS переменного поля (SPEC 7.6).
type segTag int

const (
	tagRand   segTag = iota // <r N>   — N случайных байт
	tagChars                // <rc N>  — N случайных букв
	tagDigits               // <rd N>  — N случайных цифр
)

func (t segTag) String() string { return [...]string{"r", "rc", "rd"}[t] }

// dynField — переменное поле пакета: kind — тег CPS, off/length — позиция в buf.
type dynField struct {
	kind   segTag
	off    int
	length int
}

// packet — пакет-сигнатура: конкретные байты с отмеченными переменными полями.
type packet struct {
	buf []byte
	dyn []dynField
}

// mark отмечает последние length байт buf как переменное поле kind.
func (p *packet) mark(kind segTag, length int) {
	p.dyn = append(p.dyn, dynField{kind, len(p.buf) - length, length})
}

// MimicProfiles — профили мимикрии, которые генерирует krot (SPEC 4.4, 6.12).
func MimicProfiles() []string { return []string{"dns", "stun", "ntp", "ssdp", "dtls"} }

// mimicProfiles — реестр генераторов цепочек: профиль → пакеты I1..Ik (SPEC 7.6).
var mimicProfiles = map[string]func(g *gen, now time.Time) ([]packet, error){}

// GenerateMimicChain генерирует цепочку CPS-шаблонов I1–Ik для профиля мимикрии
// (SPEC 6.12, 7.6). r — crypto/rand.Reader в рабочем коде, в тестах — seeded.
// now используется профилями с метками времени (ntp).
func GenerateMimicChain(profile string, r io.Reader, now time.Time) ([]string, error) {
	genFn, ok := mimicProfiles[profile]
	if !ok {
		return nil, fmt.Errorf("unknown mimic profile %q: must be one of %s", profile, strings.Join(MimicProfiles(), ", "))
	}
	g := &gen{r: r}
	// SPEC 7.6 п.2: неуникальное вхождение переменного поля — повтор генерации
	// цепочки (значения случайны, поэтому пакет, а не вся цепочка, достаточно
	// перегенерировать целиком — это проще и дешевле усложнения генераторов).
	var lastErr error
	for range 8 {
		pkts, err := genFn(g, now)
		if err != nil {
			return nil, err
		}
		if g.err != nil {
			return nil, g.err
		}
		chain, err := mimicTemplates(pkts)
		if err == nil {
			return chain, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("could not generate mimic chain: %w", lastErr)
}

// mimicTemplates собирает CPS-шаблоны пакетов и проверяет бюджет V-06.
func mimicTemplates(pkts []packet) ([]string, error) {
	var chain []string
	total := 0
	for _, p := range pkts {
		cps, err := cpsTemplate(p)
		if err != nil {
			return nil, err
		}
		total += len(cps)
		chain = append(chain, cps)
	}
	if total > MaxITotal {
		return nil, fmt.Errorf("mimic chain is %d chars, exceeds I1–I5 budget %d (SPEC 7.1 V-06)", total, MaxITotal)
	}
	return chain, nil
}

// cpsTemplate собирает CPS-шаблон пакета: статика — <b 0x…>, переменные поля —
// <r N>/<rc N>/<rd N> (SPEC 7.6). Проверяет инварианты: поля не пересекаются,
// в пределах пакета и каждое встречается в пакете ровно один раз (иначе копии
// поля разъехались бы при пересчёте тегов на отправке).
func cpsTemplate(p packet) (string, error) {
	dyn := slices.Clone(p.dyn)
	slices.SortFunc(dyn, func(a, b dynField) int { return a.off - b.off })
	var b strings.Builder
	pos := 0
	for _, f := range dyn {
		if f.off < pos {
			return "", fmt.Errorf("dynamic fields overlap at offset %d", f.off)
		}
		if f.off+f.length > len(p.buf) {
			return "", fmt.Errorf("dynamic field %s %d at %d beyond packet of %d bytes", f.kind, f.length, f.off, len(p.buf))
		}
		if f.length < 1 || f.length > maxCPSLen {
			return "", fmt.Errorf("dynamic field length %d out of range 1..%d (SPEC 7.1 V-05)", f.length, maxCPSLen)
		}
		val := p.buf[f.off : f.off+f.length]
		if f.kind == tagChars && !isLetters(val) {
			return "", fmt.Errorf("<rc> field at %d is not letters", f.off)
		}
		if f.kind == tagDigits && !isDigits(val) {
			return "", fmt.Errorf("<rd> field at %d is not digits", f.off)
		}
		if n := bytesCount(p.buf, val); n != 1 {
			return "", fmt.Errorf("dynamic field at %d (%d bytes) occurs %d times in packet; want 1 (SPEC 7.6 п.2)", f.off, f.length, n)
		}
		writeB(&b, p.buf[pos:f.off])
		b.WriteString("<" + f.kind.String() + " " + strconv.Itoa(f.length) + ">")
		pos = f.off + f.length
	}
	writeB(&b, p.buf[pos:])
	if b.Len() == 0 {
		return "", fmt.Errorf("empty packet")
	}
	return b.String(), nil
}

// writeB пишет литеральный кусок как <b 0x…> (пустые куски пропускаются).
func writeB(b *strings.Builder, lit []byte) {
	if len(lit) == 0 {
		return
	}
	const hex = "0123456789abcdef"
	b.WriteString("<b 0x")
	for _, x := range lit {
		b.WriteByte(hex[x>>4])
		b.WriteByte(hex[x&0x0f])
	}
	b.WriteString(">")
}

func bytesCount(buf, val []byte) int {
	n := 0
	for i := 0; i+len(val) <= len(buf); i++ {
		if slices.Equal(buf[i:i+len(val)], val) {
			n++
		}
	}
	return n
}

func isLetters(b []byte) bool {
	for _, x := range b {
		if !(x >= 'a' && x <= 'z' || x >= 'A' && x <= 'Z') {
			return false
		}
	}
	return true
}

func isDigits(b []byte) bool {
	for _, x := range b {
		if x < '0' || x > '9' {
			return false
		}
	}
	return true
}

// u16/u32 — сериализация полей пакетов (big-endian).
func u16(b *packet, v uint16) {
	b.buf = append(b.buf, byte(v>>8), byte(v))
}

func u24(b *packet, v uint32) {
	b.buf = append(b.buf, byte(v>>16), byte(v>>8), byte(v))
}

func u32(b *packet, v uint32) {
	b.buf = append(b.buf, byte(v>>24), byte(v>>16), byte(v>>8), byte(v))
}

// bytes appendит n случайных байт и возвращает их срез в buf.
func (g *gen) bytes(b *packet, n int) []byte {
	off := len(b.buf)
	buf := make([]byte, n)
	if _, err := io.ReadFull(g.r, buf); err != nil {
		g.err = fmt.Errorf("read random: %w", err)
	}
	b.buf = append(b.buf, buf...)
	return b.buf[off:]
}

// letters appendит n случайных строчных букв (поле <rc N>: буквы по алфавиту тега).
func (g *gen) letters(b *packet, n int) {
	const abc = "abcdefghijklmnopqrstuvwxyz"
	for range n {
		b.buf = append(b.buf, abc[g.int(0, len(abc)-1)])
	}
}
