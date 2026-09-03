package config

import (
	"os"
	"testing"
)

func TestDatabaseConfigEmptyByDefault(t *testing.T) {
	t.Setenv("DB_CONNECTION", "")
	t.Setenv("DB_CONNECTIONS", "")
	_ = os.Unsetenv("DB_CONNECTION")
	_ = os.Unsetenv("DB_CONNECTIONS")

	cfg := Database()
	if got := cfg["default"]; got != "" {
		t.Fatalf("default=%v, want empty", got)
	}
	conns, _ := cfg["connections"].(map[string]any)
	if len(conns) != 0 {
		t.Fatalf("connections=%v, want none", conns)
	}
}

func TestDatabaseConfigSQLiteWhenSelected(t *testing.T) {
	t.Setenv("DB_CONNECTION", "sqlite")
	t.Setenv("DB_DATABASE", "database/app.sqlite")
	cfg := Database()
	if cfg["default"] != "sqlite" {
		t.Fatalf("default=%v", cfg["default"])
	}
	conns := cfg["connections"].(map[string]any)
	sqlite := conns["sqlite"].(map[string]any)
	if sqlite["driver"] != "sqlite" {
		t.Fatalf("driver=%v", sqlite["driver"])
	}
}
