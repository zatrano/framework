package support_test

import (
	"testing"

	"github.com/zatrano/framework/packages/support"
)

func TestWhenUnlessTap(t *testing.T) {
	if support.When(true, "a", "b") != "a" {
		t.Fatal("when true")
	}
	if support.When(false, "a", "b") != "b" {
		t.Fatal("when false")
	}
	if support.Unless(false, "x", "y") != "x" {
		t.Fatal("unless")
	}
	var seen int
	out := support.Tap(5, func(n int) { seen = n })
	if out != 5 || seen != 5 {
		t.Fatalf("tap out=%d seen=%d", out, seen)
	}
	if support.Transform(2, func(n int) int { return n * 3 }) != 6 {
		t.Fatal("transform")
	}
}
