package orm_test

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/zatrano/framework/packages/orm"
)

type defaultConnModel struct {
	orm.Model
	Name string `db:"name"`
}

func (defaultConnModel) TableName() string { return "default_conn_models" }

type namedConnModel struct {
	orm.Model
	Name string `db:"name"`
}

func (namedConnModel) TableName() string { return "named_conn_models" }

func (namedConnModel) Connection() string { return "analytics" }

func TestModelConnectionRouting(t *testing.T) {
	t.Parallel()

	defaultDB, err := sql.Open("sqlite", "file:orm_conn_default?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = defaultDB.Close() })
	if _, err := defaultDB.Exec(`CREATE TABLE default_conn_models (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		created_at DATETIME,
		updated_at DATETIME
	)`); err != nil {
		t.Fatal(err)
	}

	analyticsDB, err := sql.Open("sqlite", "file:orm_conn_analytics?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = analyticsDB.Close() })
	if _, err := analyticsDB.Exec(`CREATE TABLE named_conn_models (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		created_at DATETIME,
		updated_at DATETIME
	)`); err != nil {
		t.Fatal(err)
	}

	orm.Configure(defaultDB, "sqlite")
	orm.SetConnectionResolver(func(name string) (*sql.DB, string, error) {
		if name == "analytics" {
			return analyticsDB, "sqlite", nil
		}
		return nil, "", sql.ErrConnDone
	})
	t.Cleanup(func() { orm.SetConnectionResolver(nil) })

	if got := orm.ConnectionName[namedConnModel](); got != "analytics" {
		t.Fatalf("ConnectionName = %q, want analytics", got)
	}
	if got := orm.ConnectionName[defaultConnModel](); got != "" {
		t.Fatalf("ConnectionName default = %q, want empty", got)
	}

	if _, err := orm.Create[namedConnModel](map[string]any{"name": "on-analytics"}); err != nil {
		t.Fatal(err)
	}
	if _, err := orm.Create[defaultConnModel](map[string]any{"name": "on-default"}); err != nil {
		t.Fatal(err)
	}

	named, err := orm.Where[namedConnModel]("name", "on-analytics").First()
	if err != nil {
		t.Fatal(err)
	}
	if named.Name != "on-analytics" {
		t.Fatalf("named name = %q", named.Name)
	}

	var count int
	if err := defaultDB.QueryRow(`SELECT COUNT(*) FROM default_conn_models`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("default table count = %d", count)
	}
	if err := analyticsDB.QueryRow(`SELECT COUNT(*) FROM named_conn_models`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("analytics table count = %d", count)
	}
	if err := defaultDB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE name='named_conn_models'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("named model leaked onto default connection")
	}
}
