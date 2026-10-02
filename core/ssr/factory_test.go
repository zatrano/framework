package ssr_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zatrano/canvas"
	"github.com/zatrano/framework/v3/core/ssr"
)

func TestSetFactorySwapsEngine(t *testing.T) {
	defer ssr.SetFactory(nil)

	ssr.SetFactory(func(directory string) *canvas.Engine {
		e := canvas.New(directory)
		e.Share("stub", true)
		return e
	})
	eng := ssr.New(t.TempDir())
	if eng == nil {
		t.Fatal("nil engine")
	}
}

func TestDefaultFactoryIsCanvas(t *testing.T) {
	ssr.SetFactory(nil)
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "hi.html"), []byte(`Hello`), 0o644)
	eng := ssr.New(dir)
	if eng == nil {
		t.Fatal("nil")
	}
	out, err := eng.Render("hi", nil)
	if err != nil || out != "Hello" {
		t.Fatalf("got %q err=%v", out, err)
	}
}
