package orm_test

import (
	"database/sql"
	"fmt"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/zatrano/framework/packages/orm"
)

type upsertItem struct {
	orm.Model
	Email string `db:"email"`
	Name  string `db:"name"`
	Score int64  `db:"score"`
}

func (upsertItem) TableName() string { return "upsert_items" }

func TestWhereBetweenNotInAndUpsert(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE upsert_items (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT NOT NULL UNIQUE,
		name TEXT,
		score INTEGER DEFAULT 0,
		created_at DATETIME,
		updated_at DATETIME
	)`)
	if err != nil {
		t.Fatal(err)
	}
	orm.Configure(db, "sqlite")

	for _, score := range []int64{1, 5, 10, 15} {
		if _, err := orm.Create[upsertItem](map[string]any{
			"email": fmt.Sprintf("u%d@test", score),
			"name":  "n",
			"score": score,
		}); err != nil {
			t.Fatal(err)
		}
	}

	mid, err := orm.Query[upsertItem]().WhereBetween("score", 5, 10).OrderBy("score").Get()
	if err != nil || len(mid) != 2 {
		t.Fatalf("between: len=%d err=%v", len(mid), err)
	}
	out, err := orm.Query[upsertItem]().WhereNotBetween("score", 5, 10).Get()
	if err != nil || len(out) != 2 {
		t.Fatalf("not between: len=%d err=%v", len(out), err)
	}
	filtered, err := orm.Query[upsertItem]().WhereNotIn("score", []any{int64(1), int64(15)}).Get()
	if err != nil || len(filtered) != 2 {
		t.Fatalf("not in: len=%d err=%v", len(filtered), err)
	}

	affected, err := orm.Upsert[upsertItem](map[string]any{
		"email": "unique@test",
		"name":  "Ada",
		"score": int64(7),
	}, []string{"email"})
	if err != nil || affected < 1 {
		t.Fatalf("upsert insert: affected=%d err=%v", affected, err)
	}
	affected, err = orm.Upsert[upsertItem](map[string]any{
		"email": "unique@test",
		"name":  "Grace",
		"score": int64(9),
	}, []string{"email"}, "name", "score", "updated_at")
	if err != nil {
		t.Fatal(err)
	}
	row, err := orm.Query[upsertItem]().Where("email", "unique@test").First()
	if err != nil || row.Name != "Grace" || row.Score != 9 {
		t.Fatalf("upsert update: %+v err=%v affected=%d", row, err, affected)
	}
}
