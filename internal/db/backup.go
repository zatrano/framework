package db

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// BackupFormat selects pg_dump output format.
type BackupFormat string

const (
	FormatCustom    BackupFormat = "custom"    // -Fc (recommended for pg_restore)
	FormatPlain     BackupFormat = "plain"     // -Fp (plain SQL)
	FormatDirectory BackupFormat = "directory" // -Fd (directory archive)
)

// Backup runs pg_dump. Requires `pg_dump` on PATH (PostgreSQL client tools).
func Backup(databaseURL, outPath string, format BackupFormat) error {
	if strings.TrimSpace(databaseURL) == "" {
		return fmt.Errorf("database URL is empty (set ZATRANO_DATABASE_URL)")
	}
	outPath = filepath.Clean(outPath)
	dir := filepath.Dir(outPath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	args := []string{"--no-owner", "--no-acl"}
	switch format {
	case FormatCustom:
		args = append(args, "-Fc", "-f", outPath)
	case FormatPlain:
		args = append(args, "-Fp", "-f", outPath)
	case FormatDirectory:
		if err := os.MkdirAll(outPath, 0o755); err != nil {
			return err
		}
		args = append(args, "-Fd", "-f", outPath)
	default:
		return fmt.Errorf("unknown backup format %q (use custom, plain, directory)", format)
	}
	args = append(args, databaseURL)

	cmd := exec.Command("pg_dump", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_dump: %w\n\nhint: install PostgreSQL client tools and ensure pg_dump is on PATH", err)
	}
	return nil
}

// Restore loads a backup. Plain SQL uses psql -f; custom/directory use pg_restore.
// Requires `psql` / `pg_restore` on PATH.
func Restore(databaseURL, inputPath string, format BackupFormat, clean bool) error {
	if strings.TrimSpace(databaseURL) == "" {
		return fmt.Errorf("database URL is empty (set ZATRANO_DATABASE_URL)")
	}
	inputPath = filepath.Clean(inputPath)
	fi, err := os.Stat(inputPath)
	if err != nil {
		return fmt.Errorf("backup path: %w", err)
	}

	if format == FormatPlain {
		return restorePlain(databaseURL, inputPath)
	}

	// custom single file or directory format
	return restoreArchive(databaseURL, inputPath, clean, fi.IsDir())
}

func restorePlain(databaseURL, sqlFile string) error {
	cmd := exec.Command("psql", databaseURL, "-v", "ON_ERROR_STOP=1", "-f", sqlFile)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("psql: %w\n\nhint: install PostgreSQL client tools and ensure psql is on PATH", err)
	}
	return nil
}

func restoreArchive(databaseURL, path string, clean, isDir bool) error {
	args := []string{"--dbname", databaseURL, "--no-owner"}
	if clean {
		args = append(args, "--clean", "--if-exists")
	}
	if isDir {
		args = append(args, path)
	} else {
		args = append(args, path)
	}
	cmd := exec.Command("pg_restore", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_restore: %w\n\nhint: install PostgreSQL client tools and ensure pg_restore is on PATH", err)
	}
	return nil
}

// DefaultBackupPath returns backups/zatrano-<utc-timestamp>.<ext> (or a directory path for directory format).
func DefaultBackupPath(baseDir string, format BackupFormat) (string, error) {
	ts := time.Now().UTC().Format("20060102-150405")
	if baseDir == "" {
		baseDir = "backups"
	}
	switch format {
	case FormatDirectory:
		return filepath.Join(baseDir, "zatrano-"+ts), nil
	case FormatPlain:
		return filepath.Join(baseDir, fmt.Sprintf("zatrano-%s.sql", ts)), nil
	default:
		return filepath.Join(baseDir, fmt.Sprintf("zatrano-%s.dump", ts)), nil
	}
}

// InferFormatFromPath guesses format from extension.
func InferFormatFromPath(p string) BackupFormat {
	switch strings.ToLower(filepath.Ext(p)) {
	case ".sql":
		return FormatPlain
	case ".dump", ".backup":
		return FormatCustom
	default:
		// no extension: treat as directory if path exists and is dir
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			return FormatDirectory
		}
		return FormatCustom
	}
}

