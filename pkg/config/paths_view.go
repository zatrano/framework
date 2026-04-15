package config

import (
	"os"
	"path/filepath"
	"strings"
)

// PathsView is a compact, share-safe summary for debugging which files and dirs apply.
func PathsView(cfg *Config, wd, configDir string, dotenvPresent bool) map[string]any {
	profile := filepath.Join(configDir, cfg.Env+".yaml")
	profileState := profile
	if st, err := os.Stat(profile); err != nil || st.IsDir() {
		profileState = "missing"
	}
	localesDir := ""
	if cfg.I18n.Enabled && strings.TrimSpace(cfg.I18n.LocalesDir) != "" {
		ld := filepath.Clean(cfg.I18n.LocalesDir)
		localesDir = ld
		if fi, err := os.Stat(ld); err != nil || !fi.IsDir() {
			localesDir = ld + " (missing)"
		}
	}
	dotenv := "absent"
	if dotenvPresent {
		dotenv = "present"
	}
	return map[string]any{
		"env":            cfg.Env,
		"working_dir":    wd,
		"dotenv":         dotenv,
		"config_dir":     configDir,
		"config_profile": profileState,
		"http_addr":      cfg.HTTPAddr,
		"openapi_path":   cfg.OpenAPIPath,
		"migrations_dir": cfg.MigrationsDir,
		"seeds_dir":      cfg.SeedsDir,
		"locales_dir":    localesDir,
	}
}

