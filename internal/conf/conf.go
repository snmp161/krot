// Package conf читает и изменяет конфиги wg-quick/awg-quick, сохраняя все чужие строки,
// комментарии и порядок байт-в-байт (SPEC 5.1, 5.3). Метаданные krot хранятся в комментариях.
package conf

import (
	"errors"
	"strconv"
	"strings"
)

var (
	ErrPeerNotFound  = errors.New("peer not found")
	ErrAlreadyLocked = errors.New("peer is already locked")
	ErrNotLocked     = errors.New("peer is not locked")
)

const (
	lockPrefix = "#~ "
	metaPrefix = "krot:"
)

// KV — пара ключ/значение (строка конфига или метаданные).
type KV struct {
	Key, Value string
}

// Config — конфиг как набор сырых строк. Изменения затрагивают только строки krot.
type Config struct {
	lines   []string // без "\n"; "\r" для CRLF-файлов остаётся в строке
	finalNL bool
	eol     string // "" или "\r" — окончание новых строк
}

// Parse разбирает конфиг. Разбор не бывает ошибочным: непонятные строки сохраняются как есть.
func Parse(b []byte) *Config {
	c := &Config{}
	s := string(b)
	if s == "" {
		return c
	}
	if strings.HasSuffix(s, "\n") {
		c.finalNL = true
		s = s[:len(s)-1]
	}
	c.lines = strings.Split(s, "\n")
	if strings.HasSuffix(c.lines[0], "\r") {
		c.eol = "\r"
	}
	return c
}

// Bytes сериализует конфиг.
func (c *Config) Bytes() []byte {
	s := strings.Join(c.lines, "\n")
	if c.finalNL {
		s += "\n"
	}
	return []byte(s)
}

type lineKind int

const (
	kBlank lineKind = iota
	kComment
	kLocked
	kHeader
	kKV
	kOther
)

type lineInfo struct {
	kind   lineKind
	header string // имя секции в нижнем регистре для kHeader
	key    string
	value  string
	inner  *lineInfo // разбор содержимого locked-строки
}

func stripComment(s string) string {
	if i := strings.IndexByte(s, '#'); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

func classify(raw string) lineInfo {
	t := strings.TrimSpace(strings.TrimSuffix(raw, "\r"))
	switch {
	case t == "":
		return lineInfo{kind: kBlank}
	case strings.HasPrefix(t, "#~"):
		inner := classify(strings.TrimPrefix(strings.TrimPrefix(t, "#~"), " "))
		return lineInfo{kind: kLocked, inner: &inner}
	case strings.HasPrefix(t, "#"):
		return lineInfo{kind: kComment}
	}
	s := stripComment(t)
	if strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]") {
		return lineInfo{kind: kHeader, header: strings.ToLower(strings.TrimSpace(s[1 : len(s)-1]))}
	}
	if k, v, ok := strings.Cut(s, "="); ok && strings.TrimSpace(k) != "" {
		return lineInfo{kind: kKV, key: strings.TrimSpace(k), value: strings.TrimSpace(v)}
	}
	return lineInfo{kind: kOther}
}

// parseMeta разбирает строку "# krot:key=value".
func parseMeta(raw string) (KV, bool) {
	t := strings.TrimSpace(strings.TrimSuffix(raw, "\r"))
	if !strings.HasPrefix(t, "#") || strings.HasPrefix(t, "#~") {
		return KV{}, false
	}
	t = strings.TrimSpace(t[1:])
	if !strings.HasPrefix(t, metaPrefix) {
		return KV{}, false
	}
	k, v, ok := strings.Cut(t[len(metaPrefix):], "=")
	if !ok || strings.TrimSpace(k) == "" {
		return KV{}, false
	}
	return KV{strings.TrimSpace(k), strings.TrimSpace(v)}, true
}

// section — найденная секция и её границы в строках (включительно).
type section struct {
	name      string // "interface", "peer", ...
	locked    bool
	metaStart int // первая строка метаданных (== header, если их нет)
	header    int
	end       int // последняя строка тела (== header, если тело пустое)
}

func (c *Config) infos() []lineInfo {
	out := make([]lineInfo, len(c.lines))
	for i, l := range c.lines {
		out[i] = classify(l)
	}
	return out
}

func isHeader(li lineInfo) (name string, locked, ok bool) {
	if li.kind == kHeader {
		return li.header, false, true
	}
	if li.kind == kLocked && li.inner.kind == kHeader && li.inner.header == "peer" {
		return "peer", true, true
	}
	return "", false, false
}

func (c *Config) sections() []section {
	infos := c.infos()
	var heads []int
	for i, li := range infos {
		if _, _, ok := isHeader(li); ok {
			heads = append(heads, i)
		}
	}
	var out []section
	for n, h := range heads {
		name, locked, _ := isHeader(infos[h])
		next := len(c.lines)
		if n+1 < len(heads) {
			next = heads[n+1]
		}
		s := section{name: name, locked: locked, metaStart: h, header: h, end: h}
		if locked {
			// Тело заблокированного peer'а — непрерывные строки "#~" после заголовка.
			for i := h + 1; i < next && infos[i].kind == kLocked; i++ {
				s.end = i
			}
		} else {
			// Тело активной секции — до последней строки key=value перед следующим заголовком.
			for i := h + 1; i < next; i++ {
				if infos[i].kind == kKV || infos[i].kind == kOther {
					s.end = i
				}
			}
		}
		// Метаданные peer'а — непрерывная серия "# krot:" прямо над заголовком.
		for i := h - 1; i >= 0; i-- {
			if _, ok := parseMeta(c.lines[i]); !ok {
				break
			}
			s.metaStart = i
		}
		out = append(out, s)
	}
	return out
}

// Interface — секция [Interface] и метаданные интерфейса.
type Interface struct {
	Meta map[string]string // все "# krot:k=v" до [Interface]
	Keys []KV              // строки key=value секции в порядке файла
}

// Get возвращает значение ключа (без учёта регистра; при повторах — последнее).
func (i Interface) Get(key string) string { return get(i.Keys, key) }

// List возвращает все значения ключа, разделённые по запятым (Address и т.п.).
func (i Interface) List(key string) []string { return list(i.Keys, key) }

func get(kvs []KV, key string) string {
	v := ""
	for _, kv := range kvs {
		if strings.EqualFold(kv.Key, key) {
			v = kv.Value
		}
	}
	return v
}

func list(kvs []KV, key string) []string {
	var out []string
	for _, kv := range kvs {
		if !strings.EqualFold(kv.Key, key) {
			continue
		}
		for _, item := range strings.Split(kv.Value, ",") {
			if item = strings.TrimSpace(item); item != "" {
				out = append(out, item)
			}
		}
	}
	return out
}

func (c *Config) bodyKVs(s section) []KV {
	var out []KV
	for i := s.header + 1; i <= s.end; i++ {
		li := classify(c.lines[i])
		if s.locked && li.kind == kLocked {
			li = *li.inner
		}
		if li.kind == kKV {
			out = append(out, KV{li.key, li.value})
		}
	}
	return out
}

// Interface возвращает секцию [Interface]; ok=false, если её нет.
func (c *Config) Interface() (Interface, bool) {
	for _, s := range c.sections() {
		if s.name != "interface" || s.locked {
			continue
		}
		ifc := Interface{Meta: map[string]string{}, Keys: c.bodyKVs(s)}
		for i := 0; i < s.header; i++ {
			if kv, ok := parseMeta(c.lines[i]); ok {
				ifc.Meta[kv.Key] = kv.Value
			}
		}
		return ifc, true
	}
	return Interface{Meta: map[string]string{}}, false
}

// Peer — peer конфига (активный или заблокированный).
type Peer struct {
	ID       string // krot:id; "" у чужих peer'ов
	Created  string // krot:created
	LockedAt string // krot:locked
	Locked   bool   // блок закомментирован префиксом "#~ "
	Meta     map[string]string
	Keys     []KV
}

func (p Peer) Get(key string) string    { return get(p.Keys, key) }
func (p Peer) List(key string) []string { return list(p.Keys, key) }
func (p Peer) PublicKey() string        { return p.Get("PublicKey") }
func (p Peer) PresharedKey() string     { return p.Get("PresharedKey") }
func (p Peer) AllowedIPs() []string     { return p.List("AllowedIPs") }
func (c *Config) peer(s section) Peer {
	p := Peer{Locked: s.locked, Meta: map[string]string{}, Keys: c.bodyKVs(s)}
	for i := s.metaStart; i < s.header; i++ {
		kv, _ := parseMeta(c.lines[i])
		p.Meta[kv.Key] = kv.Value
	}
	p.ID, p.Created, p.LockedAt = p.Meta["id"], p.Meta["created"], p.Meta["locked"]
	return p
}

// Peers возвращает все peer'ы в порядке файла.
func (c *Config) Peers() []Peer {
	var out []Peer
	for _, s := range c.sections() {
		if s.name == "peer" {
			out = append(out, c.peer(s))
		}
	}
	return out
}

func (c *Config) findPeer(id string) (section, Peer, bool) {
	if id == "" {
		return section{}, Peer{}, false
	}
	for _, s := range c.sections() {
		if s.name != "peer" {
			continue
		}
		if p := c.peer(s); p.ID == id {
			return s, p, true
		}
	}
	return section{}, Peer{}, false
}

// PeerByID ищет peer по krot:id (чужие peer'ы без krot:id не находятся).
func (c *Config) PeerByID(id string) (Peer, bool) {
	_, p, ok := c.findPeer(id)
	return p, ok
}

// NewPeer — данные нового peer'а krot.
type NewPeer struct {
	ID, Created  string
	PublicKey    string
	PresharedKey string // "" — без PSK
	AllowedIPs   []string
}

// AddPeer дописывает peer в конец файла через пустую строку (SPEC 5.1).
func (c *Config) AddPeer(p NewPeer) {
	if len(c.lines) > 0 {
		c.finalNL = true
	}
	block := []string{"", "# krot:id=" + p.ID, "# krot:created=" + p.Created, "[Peer]", "PublicKey = " + p.PublicKey}
	if p.PresharedKey != "" {
		block = append(block, "PresharedKey = "+p.PresharedKey)
	}
	block = append(block, "AllowedIPs = "+strings.Join(p.AllowedIPs, ", "))
	for _, l := range block {
		c.lines = append(c.lines, l+c.eol)
	}
	c.finalNL = true
}

// RemovePeer удаляет peer (метаданные, блок и одну предшествующую пустую строку).
func (c *Config) RemovePeer(id string) error {
	s, _, ok := c.findPeer(id)
	if !ok {
		return ErrPeerNotFound
	}
	from := s.metaStart
	if from > 0 && classify(c.lines[from-1]).kind == kBlank {
		from--
	}
	c.lines = append(c.lines[:from:from], c.lines[s.end+1:]...)
	if len(c.lines) == 0 {
		c.finalNL = false
	}
	return nil
}

// LockPeer комментирует блок peer'а префиксом "#~ " и добавляет "# krot:locked=<at>" (SPEC 5.1).
func (c *Config) LockPeer(id, at string) error {
	s, p, ok := c.findPeer(id)
	if !ok {
		return ErrPeerNotFound
	}
	if p.Locked {
		return ErrAlreadyLocked
	}
	for i := s.header; i <= s.end; i++ {
		c.lines[i] = lockPrefix + c.lines[i]
	}
	c.insert(s.header, "# krot:locked="+at+c.eol)
	return nil
}

// UnlockPeer снимает префикс "#~ " с блока и удаляет "# krot:locked".
func (c *Config) UnlockPeer(id string) error {
	s, p, ok := c.findPeer(id)
	if !ok {
		return ErrPeerNotFound
	}
	if !p.Locked {
		return ErrNotLocked
	}
	for i := s.header; i <= s.end; i++ {
		l := c.lines[i]
		if strings.HasPrefix(l, lockPrefix) {
			c.lines[i] = l[len(lockPrefix):]
		} else {
			c.lines[i] = strings.TrimPrefix(l, "#~")
		}
	}
	for i := s.header - 1; i >= s.metaStart; i-- {
		if kv, _ := parseMeta(c.lines[i]); kv.Key == "locked" {
			c.lines = append(c.lines[:i], c.lines[i+1:]...)
		}
	}
	return nil
}

func (c *Config) insert(at int, line string) {
	c.lines = append(c.lines, "")
	copy(c.lines[at+1:], c.lines[at:])
	c.lines[at] = line
}

// Server — данные конфига сервера для --create.
type Server struct {
	Meta       []KV // type, endpoint, dns
	PrivateKey string
	Addresses  []string
	ListenPort int
	MTU        int
	Params     []KV // параметры AmneziaWG в каноническом порядке
}

// RenderServer формирует конфиг, создаваемый --create (SPEC 5.1).
func RenderServer(s Server) []byte {
	var b strings.Builder
	for _, m := range s.Meta {
		b.WriteString("# krot:" + m.Key + "=" + m.Value + "\n")
	}
	b.WriteString("[Interface]\n")
	b.WriteString("PrivateKey = " + s.PrivateKey + "\n")
	b.WriteString("Address = " + strings.Join(s.Addresses, ", ") + "\n")
	b.WriteString("ListenPort = " + strconv.Itoa(s.ListenPort) + "\n")
	b.WriteString("MTU = " + strconv.Itoa(s.MTU) + "\n")
	for _, p := range s.Params {
		b.WriteString(p.Key + " = " + p.Value + "\n")
	}
	b.WriteString("# PreUp =\n# PostUp =\n# PreDown =\n# PostDown =\n")
	return []byte(b.String())
}
