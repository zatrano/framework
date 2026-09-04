package config_test

import (
	"path/filepath"
	"testing"

	"github.com/zatrano/framework/kernel/config"
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

func TestLoadIfAbsentSkipsExisting(t *testing.T) {
	repo := config.New()
	repo.Load("mongo", map[string]any{"uri": "cached"})
	config.LoadIfAbsent(repo, "mongo", map[string]any{"uri": "default"})
	if repo.GetString("mongo.uri") != "cached" {
		t.Fatalf("cache must win, got %q", repo.GetString("mongo.uri"))
	}
	config.LoadIfAbsent(repo, "oauth", map[string]any{"store_path": "x"})
	if repo.GetString("oauth.store_path") != "x" {
		t.Fatal("missing namespace should load")
	}
}

func TestRepositoryFreeze(t *testing.T) {
	repo := config.New()
	repo.Load("app", map[string]any{"name": "ZATRANO"})
	repo.Freeze()
	if !repo.Frozen() {
		t.Fatal("expected frozen")
	}
	if repo.GetString("app.name") != "ZATRANO" {
		t.Fatal("reads must still work")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("expected write panic")
		}
	}()
	repo.Set("app.name", "nope")
}

func TestAllDeepCopiesSlices(t *testing.T) {
	repo := config.New()
	repo.Load("app", map[string]any{
		"hosts": []any{"a", "b"},
		"tags":  []string{"one", "two"},
	})
	all := repo.All()
	hosts, ok := all["app"].(map[string]any)["hosts"].([]any)
	if !ok {
		t.Fatal("hosts")
	}
	hosts[0] = "mutated"
	got := repo.Get("app.hosts").([]any)
	if got[0] != "a" {
		t.Fatalf("All() must not alias nested slices, got %#v", got)
	}
	tags := all["app"].(map[string]any)["tags"].([]string)
	tags[0] = "mutated"
	raw := repo.Get("app.tags").([]string)
	if raw[0] != "one" {
		t.Fatalf("All() must copy []string, got %#v", raw)
	}
}
