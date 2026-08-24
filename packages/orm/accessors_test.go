package orm_test

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/zatrano/framework/packages/orm"
)

type accessorModel struct {
	orm.Model
	First string `db:"first_name"`
	Last  string `db:"last_name"`
}

func (accessorModel) TableName() string { return "accessor_models" }

func (m *accessorModel) GetAttribute(name string) (any, bool) {
	switch name {
	case "full_name":
		return strings.TrimSpace(m.First + " " + m.Last), true
	case "first_name":
		return strings.ToUpper(m.First), true
	default:
		return nil, false
	}
}

func (m *accessorModel) Appends() []string { return []string{"full_name"} }

func (accessorModel) SetAttribute(name string, value any) (any, bool) {
	if name == "first_name" {
		return strings.ToLower(strings.TrimSpace(fmt.Sprint(value))), true
	}
	return nil, false
}

func (accessorModel) Defaults() map[string]any {
	return map[string]any{"last_name": "Doe"}
}

type uuidModel struct {
	ID   string `db:"id"`
	Name string `db:"name"`
}

func (uuidModel) TableName() string  { return "uuid_models" }
func (uuidModel) PrimaryKey() string { return "id" }
func (uuidModel) UsesUUIDKeys() bool { return true }

func TestAccessorsMutatorsDefaultsUUID(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, sqlStr := range []string{
		`CREATE TABLE accessor_models (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			first_name TEXT,
			last_name TEXT,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE uuid_models (
			id TEXT PRIMARY KEY,
			name TEXT,
			created_at DATETIME,
			updated_at DATETIME
		)`,
	} {
		if _, err := db.Exec(sqlStr); err != nil {
			t.Fatal(err)
		}
	}
	orm.Configure(db, "sqlite")

	created, err := orm.Create[accessorModel](map[string]any{"first_name": " ADA "})
	if err != nil {
		t.Fatal(err)
	}
	if created.First != "ada" {
		t.Fatalf("mutator first=%q", created.First)
	}
	if created.Last != "Doe" {
		t.Fatalf("default last=%q", created.Last)
	}
	m := orm.ToMap(created)
	if m["first_name"] != "ADA" {
		t.Fatalf("accessor first=%v", m["first_name"])
	}
	if m["full_name"] != "ada Doe" {
		t.Fatalf("full_name=%v", m["full_name"])
	}

	u, err := orm.Create[uuidModel](map[string]any{"name": "x"})
	if err != nil {
		t.Fatal(err)
	}
	if !orm.IsUUID(u.ID) {
		t.Fatalf("uuid=%q", u.ID)
	}
}
