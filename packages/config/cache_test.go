package config_test

import (
	"path/filepath"
	"testing"

	"github.com/zatrano/framework/packages/config"
)

func TestConfigCacheRoundTrip(t *testing.T) {
	repo := config.New()
	repo.Load("app", map[string]any{"name": "ZATRANO", "debug": true})
	path := filepath.Join(t.TempDir(), "config.json")
	if err := config.SaveCache(path, repo); err != nil {
		t.Fatal(err)
	}
	data, err := config.LoadCache(path)
	if err != nil {
		t.Fatal(err)
	}
	fresh := config.New()
	fresh.MergeCached(data)
	if fresh.GetString("app.name") != "ZATRANO" {
		t.Fatalf("got %q", fresh.GetString("app.name"))
	}
	if err := config.ClearCache(path); err != nil {
		t.Fatal(err)
	}
}
