// Package cli разбирает аргументы командной строки krot (SPEC 4).
package cli

import (
	"net/netip"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/sovchinnikov/krot/internal/awg"
	"github.com/sovchinnikov/krot/internal/errs"
)

// Action — действие (SPEC 4.2).
type Action string

const (
	ActList         Action = "list"
	ActCreate       Action = "create"
	ActDelete       Action = "delete"
	ActUp           Action = "up"
	ActDown         Action = "down"
	ActAddClient    Action = "add-client"
	ActDeleteClient Action = "delete-client"
	ActLockClient   Action = "lock-client"
	ActUnlockClient Action = "unlock-client"
	ActListClient   Action = "list-client"
)

var allActions = []Action{
	ActList, ActCreate, ActDelete, ActUp, ActDown,
	ActAddClient, ActDeleteClient, ActLockClient, ActUnlockClient, ActListClient,
}

type need int

const (
	forbidden need = iota
	optional
	required
)

// Короткие имена для таблицы ниже.
const (
	opt = optional
	req = required
)

type param struct {
	name   string
	isBool bool
	matrix map[Action]need
}

// Матрица SPEC 4.3 — единственное место, где она описана.
// Порядок параметров задаёт порядок проверок и сообщений.
var paramTable = []param{
	{"type", false, map[Action]need{ActList: opt, ActCreate: req}},
	{"interface", false, map[Action]need{ActList: opt, ActCreate: opt, ActDelete: req, ActUp: req, ActDown: req,
		ActAddClient: req, ActDeleteClient: req, ActLockClient: req, ActUnlockClient: req, ActListClient: req}},
	{"ip", false, map[Action]need{ActList: opt, ActCreate: req, ActAddClient: opt}},
	{"port", false, map[Action]need{ActList: opt, ActCreate: req}},
	{"endpoint", false, map[Action]need{ActCreate: req, ActAddClient: opt}},
	{"mtu", false, map[Action]need{ActCreate: opt}},
	{"dns", false, map[Action]need{ActCreate: opt}},
	{"i1", false, map[Action]need{ActCreate: opt}},
	{"id", false, map[Action]need{ActAddClient: req, ActDeleteClient: req, ActLockClient: req, ActUnlockClient: req, ActListClient: opt}},
	{"allowed-ips", false, map[Action]need{ActAddClient: req}},
	{"psk", true, map[Action]need{ActAddClient: opt}},
	{"qr", true, map[Action]need{ActAddClient: opt}},
	{"json", true, map[Action]need{ActList: opt, ActListClient: opt}},
	{"yes", true, map[Action]need{ActDelete: opt}},
}

// Command — результат разбора.
type Command struct {
	Action  Action
	Help    bool
	Version bool

	Type       awg.Type
	Interface  string
	IPs        []netip.Prefix // create/add-client: адреса; list: фильтр (голый IP — префикс полной длины)
	Port       int
	Endpoint   string
	MTU        int // 0 — по умолчанию для типа
	DNS        []netip.Addr
	I1         string
	ID         string
	AllowedIPs []netip.Prefix
	PSK        bool
	QR         bool
	JSON       bool
	Yes        bool

	set map[string]bool
}

// Has сообщает, был ли параметр задан в командной строке.
func (c *Command) Has(name string) bool { return c.set[name] }

var (
	interfaceRe = regexp.MustCompile(`^[a-zA-Z0-9_.=+-]{1,15}$`)
	idRe        = regexp.MustCompile(`^[a-zA-Z0-9_.-]{1,64}$`)
	digitsRe    = regexp.MustCompile(`^[0-9]{1,6}$`)
	labelRe     = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?$`)
)

// ValidInterfaceName проверяет имя интерфейса (SPEC 4.4; "." и ".." Linux не допускает).
func ValidInterfaceName(s string) bool {
	return interfaceRe.MatchString(s) && s != "." && s != ".."
}

// ValidID проверяет идентификатор клиента (SPEC 4.4).
func ValidID(s string) bool { return idRe.MatchString(s) }

// Parse разбирает аргументы (без имени программы).
// Ошибки использования — код 2, ошибки значений — код 3 (SPEC 4.1, 4.5).
func Parse(args []string) (*Command, error) {
	if slices.Contains(args, "--help") {
		return &Command{Help: true}, nil
	}

	values := map[string]string{}
	set := map[string]bool{}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "--") || len(a) == 2 {
			return nil, errs.Usagef("unexpected argument %q (flags must be in the form --flag or --flag=value)", a)
		}
		name, value, hasValue := strings.Cut(a[2:], "=")
		isBool, ok := lookupFlag(name)
		if !ok {
			return nil, errs.Usagef("unknown flag --%s", name)
		}
		if set[name] {
			return nil, errs.Usagef("flag --%s given more than once", name)
		}
		set[name] = true
		switch {
		case isBool && hasValue:
			return nil, errs.Usagef("flag --%s does not take a value", name)
		case !isBool && !hasValue:
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") {
				return nil, errs.Usagef("flag --%s requires a value", name)
			}
			i++
			value = args[i]
		}
		values[name] = value
	}

	if set["version"] {
		if len(set) > 1 {
			return nil, errs.Usagef("--version cannot be combined with other flags")
		}
		return &Command{Version: true}, nil
	}

	var given []Action
	for _, a := range allActions {
		if set[string(a)] {
			given = append(given, a)
		}
	}
	switch len(given) {
	case 0:
		return nil, errs.Usagef("no action given; use one of %s (see --help)", actionList())
	case 1:
	default:
		return nil, errs.Usagef("only one action per run is allowed, got --%s and --%s", given[0], given[1])
	}
	act := given[0]

	for _, p := range paramTable {
		if set[p.name] && p.matrix[act] == forbidden {
			return nil, errs.Usagef("flag --%s is not applicable to --%s", p.name, act)
		}
	}
	for _, p := range paramTable {
		if !set[p.name] && p.matrix[act] == required {
			return nil, errs.Usagef("--%s requires --%s", act, p.name)
		}
	}
	// SPEC 4.3: --i1 допустим только для awg*.
	if set["i1"] {
		if t, ok := awg.ParseType(values["type"]); ok && !t.IsAmnezia() {
			return nil, errs.Usagef("flag --i1 is only applicable to --type awg, awg2 or awg3")
		}
	}

	c := &Command{Action: act, set: set}
	if err := c.fill(values); err != nil {
		return nil, err
	}
	return c, nil
}

func lookupFlag(name string) (isBool, ok bool) {
	switch name {
	case "help", "version":
		return true, true
	}
	for _, a := range allActions {
		if string(a) == name {
			return true, true
		}
	}
	for _, p := range paramTable {
		if p.name == name {
			return p.isBool, true
		}
	}
	return false, false
}

func actionList() string {
	var s []string
	for _, a := range allActions {
		s = append(s, "--"+string(a))
	}
	return strings.Join(s, ", ")
}

// fill проверяет и переводит значения в типы (SPEC 4.4); ошибки — код 3.
func (c *Command) fill(v map[string]string) error {
	var err error
	if c.Has("type") {
		var ok bool
		if c.Type, ok = awg.ParseType(v["type"]); !ok {
			return errs.Invalidf("invalid --type %q: must be wg, awg, awg2 or awg3", v["type"])
		}
	}
	if c.Has("interface") {
		if !ValidInterfaceName(v["interface"]) {
			return errs.Invalidf("invalid --interface %q: must match ^[a-zA-Z0-9_.=+-]{1,15}$", v["interface"])
		}
		c.Interface = v["interface"]
	}
	if c.Has("ip") {
		switch c.Action {
		case ActCreate:
			c.IPs, err = parseServerIPs(v["ip"])
		case ActAddClient:
			c.IPs, err = parseClientIPs(v["ip"])
		default:
			c.IPs, err = parseIPFilter(v["ip"])
		}
		if err != nil {
			return err
		}
	}
	if c.Has("port") {
		if c.Port, err = parseInt(v["port"], 1, 65535); err != nil {
			return errs.Invalidf("invalid --port %q: must be an integer 1..65535", v["port"])
		}
	}
	if c.Has("endpoint") {
		if !validEndpoint(v["endpoint"]) {
			return errs.Invalidf("invalid --endpoint %q: must be an IPv4 address, IPv6 address or hostname (without port)", v["endpoint"])
		}
		c.Endpoint = v["endpoint"]
	}
	if c.Has("mtu") {
		if c.MTU, err = parseInt(v["mtu"], 576, 9000); err != nil {
			return errs.Invalidf("invalid --mtu %q: must be an integer 576..9000", v["mtu"])
		}
	}
	if c.Has("dns") {
		for _, s := range splitList(v["dns"]) {
			a, perr := netip.ParseAddr(s)
			if perr != nil || a.Zone() != "" {
				return errs.Invalidf("invalid --dns %q: must be a comma-separated list of IP addresses", v["dns"])
			}
			c.DNS = append(c.DNS, a)
		}
	}
	if c.Has("i1") {
		if err := validateI1(v["i1"]); err != nil {
			return err
		}
		c.I1 = v["i1"]
	}
	if c.Has("id") {
		if !ValidID(v["id"]) {
			return errs.Invalidf("invalid --id %q: must match ^[a-zA-Z0-9_.-]{1,64}$", v["id"])
		}
		c.ID = v["id"]
	}
	if c.Has("allowed-ips") {
		for _, s := range splitList(v["allowed-ips"]) {
			p, perr := netip.ParsePrefix(s)
			if perr != nil {
				return errs.Invalidf("invalid --allowed-ips %q: must be a comma-separated list of CIDR prefixes", v["allowed-ips"])
			}
			c.AllowedIPs = append(c.AllowedIPs, p)
		}
	}
	c.PSK, c.QR, c.JSON, c.Yes = c.Has("psk"), c.Has("qr"), c.Has("json"), c.Has("yes")
	return nil
}

func validateI1(s string) error {
	if s == "" {
		return errs.Invalidf("invalid --i1: must not be empty")
	}
	return nil
}

func parseInt(s string, lo, hi int) (int, error) {
	if !digitsRe.MatchString(s) {
		return 0, strconv.ErrSyntax
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < lo || n > hi {
		return 0, strconv.ErrRange
	}
	return n, nil
}

// splitList делит список через запятую; пустой элемент возвращается как "" и не пройдёт разбор.
func splitList(s string) []string {
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func parsePrefixes(flag, s string) ([]netip.Prefix, error) {
	var out []netip.Prefix
	for _, item := range splitList(s) {
		p, err := netip.ParsePrefix(item)
		if err != nil || p.Addr().Is4In6() {
			return nil, errs.Invalidf("invalid %s %q: must be a comma-separated list of CIDR addresses", flag, s)
		}
		out = append(out, p)
	}
	return out, nil
}

func checkOnePerFamily(flag string, ps []netip.Prefix) error {
	var v4, v6 int
	for _, p := range ps {
		if p.Addr().Is4() {
			v4++
		} else {
			v6++
		}
	}
	if v4 > 1 || v6 > 1 {
		return errs.Invalidf("invalid %s: at most one IPv4 and one IPv6 address are allowed", flag)
	}
	return nil
}

// IsNetworkOrBroadcast сообщает, является ли адрес префикса адресом сети (IPv4 и IPv6)
// или broadcast (IPv4) — такие адреса не выдаются и не назначаются (SPEC 6.5).
func IsNetworkOrBroadcast(p netip.Prefix) bool {
	a := p.Addr()
	if a == p.Masked().Addr() {
		return true
	}
	return a.Is4() && a == lastAddr(p)
}

func lastAddr(p netip.Prefix) netip.Addr {
	b := p.Masked().Addr().AsSlice()
	for i := p.Bits(); i < len(b)*8; i++ {
		b[i/8] |= 0x80 >> (i % 8)
	}
	a, _ := netip.AddrFromSlice(b)
	return a
}

// create: адреса сервера, не более одного на семейство, не адрес сети/broadcast.
func parseServerIPs(s string) ([]netip.Prefix, error) {
	ps, err := parsePrefixes("--ip", s)
	if err != nil {
		return nil, err
	}
	if err := checkOnePerFamily("--ip", ps); err != nil {
		return nil, err
	}
	for _, p := range ps {
		if IsNetworkOrBroadcast(p) {
			return nil, errs.Invalidf("invalid --ip %s: network or broadcast address cannot be used for the server", p)
		}
	}
	return ps, nil
}

// add-client: только /32 и /128, не более одного на семейство.
func parseClientIPs(s string) ([]netip.Prefix, error) {
	ps, err := parsePrefixes("--ip", s)
	if err != nil {
		return nil, err
	}
	for _, p := range ps {
		if !p.IsSingleIP() {
			return nil, errs.Invalidf("invalid --ip %s: client addresses must be /32 (IPv4) or /128 (IPv6)", p)
		}
	}
	return ps, checkOnePerFamily("--ip", ps)
}

// list: IP или CIDR через запятую.
func parseIPFilter(s string) ([]netip.Prefix, error) {
	var out []netip.Prefix
	for _, item := range splitList(s) {
		if a, err := netip.ParseAddr(item); err == nil && a.Zone() == "" {
			out = append(out, netip.PrefixFrom(a, a.BitLen()))
			continue
		}
		p, err := netip.ParsePrefix(item)
		if err != nil {
			return nil, errs.Invalidf("invalid --ip %q: must be a comma-separated list of IP addresses or CIDR prefixes", s)
		}
		out = append(out, p)
	}
	return out, nil
}

// validEndpoint: IPv4, IPv6 или hostname по RFC 1123 (без порта).
func validEndpoint(s string) bool {
	if a, err := netip.ParseAddr(s); err == nil {
		return a.Zone() == ""
	}
	if len(s) == 0 || len(s) > 253 {
		return false
	}
	labels := strings.Split(s, ".")
	for _, l := range labels {
		if !labelRe.MatchString(l) {
			return false
		}
	}
	// Последняя метка не может быть числовой: иначе это испорченный IPv4 (1.2.3.256).
	_, err := strconv.Atoi(labels[len(labels)-1])
	return err != nil
}
