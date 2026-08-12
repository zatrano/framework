package once_test

import (
	"testing"

	"github.com/zatrano/framework/packages/support/once"
)

func TestValueAndMemo(t *testing.T) {
	calls := 0
	get := once.Value(func() int {
		calls++
		return 42
	})
	a, b := get(), get()
	if a != 42 || b != 42 || calls != 1 {
		t.Fatalf("calls=%d", calls)
	}

	memoCalls := 0
	double := once.Memo(func(n int) int {
		memoCalls++
		return n * 2
	})
	x, y := double(3), double(3)
	if x != 6 || y != 6 || memoCalls != 1 {
		t.Fatalf("memoCalls=%d", memoCalls)
	}
	if double(4) != 8 || memoCalls != 2 {
		t.Fatalf("memoCalls=%d", memoCalls)
	}
}

func TestDo(t *testing.T) {
	once.Reset("phase24")
	n := 0
	once.Do("phase24", func() { n++ })
	once.Do("phase24", func() { n++ })
	if n != 1 {
		t.Fatalf("n=%d", n)
	}
}
