package errs

import (
	"errors"
	"fmt"
	"testing"
)

// SPEC 4.5: коды выхода.
func TestCodes(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{nil, 0},
		{errors.New("boom"), 1},
		{Usagef("bad flag %s", "--x"), 2},
		{Invalidf("bad value"), 3},
		{NotFoundf("no such interface"), 4},
		{Conflictf("exists"), 5},
		{Envf("not root"), 6},
		{Lockf("busy"), 7},
		{Cancelledf("cancelled"), 10},
		{fmt.Errorf("wrapped: %w", Conflictf("exists")), 5},
		{Wrap(Env, errors.New("inner")), 6},
	}
	for _, c := range cases {
		if got := Code(c.err); got != c.want {
			t.Errorf("Code(%v) = %d; want %d", c.err, got, c.want)
		}
	}
	if Usagef("bad flag %s", "--x").Error() != "bad flag --x" {
		t.Error("message formatting is wrong")
	}
	inner := errors.New("inner")
	if !errors.Is(Wrap(Env, inner), inner) {
		t.Error("Wrap must keep the cause")
	}
}
