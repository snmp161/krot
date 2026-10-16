// Package keys генерирует ключи WireGuard (Curve25519) средствами Go, без `wg genkey` (SPEC 2.3).
package keys

import (
	"crypto/ecdh"
	"encoding/base64"
	"fmt"
	"io"
)

// Key — 32-байтовый ключ WireGuard (приватный, публичный или PresharedKey).
type Key [32]byte

// String кодирует ключ в base64, как wg.
func (k Key) String() string { return base64.StdEncoding.EncodeToString(k[:]) }

// ParseKey разбирает ключ в base64.
func ParseKey(s string) (Key, error) {
	var k Key
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil || len(b) != len(k) {
		return k, fmt.Errorf("invalid key %q: must be 32 bytes in base64", s)
	}
	copy(k[:], b)
	return k, nil
}

func random(r io.Reader) (Key, error) {
	var k Key
	if _, err := io.ReadFull(r, k[:]); err != nil {
		return k, fmt.Errorf("read random: %w", err)
	}
	return k, nil
}

// NewPrivate создаёт приватный ключ из 32 случайных байт с clamp, как `wg genkey`.
func NewPrivate(r io.Reader) (Key, error) {
	k, err := random(r)
	if err != nil {
		return k, err
	}
	k[0] &= 248
	k[31] = (k[31] & 127) | 64
	return k, nil
}

// NewPreshared создаёт PresharedKey — 32 случайных байта.
func NewPreshared(r io.Reader) (Key, error) { return random(r) }

// Public вычисляет публичный ключ X25519.
func (k Key) Public() (Key, error) {
	var pub Key
	priv, err := ecdh.X25519().NewPrivateKey(k[:])
	if err != nil {
		return pub, err
	}
	copy(pub[:], priv.PublicKey().Bytes())
	return pub, nil
}
