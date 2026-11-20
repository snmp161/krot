package portcheck

import (
	"testing"
	"time"
)

// SPEC 4.4, 6.15 T36: диапазоны и RTT имеют стабильное текстовое представление.
func TestRangeAndRTTFormatting(t *testing.T) {
	single := PortRange{Start: 51820, End: 51820}
	if single.Size() != 1 || single.String() != "51820" {
		t.Fatalf("single range: size=%d string=%q", single.Size(), single.String())
	}
	rangeThree := PortRange{Start: 51820, End: 51822}
	if rangeThree.Size() != 3 || rangeThree.String() != "51820:51822" {
		t.Fatalf("range: size=%d string=%q", rangeThree.Size(), rangeThree.String())
	}
	if got := FormatRTT(Result{}); got != "-" {
		t.Fatalf("empty RTT = %q", got)
	}
	result := Result{RTTMin: 1500 * time.Microsecond, RTTAvg: 2 * time.Millisecond, RTTMax: 2499 * time.Microsecond}
	if got := FormatRTT(result); got != "2/2/2 ms" {
		t.Fatalf("RTT = %q", got)
	}
}

// SPEC 6.14 T36: echo payload отделяется только при точном префиксе.
func TestStripEchoPrefix(t *testing.T) {
	if got, ok := StripEchoPrefix([]byte("ECHO payload")); !ok || got != "payload" {
		t.Fatalf("StripEchoPrefix() = %q, %v", got, ok)
	}
	if got, ok := StripEchoPrefix([]byte("ECHO ")); !ok || got != "" {
		t.Fatalf("empty echo = %q, %v", got, ok)
	}
	if got, ok := StripEchoPrefix([]byte("ping SEQ:1")); ok || got != "" {
		t.Fatalf("non-echo = %q, %v", got, ok)
	}
}
