package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zatrano/framework/internal/db"
	"github.com/zatrano/framework/pkg/config"
)

var dbCmd = &cobra.Command{
	Use:   "db",
	Short: "Database migrations, seeds, backup, and restore",
}

var dbMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Apply SQL migrations (golang-migrate)",
	Long:  `Runs *.up.sql migrations from migrations_dir (default ./migrations). Requires DATABASE_URL.`,
	RunE:  runDBMigrate,
}

var dbRollbackCmd = &cobra.Command{
	Use:   "rollback",
	Short: "Rollback the last migration step (or use --steps)",
	RunE:  runDBRollback,
}

var dbSeedCmd = &cobra.Command{
	Use:   "seed",
	Short: "Run all *.sql files in seeds_dir inside one transaction",
	RunE:  runDBSeed,
}

var dbBackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Run pg_dump to a file (PostgreSQL client tools required)",
	Long: `Creates a logical backup using pg_dump.

Formats:
  custom     -Fc (default, best for pg_restore)
  plain      -Fp SQL text
  directory  -Fd directory archive`,
	RunE: runDBBackup,
}

var dbRestoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore from pg_dump output (destructive — requires --yes)",
	Long: `Uses psql for plain SQL or pg_restore for custom/directory archives.

This can DROP objects when --clean is used. Always test on a copy first.`,
	RunE: runDBRestore,
}

func init() {
	dbMigrateCmd.Flags().String("env", "", "environment name; default ENV or dev")
	dbMigrateCmd.Flags().String("config-dir", "config", "directory containing {env}.yaml")
	dbMigrateCmd.Flags().Bool("no-dotenv", false, "do not load .env")
	dbMigrateCmd.Flags().String("migrations", "", "override migrations directory (default from config migrations_dir)")

	dbRollbackCmd.Flags().String("env", "", "environment name; default ENV or dev")
	dbRollbackCmd.Flags().String("config-dir", "config", "directory containing {env}.yaml")
	dbRollbackCmd.Flags().Bool("no-dotenv", false, "do not load .env")
	dbRollbackCmd.Flags().String("migrations", "", "override migrations directory")
	dbRollbackCmd.Flags().Int("steps", 1, "number of migrations to roll back")

	dbSeedCmd.Flags().String("env", "", "environment name; default ENV or dev")
	dbSeedCmd.Flags().String("config-dir", "config", "directory containing {env}.yaml")
	dbSeedCmd.Flags().Bool("no-dotenv", false, "do not load .env")
	dbSeedCmd.Flags().String("seeds", "", "override seeds directory (default from config seeds_dir)")

	dbBackupCmd.Flags().String("env", "", "environment name; default ENV or dev")
	dbBackupCmd.Flags().String("config-dir", "config", "directory containing {env}.yaml")
	dbBackupCmd.Flags().Bool("no-dotenv", false, "do not load .env")
	dbBackupCmd.Flags().String("output", "", "output file or directory (default: backups/zatrano-<timestamp>.dump)")
	dbBackupCmd.Flags().String("format", "custom", "custom | plain | directory")

	dbRestoreCmd.Flags().String("env", "", "environment name; default ENV or dev")
	dbRestoreCmd.Flags().String("config-dir", "config", "directory containing {env}.yaml")
	dbRestoreCmd.Flags().Bool("no-dotenv", false, "do not load .env")
	dbRestoreCmd.Flags().String("input", "", "backup file or directory (required)")
	dbRestoreCmd.Flags().String("format", "", "custom | plain | directory (default: infer from --input)")
	dbRestoreCmd.Flags().Bool("clean", true, "pass --clean --if-exists to pg_restore (custom/directory only)")
	dbRestoreCmd.Flags().Bool("yes", false, "required acknowledgement for restore")

	dbCmd.AddCommand(dbMigrateCmd, dbRollbackCmd, dbSeedCmd, dbBackupCmd, dbRestoreCmd)
	rootCmd.AddCommand(dbCmd)
}

func runDBMigrate(cmd *cobra.Command, _ []string) error {
	cfg, err := loadCfgFlags(cmd)
	if err != nil {
		return err
	}
	dir := cfg.MigrationsDir
	if f, _ := cmd.Flags().GetString("migrations"); f != "" {
		dir = f
	}
	ver, dirty, err := db.MigrateUp(cfg.DatabaseURL, dir, 0)
	if err != nil {
		return fmt.Errorf("migrate up: %w\n\nhint: ensure postgres is reachable and migrations use versioned filenames (e.g. 000001_name.up.sql)", err)
	}
	fmt.Printf("ok: version=%d dirty=%v\n", ver, dirty)
	return nil
}

func runDBRollback(cmd *cobra.Command, _ []string) error {
	cfg, err := loadCfgFlags(cmd)
	if err != nil {
		return err
	}
	dir := cfg.MigrationsDir
	if f, _ := cmd.Flags().GetString("migrations"); f != "" {
		dir = f
	}
	steps, _ := cmd.Flags().GetInt("steps")
	ver, dirty, err := db.MigrateDown(cfg.DatabaseURL, dir, steps)
	if err != nil {
		return fmt.Errorf("migrate down: %w", err)
	}
	fmt.Printf("ok: version=%d dirty=%v\n", ver, dirty)
	return nil
}

func runDBSeed(cmd *cobra.Command, _ []string) error {
	cfg, err := loadCfgFlags(cmd)
	if err != nil {
		return err
	}
	dir := cfg.SeedsDir
	if f, _ := cmd.Flags().GetString("seeds"); f != "" {
		dir = f
	}
	if err := db.RunSeeds(cfg.DatabaseURL, dir); err != nil {
		return fmt.Errorf("seed: %w\n\nhint: add ordered .sql files under %s", err, dir)
	}
	fmt.Println("ok: seeds finished (no-op if no .sql files)")
	return nil
}

func loadCfgFlags(cmd *cobra.Command) (*config.Config, error) {
	env, _ := cmd.Flags().GetString("env")
	dir, _ := cmd.Flags().GetString("config-dir")
	noDot, _ := cmd.Flags().GetBool("no-dotenv")
	return config.Load(config.LoadOptions{
		Env:       env,
		ConfigDir: dir,
		DotEnv:    !noDot,
	})
}

func runDBBackup(cmd *cobra.Command, _ []string) error {
	cfg, err := loadCfgFlags(cmd)
	if err != nil {
		return err
	}
	out, _ := cmd.Flags().GetString("output")
	formatStr, _ := cmd.Flags().GetString("format")
	f := db.BackupFormat(strings.ToLower(strings.TrimSpace(formatStr)))
	if f != db.FormatCustom && f != db.FormatPlain && f != db.FormatDirectory {
		return fmt.Errorf("invalid --format %q", formatStr)
	}
	if out == "" {
		var e error
		out, e = db.DefaultBackupPath("backups", f)
		if e != nil {
			return e
		}
	}
	if err := db.Backup(cfg.DatabaseURL, out, f); err != nil {
		return err
	}
	fmt.Printf("ok: backup written to %s\n", out)
	return nil
}

func runDBRestore(cmd *cobra.Command, _ []string) error {
	yes, _ := cmd.Flags().GetBool("yes")
	if !yes {
		return fmt.Errorf("refusing to restore without --yes (this can destroy data). Re-run with --yes after reading the docs")
	}
	cfg, err := loadCfgFlags(cmd)
	if err != nil {
		return err
	}
	in, _ := cmd.Flags().GetString("input")
	if strings.TrimSpace(in) == "" {
		return fmt.Errorf("--input is required")
	}
	formatStr, _ := cmd.Flags().GetString("format")
	var f db.BackupFormat
	if strings.TrimSpace(formatStr) == "" {
		f = db.InferFormatFromPath(in)
	} else {
		f = db.BackupFormat(strings.ToLower(strings.TrimSpace(formatStr)))
	}
	if f != db.FormatCustom && f != db.FormatPlain && f != db.FormatDirectory {
		return fmt.Errorf("invalid --format %q", formatStr)
	}
	clean, _ := cmd.Flags().GetBool("clean")
	if err := db.Restore(cfg.DatabaseURL, in, f, clean); err != nil {
		return err
	}
	fmt.Println("ok: restore finished")
	return nil
}
