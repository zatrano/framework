package ssr_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/canvas"
	"github.com/zatrano/framework/v3/core/kernel"
	"github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/framework/v3/core/ssr"
)

func TestRenderTemplateFailsLoudWithoutEngine(t *testing.T) {
	app := kernel.NewApplication(t.TempDir())
	resp := ssr.RenderTemplate(app, http.Template("web.welcome"))
	if resp == nil || resp.StatusCode() != 500 {
		t.Fatalf("status=%v", resp)
	}
	body := string(resp.Content())
	if !strings.Contains(body, "Canvas engine not bound") && !strings.Contains(body, "Template rendering failed") {
		t.Fatalf("body=%s", body)
	}
}

func TestRenderTemplateUsesCanvasEngine(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "hi.html"), []byte(`Hello {{ $name }}`), 0o644); err != nil {
		t.Fatal(err)
	}
	app := kernel.NewApplication(dir)
	eng := canvas.New(dir)
	app.Container().Instance(ssr.ContainerKey, eng)

	resp := ssr.RenderTemplate(app, http.Template("hi", map[string]any{"name": "Ada"}))
	if resp == nil || resp.StatusCode() != 200 {
		t.Fatalf("status=%v body=%s", resp.StatusCode(), resp.Content())
	}
	if !strings.Contains(string(resp.Content()), "Hello Ada") {
		t.Fatalf("body=%s", resp.Content())
	}
}
