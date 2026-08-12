package orm_test

import (
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/zatrano/framework/packages/orm"
)

type touchModel struct {
	orm.Model
	Title string `db:"title"`
}

func (touchModel) TableName() string { return "touch_models" }

func TestTouchAndRefresh(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE touch_models (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT,
		created_at DATETIME,
		updated_at DATETIME
	)`)
	if err != nil {
		t.Fatal(err)
	}
	orm.Configure(db, "sqlite")
	m := &touchModel{Title: "a"}
	if err := orm.Save(m); err != nil {
		t.Fatal(err)
	}
	time.Sleep(10 * time.Millisecond)
	if err := orm.Touch(m); err != nil {
		t.Fatal(err)
	}
	m.Title = "stale"
	if err := orm.Refresh(m); err != nil {
		t.Fatal(err)
	}
	if m.Title != "a" {
		t.Fatalf("refresh title=%q", m.Title)
	}
}
