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

func TestAllAndGetDeepCopyNestedMapsInSlices(t *testing.T) {
	repo := config.New()
	repo.Load("db", map[string]any{
		"connections": []map[string]any{
			{"name": "mysql", "opts": map[string]any{"host": "127.0.0.1"}},
		},
	})
	all := repo.All()
	conns := all["db"].(map[string]any)["connections"].([]map[string]any)
	conns[0]["name"] = "hacked"
	conns[0]["opts"].(map[string]any)["host"] = "evil"
	got := repo.Get("db.connections").([]map[string]any)
	if got[0]["name"] != "mysql" {
		t.Fatalf("nested map in slice aliased via All(), got %#v", got[0])
	}
	if got[0]["opts"].(map[string]any)["host"] != "127.0.0.1" {
		t.Fatalf("nested map inside slice aliased, got %#v", got[0]["opts"])
	}
	got[0]["name"] = "via-get"
	again := repo.Get("db.connections").([]map[string]any)
	if again[0]["name"] != "mysql" {
		t.Fatal("Get() must return a copy")
	}
}

func TestLoadAndSetCopyIncomingValues(t *testing.T) {
	src := map[string]any{"hosts": []string{"a"}}
	repo := config.New()
	repo.Load("app", src)
	src["hosts"].([]string)[0] = "mutated"
	if repo.Get("app.hosts").([]string)[0] != "a" {
		t.Fatal("Load must copy the incoming map")
	}
	nested := []map[string]any{{"n": "keep"}}
	repo.Set("app.list", nested)
	nested[0]["n"] = "mutated"
	if repo.Get("app.list").([]map[string]any)[0]["n"] != "keep" {
		t.Fatal("Set must copy the incoming value")
	}
}
