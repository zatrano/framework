package routing_test

import (
	"path/filepath"
	"testing"

	"github.com/zatrano/framework/http"
	"github.com/zatrano/framework/routing"
)

func TestRouteSnapshotAndCache(t *testing.T) {
	r := routing.New()
	r.Get("/ping", func(req *http.Request) *http.Response {
		return http.JSON(map[string]any{"ok": true})
	}).As("ping")
	snap := r.Snapshot()
	if len(snap) != 1 || snap[0].Name != "ping" {
		t.Fatalf("snap=%#v", snap)
	}
	path := filepath.Join(t.TempDir(), "routes.json")
	if err := r.SaveCache(path); err != nil {
		t.Fatal(err)
	}
	items, err := routing.LoadRouteCache(path)
	if err != nil || len(items) != 1 {
		t.Fatalf("items=%v err=%v", items, err)
	}
}
