package kernel

import (
	"strings"
	"testing"
	"time"
)

func TestServerTimeoutParse(t *testing.T) {
	const key = "ZATRANO_TEST_SERVER_TIMEOUT"
	fallback := 60 * time.Second
	cases := []struct {
		name string
		set  bool
		raw  string
		want time.Duration
		bad  bool
	}{
		{name: "unset", want: fallback},
		{name: "empty", set: true, raw: "", want: fallback},
		{name: "whitespace", set: true, raw: "   ", want: fallback},
		{name: "zero", set: true, raw: "0", want: -1},
		{name: "zero duration", set: true, raw: "0s", want: -1},
		{name: "negative int", set: true, raw: "-1", want: -1},
		{name: "negative duration", set: true, raw: "-1s", want: -1},
		{name: "unitless seconds", set: true, raw: "30", want: 30 * time.Second},
		{name: "go seconds", set: true, raw: "30s", want: 30 * time.Second},
		{name: "go minutes", set: true, raw: "1m", want: time.Minute},
		{name: "padded duration", set: true, raw: "  30s  ", want: 30 * time.Second},
		{name: "abc", set: true, raw: "abc", bad: true},
		{name: "1x", set: true, raw: "1x", bad: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.set {
				t.Setenv(key, tc.raw)
			}
			got, err := serverTimeout(key, fallback)
			if tc.bad {
				if err == nil || got != 0 {
					t.Fatalf("err=%v got=%v, want a boot error", err, got)
				}
				if !strings.Contains(err.Error(), key) || !strings.Contains(err.Error(), tc.raw) {
					t.Fatalf("error %q should name %s and %q", err, key, tc.raw)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Fatalf("got %v err=%v, want %v", got, err, tc.want)
			}
		})
	}
}
