package execx

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

// Call — записанный вызов.
type Call struct {
	Name string
	Args []string
}

// String — командная строка вызова.
func (c Call) String() string { return commandLine(c.Name, c.Args) }

// Response — ответ фейка на вызов.
type Response struct {
	Result Result
	Err    error
}

type rule struct {
	prefix string
	resp   Response
}

// Fake — исполнитель для тестов: записывает вызовы и отвечает по правилам.
// По умолчанию любая команда успешна с пустым выводом, а любая программа есть в PATH.
type Fake struct {
	// Handler вызывается первым; nil-ответ — перейти к правилам.
	Handler func(c Call) *Response
	// Missing — программы, которых «нет» в PATH.
	Missing map[string]bool

	mu    sync.Mutex
	Calls []Call
	rules []rule
}

// On задаёт успешный ответ для команд с префиксом prefix (по словам).
func (f *Fake) On(prefix, stdout string) *Fake {
	return f.add(prefix, Response{Result: Result{Stdout: []byte(stdout)}})
}

// Fail задаёт неуспешный ответ для команд с префиксом prefix.
func (f *Fake) Fail(prefix string, exitCode int, stderr string) *Fake {
	return f.add(prefix, Response{
		Result: Result{Stderr: []byte(stderr), ExitCode: exitCode},
		Err:    fmt.Errorf("%s: %s", prefix, strings.TrimSpace(stderr)),
	})
}

func (f *Fake) add(prefix string, r Response) *Fake {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rules = append(f.rules, rule{prefix, r})
	return f
}

// Run записывает вызов и возвращает ответ: Handler, затем последнее подходящее правило.
func (f *Fake) Run(name string, args ...string) (Result, error) {
	c := Call{Name: name, Args: append([]string(nil), args...)}
	f.mu.Lock()
	f.Calls = append(f.Calls, c)
	rules := f.rules
	h := f.Handler
	f.mu.Unlock()

	if h != nil {
		if r := h(c); r != nil {
			return r.Result, r.Err
		}
	}
	line := c.String()
	for i := len(rules) - 1; i >= 0; i-- {
		p := rules[i].prefix
		if line == p || strings.HasPrefix(line, p+" ") {
			return rules[i].resp.Result, rules[i].resp.Err
		}
	}
	return Result{}, nil
}

// LookPath возвращает "/usr/bin/<name>", если программа не указана в Missing.
func (f *Fake) LookPath(name string) (string, error) {
	if f.Missing[name] {
		return "", errors.New(name + ": executable file not found in $PATH")
	}
	return "/usr/bin/" + name, nil
}

// CommandLines возвращает записанные вызовы как командные строки.
func (f *Fake) CommandLines() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]string, len(f.Calls))
	for i, c := range f.Calls {
		out[i] = c.String()
	}
	return out
}
