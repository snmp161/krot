package awg

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// maxCPSLen — максимальное N в тегах <r N>, <rc N>, <rd N> (SPEC 7.1 V-05).
const maxCPSLen = 1000

// ValidateCPS проверяет синтаксис CPS-строки I1–I5 (SPEC 7.1 V-05): последовательность тегов
// <b 0xHEX>, <t>, <r N>, <rc N>, <rd N>; N ≤ 1000; hex чётной длины. Между тегами допустимы
// только пробелы, внутри тега аргумент отделяется пробелами (как в amneziawg-go).
func ValidateCPS(s string) error {
	rest := s
	tags := 0
	for {
		rest = strings.TrimLeft(rest, " \t")
		if rest == "" {
			break
		}
		if rest[0] != '<' {
			return fmt.Errorf("unexpected text %q outside of tags", rest)
		}
		end := strings.IndexByte(rest, '>')
		if end < 0 {
			return errors.New("missing closing '>'")
		}
		if err := checkTag(rest[1:end]); err != nil {
			return err
		}
		tags++
		rest = rest[end+1:]
	}
	if tags == 0 {
		return errors.New("no tags")
	}
	return nil
}

func checkTag(tag string) error {
	parts := strings.Fields(tag)
	if len(parts) == 0 {
		return errors.New("empty tag <>")
	}
	name, args := parts[0], parts[1:]
	switch name {
	case "t":
		if len(args) != 0 {
			return errors.New("tag <t> takes no argument")
		}
		return nil
	case "b":
		if len(args) != 1 {
			return errors.New("tag <b> requires one argument 0x<hex>")
		}
		h, ok := strings.CutPrefix(args[0], "0x")
		if !ok || h == "" || len(h)%2 != 0 {
			return fmt.Errorf("tag <b %s>: argument must be 0x followed by an even number of hex digits", args[0])
		}
		for _, r := range h {
			if !strings.ContainsRune("0123456789abcdefABCDEF", r) {
				return fmt.Errorf("tag <b %s>: invalid hex digit %q", args[0], r)
			}
		}
		return nil
	case "r", "rc", "rd":
		if len(args) != 1 {
			return fmt.Errorf("tag <%s> requires one numeric argument", name)
		}
		for _, r := range args[0] {
			if r < '0' || r > '9' {
				return fmt.Errorf("tag <%s %s>: argument must be a number", name, args[0])
			}
		}
		n, err := strconv.Atoi(args[0])
		if err != nil || n > maxCPSLen {
			return fmt.Errorf("tag <%s %s>: length must be at most %d", name, args[0], maxCPSLen)
		}
		return nil
	}
	return fmt.Errorf("unknown tag <%s>", name)
}
