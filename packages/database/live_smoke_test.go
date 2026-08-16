package database_test

import (
	"database/sql"
	"os"
	"testing"

	"github.com/zatrano/framework/packages/database/query"
	"github.com/zatrano/framework/packages/database/schema"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/microsoft/go-mssqldb"
)

// Live driver smoke tests. Set ZATRANO_LIVE_DB=1 and matching DB_* env vars.
//
// MySQL:  ZATRANO_LIVE_DB=1 DB_CONNECTION=mysql DB_HOST=127.0.0.1 DB_PORT=3306 DB_DATABASE=zatrano DB_USERNAME=root go test ./packages/database -run Live -count=1
// Postgres: ZATRANO_LIVE_DB=1 DB_CONNECTION=pgsql DB_USERNAME=postgres DB_PASSWORD=secret DB_DATABASE=zatrano go test ./packages/database -run Live -count=1
// SQL Server: ZATRANO_LIVE_DB=1 DB_CONNECTION=mssql DB_USERNAME=sa DB_PASSWORD=Your_strong_Password123 DB_DATABASE=master go test ./packages/database -run Live -count=1

func TestLiveDriverSmoke(t *testing.T) {
	if os.Getenv("ZATRANO_LIVE_DB") != "1" {
		t.Skip("set ZATRANO_LIVE_DB=1 to run live driver smoke")
	}
	driver := os.Getenv("DB_CONNECTION")
	if driver == "" {
		t.Fatal("DB_CONNECTION required")
	}

	db, err := openLive(t, driver)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	s := schema.New(db, normalizeDriver(driver))
	_ = s.DropIfExists("zatrano_live_smoke")
	if err := s.Create("zatrano_live_smoke", func(bp *schema.Blueprint) {
		bp.ID()
		bp.String("name")
		bp.Timestamps()
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	defer func() { _ = s.DropIfExists("zatrano_live_smoke") }()

	b := query.New(db, normalizeDriver(driver), "zatrano_live_smoke")
	id, err := b.Insert(map[string]any{"name": "ok"})
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected insert id > 0, got %d", id)
	}

	rows, err := query.New(db, normalizeDriver(driver), "zatrano_live_smoke").Where("name", "ok").Get()
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
}

func normalizeDriver(d string) string {
	switch d {
	case "postgres", "postgresql":
		return "pgsql"
	case "sqlserver":
		return "mssql"
	default:
		return d
	}
}

func openLive(t *testing.T, driver string) (*sql.DB, error) {
	t.Helper()
	host := envOr("DB_HOST", "127.0.0.1")
	pass := os.Getenv("DB_PASSWORD")
	dbName := envOr("DB_DATABASE", "zatrano")
	switch normalizeDriver(driver) {
	case "mysql":
		user := envOr("DB_USERNAME", "root")
		port := envOr("DB_PORT", "3306")
		dsn := user + ":" + pass + "@tcp(" + host + ":" + port + ")/" + dbName + "?parseTime=true&loc=Local"
		return sql.Open("mysql", dsn)
	case "pgsql":
		user := envOr("DB_USERNAME", "postgres")
		port := envOr("DB_PORT", "5432")
		ssl := envOr("DB_SSLMODE", "disable")
		dsn := "host=" + host + " port=" + port + " user=" + user + " password=" + pass + " dbname=" + dbName + " sslmode=" + ssl
		return sql.Open("postgres", dsn)
	case "mssql":
		user := envOr("DB_USERNAME", "sa")
		port := envOr("DB_PORT", "1433")
		dsn := "sqlserver://" + user + ":" + pass + "@" + host + ":" + port + "?database=" + dbName
		return sql.Open("sqlserver", dsn)
	default:
		t.Fatalf("unsupported live driver %q", driver)
		return nil, nil
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
