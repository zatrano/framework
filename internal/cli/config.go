package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	yaml "go.yaml.in/yaml/v3"

	"github.com/zatrano/framework/pkg/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Inspect resolved configuration (secrets masked)",
}

var configPrintCmd = &cobra.Command{
	Use:   "print",
	Short: "Print effective config after .env, YAML, and environment variables merge",
	Long: `Loads configuration the same way as serve/doctor and prints a sanitized view:
database/redis URLs and secrets are masked; OAuth client IDs stay visible, secrets masked.

Formats: json (default), yaml.

Use --paths-only for a short list of env, working directory, config profile path, and key dirs
(no secrets). Default output for --paths-only is human-readable lines unless you set --format json|yaml.`,
	RunE: runConfigPrint,
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Load and validate config only (no DB/Redis connections)",
	Long: `Runs the same merge and validation rules as serve/doctor: .env, config/{env}.yaml, environment variables.

Use in CI or pre-commit when you only need schema and cross-field checks — faster than doctor
because it does not bootstrap the app or probe databases.

Exit code 0 on success, non-zero on validation error.`,
	RunE: runConfigValidate,
}

func init() {
	configPrintCmd.Flags().String("env", "", "environment name; default ENV or dev")
	configPrintCmd.Flags().String("config-dir", "config", "directory containing {env}.yaml")
	configPrintCmd.Flags().Bool("no-dotenv", false, "do not load .env from the working directory")
	configPrintCmd.Flags().Bool("paths-only", false, "only print paths summary (no secrets); default format becomes lines unless --format is set")
	configPrintCmd.Flags().String("format", "json", "output format: json, yaml, or lines (lines is typical with --paths-only)")

	configValidateCmd.Flags().String("env", "", "environment name; default ENV or dev")
	configValidateCmd.Flags().String("config-dir", "config", "directory containing {env}.yaml")
	configValidateCmd.Flags().Bool("no-dotenv", false, "do not load .env from the working directory")
	configValidateCmd.Flags().BoolP("quiet", "q", false, "no stdout on success (exit code only)")

	configCmd.AddCommand(configPrintCmd)
	configCmd.AddCommand(configValidateCmd)
	rootCmd.AddCommand(configCmd)
}

func runConfigValidate(cmd *cobra.Command, _ []string) error {
	envFlag, _ := cmd.Flags().GetString("env")
	configDir, _ := cmd.Flags().GetString("config-dir")
	noDotenv, _ := cmd.Flags().GetBool("no-dotenv")
	quiet, _ := cmd.Flags().GetBool("quiet")

	cfg, err := config.Load(config.LoadOptions{
		Env:       envFlag,
		ConfigDir: configDir,
		DotEnv:    !noDotenv,
	})
	if err != nil {
		return fmt.Errorf("config validate: %w", err)
	}
	if !quiet {
		_, _ = fmt.Fprintf(os.Stdout, "config: OK (env=%s, app=%s, listen=%s)\n",
			cfg.Env, cfg.AppName, cfg.HTTPAddr)
	}
	return nil
}

func runConfigPrint(cmd *cobra.Command, _ []string) error {
	envFlag, _ := cmd.Flags().GetString("env")
	configDir, _ := cmd.Flags().GetString("config-dir")
	noDotenv, _ := cmd.Flags().GetBool("no-dotenv")
	pathsOnly, _ := cmd.Flags().GetBool("paths-only")
	format, _ := cmd.Flags().GetString("format")
	if pathsOnly && !cmd.Flags().Changed("format") {
		format = "lines"
	}

	cfg, err := config.Load(config.LoadOptions{
		Env:       envFlag,
		ConfigDir: configDir,
		DotEnv:    !noDotenv,
	})
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	format = strings.ToLower(strings.TrimSpace(format))

	if pathsOnly {
		wd, err := os.Getwd()
		if err != nil {
			wd = "."
		}
		dotenvPresent := false
		if _, err := os.Stat(filepath.Join(wd, ".env")); err == nil {
			dotenvPresent = true
		}
		view := config.PathsView(cfg, wd, configDir, dotenvPresent)
		switch format {
		case "json":
			b, err := json.MarshalIndent(view, "", "  ")
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(os.Stdout, string(b))
			return nil
		case "yaml", "yml":
			b, err := yaml.Marshal(view)
			if err != nil {
				return err
			}
			_, _ = os.Stdout.Write(b)
			return nil
		case "lines", "":
			keys := []string{
				"env", "working_dir", "dotenv", "config_dir", "config_profile",
				"http_addr", "openapi_path", "migrations_dir", "seeds_dir",
			}
			for _, k := range keys {
				_, _ = fmt.Fprintf(os.Stdout, "%s: %v\n", k, view[k])
			}
			return nil
		default:
			return fmt.Errorf("unknown --format %q for --paths-only (use lines, json, or yaml)", format)
		}
	}

	snap := config.SanitizedSnapshot(cfg)
	switch format {
	case "json", "":
		b, err := json.MarshalIndent(snap, "", "  ")
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintln(os.Stdout, string(b))
		return nil
	case "yaml", "yml":
		b, err := yaml.Marshal(snap)
		if err != nil {
			return err
		}
		_, _ = os.Stdout.Write(b)
		return nil
	case "lines":
		return fmt.Errorf(`--format lines is only valid with --paths-only`)
	default:
		return fmt.Errorf("unknown --format %q (use json or yaml)", format)
	}
}
