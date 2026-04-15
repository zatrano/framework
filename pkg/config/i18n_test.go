package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateI18nRequiresExistingDir(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		LogLevel: "info",
		I18n: I18n{
			Enabled:          true,
			DefaultLocale:    "en",
			SupportedLocales: []string{"en"},
			LocalesDir:       filepath.Join(dir, "nope"),
		},
	}
	cfg.applyDerivedDefaults()
	if err := cfg.validate(); err == nil {
		t.Fatal("expected error for missing locales dir")
	}
}

func TestValidateI18nOK(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "locales"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{
		LogLevel: "info",
		I18n: I18n{
			Enabled:          true,
			DefaultLocale:    "en",
			SupportedLocales: []string{"en", "tr"},
			LocalesDir:       filepath.Join(dir, "locales"),
		},
	}
	cfg.applyDerivedDefaults()
	if err := cfg.validate(); err != nil {
		t.Fatal(err)
	}
}

