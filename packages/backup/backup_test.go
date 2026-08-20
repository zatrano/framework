package backup_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/backup"
	"github.com/zatrano/framework/packages/config"
)

func TestBackupCreateListRestore(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "app.sqlite")
	if err := os.WriteFile(src, []byte("sqlite-demo"), 0o644); err != nil {
		t.Fatal(err)
	}
	mgr := backup.New(src, filepath.Join(dir, "backups"))
	path, err := mgr.Create("demo")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(path, ".sqlite") {
		t.Fatalf("ext: %s", path)
	}
	files, err := mgr.List()
	if err != nil || len(files) != 1 {
		t.Fatalf("files=%v err=%v", files, err)
	}
	_ = os.WriteFile(src, []byte("changed"), 0o644)
	if err := mgr.Restore(filepath.Base(path)); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(src)
	if string(raw) != "sqlite-demo" {
		t.Fatalf("got %q", raw)
	}
}

func TestListMultipleExtensions(t *testing.T) {
	dir := t.TempDir()
	backups := filepath.Join(dir, "backups")
	_ = os.MkdirAll(backups, 0o755)
	for _, name := range []string{"backup_1.sql", "backup_2.dump", "backup_3.archive", "notes.txt"} {
		_ = os.WriteFile(filepath.Join(backups, name), []byte("x"), 0o644)
	}
	mgr := backup.NewManager(backup.Config{Driver: "mysql", Dir: backups})
	files, err := mgr.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 3 {
		t.Fatalf("want 3 got %d %#v", len(files), files)
	}
}

func TestConfigFromApp(t *testing.T) {
	dir := t.TempDir()
	repo := config.New()
	repo.Set("database.default", "mysql")
	repo.Set("database.connections.mysql", map[string]any{
		"driver":   "mysql",
		"host":     "db.example",
		"port":     "3307",
		"database": "appdb",
		"username": "root",
		"password": "secret",
	})
	app := &fakeApp{base: dir, cfg: repo}
	cfg, err := backup.ConfigFromApp(app, repo, "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Driver != "mysql" || cfg.Host != "db.example" || cfg.Port != "3307" || cfg.Database != "appdb" {
		t.Fatalf("%+v", cfg)
	}
	if cfg.Dir != filepath.Join(dir, "storage", "backups") {
		t.Fatalf("dir=%q", cfg.Dir)
	}
}

type fakeApp struct {
	base string
	cfg  *config.Repository
}

func (f *fakeApp) BasePath(parts ...string) string {
	return filepath.Join(append([]string{f.base}, parts...)...)
}

func TestMySQLArgsViaFakeBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script fake binary is unix-oriented")
	}
	dir := t.TempDir()
	logPath := filepath.Join(dir, "mysqldump.log")
	script := filepath.Join(dir, "mysqldump")
	body := "#!/bin/sh\n" +
		"echo \"$@\" > '" + logPath + "'\n" +
		"for a in \"$@\"; do\n" +
		"  case \"$a\" in --result-file=*) f=${a#--result-file=}; : > \"$f\" ;; esac\n" +
		"done\n"
	if err := os.WriteFile(script, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	if _, err := exec.LookPath("mysqldump"); err != nil {
		t.Fatal(err)
	}
	backups := filepath.Join(dir, "backups")
	mgr := backup.NewManager(backup.Config{
		Driver:   "mysql",
		Host:     "127.0.0.1",
		Port:     "3306",
		Database: "zatrano",
		Username: "root",
		Password: "x",
		Dir:      backups,
	})
	path, err := mgr.Create("t")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(logPath)
	line := string(raw)
	if !strings.Contains(line, "-h") || !strings.Contains(line, "zatrano") {
		t.Fatalf("args log: %q", line)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
}

func TestRedactSecrets(t *testing.T) {
	msg := "login failed for password=s3cret and uri=mongodb://u:s3cret@h/db"
	got := backup.RedactSecretsForTest(msg, "s3cret", "mongodb://u:s3cret@h/db")
	if strings.Contains(got, "s3cret") {
		t.Fatalf("secret leaked: %q", got)
	}
}

func TestMongoConfigKeepsPasswordOutOfArgs(t *testing.T) {
	mgr := backup.NewManager(backup.Config{
		Driver:   "mongo",
		Host:     "127.0.0.1",
		Port:     "27017",
		Username: "root",
		Password: "super-secret-pass",
		Database: "app",
		Dir:      t.TempDir(),
	})
	cfg, secrets := backup.MongoConfigYAMLForTest(mgr)
	if !strings.Contains(cfg, "super-secret-pass") {
		t.Fatal("expected password in config yaml body")
	}
	if len(secrets) == 0 || secrets[0] != "super-secret-pass" {
		t.Fatalf("secrets=%v", secrets)
	}
	if strings.Contains(cfg, "--password") {
		t.Fatal("config must not use argv-style flags")
	}
}

func TestBackupPathTraversal(t *testing.T) {
	dir := t.TempDir()
	backups := filepath.Join(dir, "backups")
	outside := filepath.Join(dir, "secret.sqlite")
	if err := os.WriteFile(outside, []byte("pwn"), 0o600); err != nil {
		t.Fatal(err)
	}
	mgr := backup.NewManager(backup.Config{Driver: "sqlite", Database: filepath.Join(dir, "app.sqlite"), Dir: backups})
	_ = os.WriteFile(filepath.Join(dir, "app.sqlite"), []byte("db"), 0o600)

	if err := mgr.Restore("../secret.sqlite"); err == nil {
		t.Fatal("relative traversal must be rejected")
	}
	if err := mgr.Restore(outside); err == nil {
		t.Fatal("absolute path outside backup dir must be rejected")
	}
	if err := mgr.Restore("..\\secret.sqlite"); err == nil {
		t.Fatal("windows-style traversal must be rejected")
	}
}

func TestBackupRestoreInsideDirAbsolute(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "app.sqlite")
	backups := filepath.Join(dir, "backups")
	if err := os.WriteFile(src, []byte("sqlite-demo"), 0o600); err != nil {
		t.Fatal(err)
	}
	mgr := backup.New(src, backups)
	path, err := mgr.Create("ok")
	if err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(src, []byte("changed"), 0o600)
	if err := mgr.Restore(path); err != nil {
		t.Fatalf("absolute path inside backup dir should work: %v", err)
	}
	raw, _ := os.ReadFile(src)
	if string(raw) != "sqlite-demo" {
		t.Fatalf("got %q", raw)
	}
}

func TestBackupInvalidConnectionName(t *testing.T) {
	dir := t.TempDir()
	app := &fakeApp{base: dir}
	cfg := config.New()
	cfg.Set("database.default", "sqlite")
	cfg.Set("database.connections.sqlite.driver", "sqlite")
	cfg.Set("database.connections.sqlite.database", "database/database.sqlite")
	_, err := backup.ConfigFromApp(app, cfg, "../etc/passwd")
	if err == nil {
		t.Fatal("expected invalid connection name")
	}
	_, err = backup.ConfigFromApp(app, cfg, "sqlite;rm -rf")
	if err == nil {
		t.Fatal("expected invalid connection name")
	}
}
