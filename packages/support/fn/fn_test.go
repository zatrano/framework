package fn_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/zatrano/framework/packages/support/fn"
)

func TestDebounce(t *testing.T) {
	var n atomic.Int32
	d := fn.Debounce(30*time.Millisecond, func() { n.Add(1) })
	d()
	d()
	d()
	time.Sleep(80 * time.Millisecond)
	if n.Load() != 1 {
		t.Fatalf("n=%d", n.Load())
	}
}

func TestThrottle(t *testing.T) {
	var n atomic.Int32
	th := fn.Throttle(50*time.Millisecond, func() { n.Add(1) })
	th()
	th()
	if n.Load() != 1 {
		t.Fatalf("n=%d", n.Load())
	}
	time.Sleep(70 * time.Millisecond)
	th()
	if n.Load() != 2 {
		t.Fatalf("n=%d", n.Load())
	}
}

func TestOnce(t *testing.T) {
	var n atomic.Int32
	o := fn.Once(func() { n.Add(1) })
	o()
	o()
	if n.Load() != 1 {
		t.Fatal(n.Load())
	}
}
