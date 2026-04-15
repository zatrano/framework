package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// I18n configures application-level translations (JSON catalogs per locale).
type I18n struct {
	Enabled bool `mapstructure:"enabled"`
	// DefaultLocale is the fallback BCP47 base tag (e.g. en, tr).
	DefaultLocale string `mapstructure:"default_locale"`
	// SupportedLocales lists which locale files are loaded ({locales_dir}/{tag}.json). Must include DefaultLocale.
	SupportedLocales []string `mapstructure:"supported_locales"`
	// LocalesDir is a directory relative to the process working directory (or absolute).
	LocalesDir string `mapstructure:"locales_dir"`
	// CookieName reads optional persistent language choice (default zatrano_lang).
	CookieName string `mapstructure:"cookie_name"`
	// QueryKey is the query parameter for language override (default lang).
	QueryKey string `mapstructure:"query_key"`
}

func (c *Config) applyI18nDefaults() {
	if !c.I18n.Enabled {
		return
	}
	if strings.TrimSpace(c.I18n.DefaultLocale) == "" {
		c.I18n.DefaultLocale = "en"
	}
	c.I18n.DefaultLocale = strings.ToLower(strings.TrimSpace(c.I18n.DefaultLocale))
	if len(c.I18n.SupportedLocales) == 0 {
		c.I18n.SupportedLocales = []string{c.I18n.DefaultLocale}
	}
	norm := make([]string, 0, len(c.I18n.SupportedLocales))
	seen := make(map[string]bool)
	for _, s := range c.I18n.SupportedLocales {
		t := strings.ToLower(strings.TrimSpace(s))
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		norm = append(norm, t)
	}
	c.I18n.SupportedLocales = norm
	found := false
	for _, t := range c.I18n.SupportedLocales {
		if t == c.I18n.DefaultLocale {
			found = true
			break
		}
	}
	if !found {
		c.I18n.SupportedLocales = append([]string{c.I18n.DefaultLocale}, c.I18n.SupportedLocales...)
	}
	if strings.TrimSpace(c.I18n.LocalesDir) == "" {
		c.I18n.LocalesDir = "locales"
	}
	if strings.TrimSpace(c.I18n.CookieName) == "" {
		c.I18n.CookieName = "zatrano_lang"
	}
	if strings.TrimSpace(c.I18n.QueryKey) == "" {
		c.I18n.QueryKey = "lang"
	}
}

func (c *Config) validateI18n() error {
	if !c.I18n.Enabled {
		return nil
	}
	dir := filepath.Clean(strings.TrimSpace(c.I18n.LocalesDir))
	if dir == "" || dir == "." {
		return fmt.Errorf("i18n.locales_dir is required when i18n.enabled is true")
	}
	fi, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("i18n.locales_dir %q: %w", c.I18n.LocalesDir, err)
	}
	if !fi.IsDir() {
		return fmt.Errorf("i18n.locales_dir must be a directory: %s", c.I18n.LocalesDir)
	}
	return nil
}

