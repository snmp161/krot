package portcheck

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

const (
	echoPrefix = "ECHO "
)

// PingRequest возвращает датаграмму ping SEQ:N.
func PingRequest(seq int) []byte {
	return []byte(fmt.Sprintf("ping SEQ:%d", seq))
}

// PongReply возвращает датаграмму pong SEQ:N.
func PongReply(seq int) []byte {
	return []byte(fmt.Sprintf("pong SEQ:%d", seq))
}

// ParsePong разбирает ответ pong SEQ:N.
func ParsePong(data []byte) (int, bool) {
	return parseSeq("pong", data)
}

func parseSeq(prefix string, data []byte) (int, bool) {
	want := prefix + " SEQ:"
	if !bytes.HasPrefix(data, []byte(want)) {
		return 0, false
	}
	rest := string(data[len(want):])
	if rest == "" {
		return 0, false
	}
	seq, err := strconv.Atoi(rest)
	if err != nil || seq < 1 {
		return 0, false
	}
	return seq, true
}

// EchoRequest возвращает датаграмму ECHO <text>.
func EchoRequest(text string) []byte {
	return []byte(echoPrefix + text)
}

// IsEchoReply проверяет, что ответ совпадает с отправленным текстом.
func IsEchoReply(data []byte, sent string) bool {
	return string(data) == sent
}

// TimeRequest возвращает датаграмму TIME.
func TimeRequest() []byte {
	return []byte("TIME")
}

// TimeReply возвращает датаграмму с unix time в миллисекундах.
func TimeReply(ms int64) []byte {
	return []byte(strconv.FormatInt(ms, 10))
}

// ParseTimeReply разбирает ответ TIME.
func ParseTimeReply(data []byte) (int64, bool) {
	ms, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		return 0, false
	}
	return ms, true
}

// StripEchoPrefix удаляет префикс "ECHO " из данных.
func StripEchoPrefix(data []byte) (string, bool) {
	s := string(data)
	if !strings.HasPrefix(s, echoPrefix) {
		return "", false
	}
	return s[len(echoPrefix):], true
}
