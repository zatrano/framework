package config

import "testing"

func TestMaskConnectionURL(t *testing.T) {
	if got := MaskConnectionURL(""); got != "(empty)" {
		t.Fatalf("empty: %q", got)
	}
	in := "postgres://user:secret@localhost:5432/db"
	got := MaskConnectionURL(in)
	if got == in || got == "(set)" {
		t.Fatalf("expected masked url, got %q", got)
	}
	if got != "postgres://user:***@localhost:5432/db" {
		t.Fatalf("got %q", got)
	}
}

func TestMaskSecret(t *testing.T) {
	if MaskSecret("") != "(empty)" {
		t.Fatal()
	}
	if MaskSecret("x") != "(set)" {
		t.Fatal()
	}
}

