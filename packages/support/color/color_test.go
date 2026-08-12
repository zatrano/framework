package color_test

import (
	"testing"

	"github.com/zatrano/framework/packages/support/color"
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
}
