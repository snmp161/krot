// Package ipalloc выдаёт адреса клиентам из подсети интерфейса (SPEC 6.5).
package ipalloc

import (
	"errors"
	"net/netip"
	"sort"
)

var (
	ErrExhausted     = errors.New("no free addresses in the subnet")
	ErrOutsideSubnet = errors.New("address is outside of the interface subnets")
	ErrReserved      = errors.New("address is the network, broadcast or server address")
	ErrInUse         = errors.New("address is already used by another client")
)

// Last возвращает последний адрес префикса.
func Last(p netip.Prefix) netip.Addr {
	b := p.Masked().Addr().AsSlice()
	for i := p.Bits(); i < len(b)*8; i++ {
		b[i/8] |= 0x80 >> (i % 8)
	}
	a, _ := netip.AddrFromSlice(b)
	return a
}

// pointToPoint — подсети /31 (RFC 3021) и /127 (RFC 6164): адресов сети и broadcast нет,
// оба адреса допустимы (SPEC 6.5).
func pointToPoint(p netip.Prefix) bool {
	return p.IsValid() && p.Bits() == p.Addr().BitLen()-1
}

// IsNetworkOrBroadcast сообщает, является ли адрес префикса адресом сети (IPv4 и IPv6)
// или broadcast (IPv4). У /31 и /127 таких адресов нет.
func IsNetworkOrBroadcast(p netip.Prefix) bool {
	if pointToPoint(p) {
		return false
	}
	a := p.Addr()
	return a == p.Masked().Addr() || (a.Is4() && a == Last(p))
}

func single(a netip.Addr) netip.Prefix { return netip.PrefixFrom(a, a.BitLen()) }

// reserved — адреса, которые не выдаются никогда: сеть, broadcast (IPv4), адрес сервера.
// У /31 и /127 — только адрес сервера.
func reserved(server netip.Prefix) []netip.Prefix {
	if pointToPoint(server) {
		return []netip.Prefix{single(server.Addr())}
	}
	r := []netip.Prefix{single(server.Masked().Addr()), single(server.Addr())}
	if server.Addr().Is4() {
		r = append(r, single(Last(server)))
	}
	return r
}

// Allocate возвращает первый свободный адрес подсети сервера server (например, 10.8.0.1/24).
// used — занятые префиксы: адреса сервера и AllowedIPs всех peer'ов (включая заблокированных и чужих).
func Allocate(server netip.Prefix, used []netip.Prefix) (netip.Addr, error) {
	network := server.Masked()
	var busy []netip.Prefix
	for _, p := range append(reserved(server), used...) {
		if p.Addr().Is4() == network.Addr().Is4() && p.Overlaps(network) {
			busy = append(busy, p.Masked())
		}
	}
	sort.Slice(busy, func(i, j int) bool { return busy[i].Addr().Less(busy[j].Addr()) })

	// Проход по занятым префиксам, отсортированным по началу: кандидат сдвигается за конец
	// каждого префикса, который его покрывает. Через широкий префикс — прыжок, без перебора.
	cand := network.Addr()
	for _, p := range busy {
		if Last(p).Less(cand) {
			continue
		}
		if cand.Less(p.Addr()) {
			break
		}
		cand = Last(p).Next()
		if !cand.IsValid() {
			return netip.Addr{}, ErrExhausted
		}
	}
	if !network.Contains(cand) {
		return netip.Addr{}, ErrExhausted
	}
	return cand, nil
}

// Check проверяет адрес клиента, заданный вручную: он должен лежать в подсети одного из адресов
// сервера своего семейства, не быть зарезервированным и не пересекаться с занятыми.
func Check(addr netip.Addr, servers []netip.Prefix, used []netip.Prefix) error {
	for _, s := range servers {
		if s.Addr().Is4() != addr.Is4() || !s.Masked().Contains(addr) {
			continue
		}
		for _, r := range reserved(s) {
			if r.Contains(addr) {
				return ErrReserved
			}
		}
		for _, u := range used {
			if u.Contains(addr) {
				return ErrInUse
			}
		}
		return nil
	}
	return ErrOutsideSubnet
}
