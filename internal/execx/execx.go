// Package execx — исполнитель внешних команд (wg, awg, systemctl, ip, …), подменяемый в тестах (SPEC 8.2).
package execx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Result — результат выполнения команды.
type Result struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int // -1 — команду не удалось запустить
}

// Message — текст для сообщения об ошибке: stderr, а если он пуст — stdout.
func (r Result) Message() string {
	if m := strings.TrimSpace(string(r.Stderr)); m != "" {
		return m
	}
	return strings.TrimSpace(string(r.Stdout))
}

// Runner запускает внешние команды. Ошибка возвращается и при ненулевом коде выхода.
type Runner interface {
	Run(name string, args ...string) (Result, error)
	LookPath(name string) (string, error)
}

// Timeout — предельное время выполнения одной команды.
const Timeout = 2 * time.Minute

// OS — реальный исполнитель.
type OS struct{}

// Run выполняет команду с LC_ALL=C (вывод утилит разбирается).
func (OS) Run(name string, args ...string) (Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), "LC_ALL=C")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	res := Result{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}
	var ee *exec.ExitError
	switch {
	case err == nil:
		return res, nil
	case errors.As(err, &ee):
		res.ExitCode = ee.ExitCode()
		msg := res.Message()
		if msg == "" {
			msg = ee.String()
		}
		return res, fmt.Errorf("%s: %s", commandLine(name, args), msg)
	default:
		res.ExitCode = -1
		return res, fmt.Errorf("%s: %w", commandLine(name, args), err)
	}
}

// LookPath ищет программу в PATH.
func (OS) LookPath(name string) (string, error) { return exec.LookPath(name) }

func commandLine(name string, args []string) string {
	return strings.Join(append([]string{name}, args...), " ")
}
