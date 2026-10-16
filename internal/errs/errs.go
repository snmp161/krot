// Package errs описывает ошибки с кодами выхода krot (SPEC 4.5).
package errs

import (
	"errors"
	"fmt"
)

// Коды выхода (SPEC 4.5).
const (
	OK        = 0
	Internal  = 1
	Usage     = 2
	Invalid   = 3
	NotFound  = 4
	Conflict  = 5
	Env       = 6
	Lock      = 7
	Cancelled = 10
)

// Error — ошибка с кодом выхода.
type Error struct {
	Code int
	Msg  string
	Err  error // причина, если есть
}

func (e *Error) Error() string { return e.Msg }
func (e *Error) Unwrap() error { return e.Err }

// New создаёт ошибку с кодом и форматированным текстом.
func New(code int, format string, a ...any) error {
	return &Error{Code: code, Msg: fmt.Sprintf(format, a...)}
}

// Wrap присваивает существующей ошибке код выхода.
func Wrap(code int, err error) error {
	return &Error{Code: code, Msg: err.Error(), Err: err}
}

func Usagef(format string, a ...any) error     { return New(Usage, format, a...) }
func Invalidf(format string, a ...any) error   { return New(Invalid, format, a...) }
func NotFoundf(format string, a ...any) error  { return New(NotFound, format, a...) }
func Conflictf(format string, a ...any) error  { return New(Conflict, format, a...) }
func Envf(format string, a ...any) error       { return New(Env, format, a...) }
func Lockf(format string, a ...any) error      { return New(Lock, format, a...) }
func Cancelledf(format string, a ...any) error { return New(Cancelled, format, a...) }

// Code возвращает код выхода для ошибки: 0 для nil, 1 для ошибок без кода.
func Code(err error) int {
	if err == nil {
		return OK
	}
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return Internal
}
