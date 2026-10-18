// Package amnezia формирует ключ AmneziaVPN (vpn://) для клиентского конфига (SPEC 6.5).
package amnezia

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
)

// Key возвращает `vpn://` + base64url без `=` от qCompress(config): 4 байта длины
// (big-endian) и zlib-поток. Приложение импортирует такой ключ как конфиг WireGuard/AmneziaWG.
func Key(config string) string {
	var b bytes.Buffer
	b.Write(binary.BigEndian.AppendUint32(nil, uint32(len(config))))
	w, _ := zlib.NewWriterLevel(&b, 8) // уровень, с которым сжимает само приложение; ошибки нет
	w.Write([]byte(config))            // запись в bytes.Buffer не возвращает ошибок
	w.Close()
	return "vpn://" + base64.RawURLEncoding.EncodeToString(b.Bytes())
}
