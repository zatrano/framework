package ssr_test

import (
	"testing"

	"github.com/zatrano/canvas"
	"github.com/zatrano/framework/v3/core/kernel"
	"github.com/zatrano/framework/v3/core/ssr"
)

func TestFromReturnsConcreteCanvasEngine(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	eng := canvas.New(t.TempDir())
	app.Container().Instance(ssr.ContainerKey, eng)

	got := ssr.From(app)
	if got != eng {
		t.Fatalf("From must return the exact *canvas.Engine instance (no wrapper)")
	}
	if ssr.From(nil) != nil {
		t.Fatal("From(nil)")
	}
}
