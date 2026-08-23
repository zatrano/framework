package database_test

import (
	"database/sql"
	"testing"

	"github.com/zatrano/framework/packages/database/query"
	"github.com/zatrano/framework/packages/database/schema"

	_ "github.com/zatrano/framework/packages/database/driver/sqlite"
)

func TestSQLiteSmokeCreateInsertSelect(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	s := schema.New(db, "sqlite")
	if err := s.Create("items", func(b *schema.Blueprint) {
		b.ID()
		b.String("name")
		b.ForeignID("user_id").Nullable()
		b.JSON("meta").Nullable()
		b.Timestamps()
	}); err != nil {
		t.Fatal(err)
	}

	id, err := query.New(db, "sqlite", "items").InsertGetID(map[string]any{"name": "alpha"})
	if err != nil {
		t.Fatal(err)
	}
	if id != 1 {
		t.Fatalf("id=%d", id)
	}

	if err := s.Table("items", func(b *schema.Blueprint) {
		b.String("sku").Nullable()
	}); err != nil {
		t.Fatalf("alter add column: %v", err)
	}
	ok, err := s.HasColumn("items", "sku")
	if err != nil || !ok {
		t.Fatalf("HasColumn sku: %v %v", ok, err)
	}
}
