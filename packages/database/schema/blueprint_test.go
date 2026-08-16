package schema_test

import (
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/database/schema"
)

func TestForeignIDConstrainedCreateSQL(t *testing.T) {
	b := schema.NewBlueprint("posts", "pgsql")
	b.ID()
	b.ForeignID("user_id").Constrained("users").CascadeOnDelete()
	sql, err := b.ToCreateSQL()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sql, "user_id BIGINT") {
		t.Fatalf("expected bigint column, got %s", sql)
	}
	if !strings.Contains(sql, "REFERENCES users(id)") {
		t.Fatalf("expected REFERENCES, got %s", sql)
	}
	if !strings.Contains(sql, "ON DELETE CASCADE") {
		t.Fatalf("expected ON DELETE CASCADE, got %s", sql)
	}
}
