package database_test

import (
	"database/sql"
	"os"
	"testing"

	"github.com/zatrano/framework/packages/database/query"
	"github.com/zatrano/framework/packages/database/schema"
)

// Live driver smoke tests. Drivers must already be linked (db:setup).
// Set ZATRANO_LIVE_DB=1 and DB_CONNECTION=mysql|pgsql|mssql|oracle.

func TestLiveDriverSmoke(t *testing.T) {
	if os.Getenv("ZATRANO_LIVE_DB") != "1" {
		t.Skip("set ZATRANO_LIVE_DB=1 to run live driver smoke")
	}
	driver := os.Getenv("DB_CONNECTION")
	if driver == "" {
		t.Fatal("DB_CONNECTION required")
	}
	norm := normalizeDriver(driver)
	sqlName := map[string]string{
		"mysql": "mysql", "pgsql": "postgres", "mssql": "sqlserver", "oracle": "oracle",
	}[norm]
	if sqlName == "" {
		t.Fatalf("unsupported %q", driver)
	}
	if !driverLinked(sqlName) {
		t.Skipf("SQL driver %q not linked — run db:setup --drivers=%s", sqlName, norm)
	}

	db, err := openLive(t, driver)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	s := schema.New(db, norm)
	_ = s.DropIfExists("zatrano_live_smoke")
	if err := s.Create("zatrano_live_smoke", func(bp *schema.Blueprint) {
		bp.ID()
		bp.String("name")
		bp.Timestamps()
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	defer func() { _ = s.DropIfExists("zatrano_live_smoke") }()

	id, err := query.New(db, norm, "zatrano_live_smoke").InsertGetID(map[string]any{"name": "ok"})
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected insert id > 0, got %d", id)
	}

	rows, err := query.New(db, norm, "zatrano_live_smoke").Where("name", "ok").Get()
	if err != nil {
		t.Fatalf("select: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
}

func driverLinked(name string) bool {
	for _, d := range sql.Drivers() {
		if d == name {
			return true
		}
	}
	return false
}

func normalizeDriver(d string) string {
	switch d {
	case "postgres", "postgresql":
		return "pgsql"
	case "sqlserver":
		return "mssql"
	case "ora":
		return "oracle"
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
	case "oracle":
		user := envOr("DB_USERNAME", "system")
		port := envOr("DB_PORT", "1521")
		svc := envOr("DB_SERVICE", dbName)
		dsn := "oracle://" + user + ":" + pass + "@" + host + ":" + port + "/" + svc
		return sql.Open("oracle", dsn)
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
