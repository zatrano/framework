package config

import "testing"

func TestPathsView(t *testing.T) {
	cfg := &Config{
		Env:           "staging",
		HTTPAddr:      ":3000",
		OpenAPIPath:   "api/spec.yaml",
		MigrationsDir: "migrations",
		SeedsDir:      "db/seeds",
	}
	v := PathsView(cfg, "/app", "config", true)
	if v["env"] != "staging" || v["dotenv"] != "present" {
		t.Fatalf("%#v", v)
	}
	if v["config_profile"] != "missing" {
		t.Fatalf("expected missing profile for fake path, got %v", v["config_profile"])
	}
}

