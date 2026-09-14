package color_test

import (
	"testing"

	"github.com/zatrano/framework/v2/kernel/support/color"
)

func TestColorHelpers(t *testing.T) {
	c, err := color.ParseHex("#1E90FF")
	if err != nil || c.R != 0x1E || c.B != 0xFF {
		t.Fatalf("%+v err=%v", c, err)
	}
	if c.Hex() != "#1E90FF" {
		t.Fatal(c.Hex())
	}
	if color.IsDark(c) {
		t.Fatal("expected light-ish blue")
	}
	black, _ := color.ParseHex("#000")
	white, _ := color.ParseHex("#fff")
	if color.ContrastRatio(black, white) < 20 {
		t.Fatal("contrast")
	}
	mixed := color.Mix(black, white, 0.5)
	if mixed.R < 100 || mixed.R > 150 {
		t.Fatalf("%+v", mixed)
	}
	alpha, err := color.ParseHex("#11223344")
	if err != nil || alpha.A != 0x44 || alpha.Hex() != "#11223344" {
		t.Fatalf("%+v %v", alpha, err)
	}
	if _, err := color.ParseHex("zz"); err == nil {
		t.Fatal("invalid hex")
	}
	clamped := color.Mix(black, white, 2)
	if clamped.R != 255 {
		t.Fatalf("t>1 %+v", clamped)
	}
	low := color.Mix(black, white, -1)
	if low.R != 0 {
		t.Fatalf("t<0 %+v", low)
	}
}
