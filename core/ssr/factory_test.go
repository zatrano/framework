package ssr_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zatrano/framework/v3/core/ssr"
)

type stubEngine struct {
	dir string
}

func (s *stubEngine) Render(name string, data map[string]any) (string, error) {
	return "stub:" + name, nil
}
func (s *stubEngine) Component(name string, data map[string]any) (string, error) {
	return "stub-component:" + name, nil
}
func (s *stubEngine) Share(key string, value any)              {}
func (s *stubEngine) AddFunc(name string, fn any)              {}
func (s *stubEngine) EnableCache(enabled bool)                 {}
func (s *stubEngine) SetEnvironment(env string)                {}

func TestSetFactorySwapsEngine(t *testing.T) {
	defer ssr.SetFactory(nil)

	ssr.SetFactory(func(directory string) ssr.Engine {
		return &stubEngine{dir: directory}
	})
	eng := ssr.New(t.TempDir())
	out, err := eng.Render("x", nil)
	if err != nil || out != "stub:x" {
		t.Fatalf("got %q err=%v", out, err)
	}
}

func TestDefaultFactoryIsCanvas(t *testing.T) {
	ssr.SetFactory(nil)
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "hi.html"), []byte(`Hello`), 0o644)
	eng := ssr.New(dir)
	if _, ok := ssr.Canvas(eng); !ok {
		t.Fatal("default factory should return Canvas-backed engine")
	}
	out, err := eng.Render("hi", nil)
	if err != nil || out != "Hello" {
		t.Fatalf("got %q err=%v", out, err)
	}
}
