package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/zatrano/framework/pkg/config"
	"github.com/zatrano/framework/pkg/core"
	"github.com/zatrano/framework/pkg/health"
	"github.com/zatrano/framework/pkg/meta"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check environment, config, and optional database/redis connectivity",
	RunE:  runDoctor,
}

func init() {
	doctorCmd.Flags().String("env", "", "environment name; default ZATRANO_ENV or dev")
	doctorCmd.Flags().String("config-dir", "config", "directory containing {env}.yaml")
	doctorCmd.Flags().Bool("no-dotenv", false, "do not load .env from the working directory")
	rootCmd.AddCommand(doctorCmd)
}

func runDoctor(cmd *cobra.Command, _ []string) error {
	envFlag, _ := cmd.Flags().GetString("env")
	configDir, _ := cmd.Flags().GetString("config-dir")
	noDotenv, _ := cmd.Flags().GetBool("no-dotenv")

	fmt.Printf("ZATRANO doctor\n")
	fmt.Printf("  cli version:     %s\n", meta.Version)
	fmt.Printf("  go version:      %s\n", runtime.Version())
	fmt.Printf("  os/arch:         %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("  working dir:     %s\n", mustWd())
	fmt.Println()

	if _, err := os.Stat(".env"); err != nil {
		fmt.Printf("  .env:            not found (optional; see .env.example)\n")
	} else {
		fmt.Printf("  .env:            present\n")
	}

	cfg, err := config.Load(config.LoadOptions{
		Env:       envFlag,
		ConfigDir: configDir,
		DotEnv:    !noDotenv,
	})
	if err != nil {
		return fmt.Errorf("config load failed: %w", err)
	}

	fmt.Printf("  environment:     %s\n", cfg.Env)
	fmt.Printf("  config dir:      %s\n", configDir)
	if fi, err := os.Stat(configDir); err != nil || !fi.IsDir() {
		fmt.Printf("  config profile:  directory missing or unreadable\n")
	} else {
		path := filepath.Join(configDir, cfg.Env+".yaml")
		if _, err := os.Stat(path); err != nil {
			fmt.Printf("  config profile:  %s (missing — defaults + env still apply)\n", path)
		} else {
			fmt.Printf("  config profile:  %s\n", path)
		}
	}

	fmt.Printf("  http addr:       %s\n", cfg.HTTPAddr)
	printHTTPSummary(cfg)
	printI18nSummary(cfg)
	fmt.Printf("  database url:    %s\n", config.MaskConnectionURL(cfg.DatabaseURL))
	fmt.Printf("  redis url:       %s\n", config.MaskConnectionURL(cfg.RedisURL))
	printOAuthSummary(cfg)
	fmt.Println()
	printPostgresClientTools()
	fmt.Println()

	app, err := core.Bootstrap(cfg)
	if err != nil {
		return fmt.Errorf("bootstrap: %w", err)
	}
	defer app.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	st := health.Probe(ctx, app)
	fmt.Printf("connectivity\n")
	fmt.Printf("  ready (probe):   %v\n", st.Ready)
	raw, err := json.MarshalIndent(st.Checks, "  ", "  ")
	if err != nil {
		return fmt.Errorf("encode checks: %w", err)
	}
	fmt.Printf("  checks:\n%s\n", raw)

	if !st.Ready {
		return fmt.Errorf("doctor: one or more checks failed — fix URLs/credentials or relax database_required/redis_required in dev")
	}
	fmt.Println()
	fmt.Println("All checks passed.")
	return nil
}

func mustWd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

// printHTTPSummary prints CORS, rate limit, timeout, and body size from config (same source as serve).
func printHTTPSummary(cfg *config.Config) {
	h := cfg.HTTP
	fmt.Printf("  http middleware\n")
	if !h.CORSEnabled {
		fmt.Printf("    cors:            off\n")
	} else {
		fmt.Printf("    cors:            on  origins=%s  credentials=%v  max_age=%ds\n",
			originsShort(h.CORSAllowOrigins), h.CORSAllowCredentials, h.CORSMaxAge)
	}
	if !h.RateLimitEnabled {
		fmt.Printf("    rate_limit:      off\n")
	} else {
		backend := "memory (per process)"
		if h.RateLimitRedis {
			backend = "redis (shared)"
		}
		fmt.Printf("    rate_limit:      on  max=%d  window=%s  store=%s\n",
			h.RateLimitMax, h.RateLimitWindow, backend)
	}
	if h.RequestTimeout > 0 {
		fmt.Printf("    request_timeout: %s\n", h.RequestTimeout)
	} else {
		fmt.Printf("    request_timeout: off\n")
	}
	fmt.Printf("    body_limit:      %s\n", bodyLimitDoctorLine(h.BodyLimit))
	fmt.Printf("    hint:            zatrano config print --paths-only  (see http.*)\n")
}

func printI18nSummary(cfg *config.Config) {
	if !cfg.I18n.Enabled {
		fmt.Printf("  i18n:            off\n")
		return
	}
	fmt.Printf("  i18n:            on  default=%s  locales=%s  files=%v\n",
		cfg.I18n.DefaultLocale, cfg.I18n.LocalesDir, cfg.I18n.SupportedLocales)
	qk := cfg.I18n.QueryKey
	if qk == "" {
		qk = "lang"
	}
	ck := cfg.I18n.CookieName
	if ck == "" {
		ck = "zatrano_lang"
	}
	fmt.Printf("    resolution:    ?%s=  cookie:%s  Accept-Language\n", qk, ck)
}

func originsShort(origins []string) string {
	if len(origins) == 0 {
		return "(default *)"
	}
	if len(origins) == 1 {
		return origins[0]
	}
	if len(origins) == 2 {
		return origins[0] + ", " + origins[1]
	}
	return fmt.Sprintf("%s, … +%d more", origins[0], len(origins)-1)
}

func bodyLimitDoctorLine(n int) string {
	if n <= 0 {
		return "default (Fiber 4 MiB)"
	}
	return fmt.Sprintf("%d bytes", n)
}

func printOAuthSummary(cfg *config.Config) {
	if !cfg.OAuth.Enabled {
		fmt.Printf("  oauth:           disabled\n")
		return
	}
	fmt.Printf("  oauth:           enabled\n")
	fmt.Printf("  oauth base_url:  %s\n", strings.TrimSpace(cfg.OAuth.BaseURL))
	g := cfg.OAuth.Providers.Google
	h := cfg.OAuth.Providers.Github
	fmt.Printf("  oauth google:    %s\n", oauthProviderLine(g))
	fmt.Printf("  oauth github:    %s\n", oauthProviderLine(h))
}

func oauthProviderLine(p config.OAuthProvider) string {
	if strings.TrimSpace(p.ClientID) != "" && strings.TrimSpace(p.ClientSecret) != "" {
		return "client_id + secret set"
	}
	if strings.TrimSpace(p.ClientID) != "" {
		return "client_id set (secret missing)"
	}
	return "not configured"
}

func printPostgresClientTools() {
	fmt.Printf("postgresql client tools (for `zatrano db backup` / `db restore`)\n")
	names := []string{"pg_dump", "pg_restore", "psql"}
	missing := 0
	for _, name := range names {
		path, err := exec.LookPath(name)
		if err != nil {
			fmt.Printf("  %-14s not on PATH\n", name+":")
			missing++
			continue
		}
		fmt.Printf("  %-14s %s\n", name+":", path)
	}
	if missing > 0 {
		fmt.Printf("  hint: install PostgreSQL client tools and ensure they are on your PATH.\n")
	}
}
