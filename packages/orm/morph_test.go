package orm_test

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/zatrano/framework/packages/orm"
)

type morphPost struct {
	orm.Model
	Title string `db:"title"`
}

func (morphPost) TableName() string { return "morph_posts" }

type morphComment struct {
	orm.Model
	Body            string `db:"body"`
	CommentableType string `db:"commentable_type"`
	CommentableID   int64  `db:"commentable_id"`
}

func (morphComment) TableName() string { return "morph_comments" }

func setupMorphDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	for _, sqlStr := range []string{
		`CREATE TABLE morph_posts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE morph_comments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			body TEXT,
			commentable_type TEXT,
			commentable_id INTEGER,
			created_at DATETIME,
			updated_at DATETIME
		)`,
	} {
		if _, err := db.Exec(sqlStr); err != nil {
			t.Fatal(err)
		}
	}
	orm.Configure(db, "sqlite")
	return db
}

func TestMorphManyAndMorphTo(t *testing.T) {
	db := setupMorphDB(t)
	defer db.Close()

	post, _ := orm.Create[morphPost](map[string]any{"title": "hello"})
	_, _ = orm.Create[morphComment](map[string]any{
		"body":             "nice",
		"commentable_type": "morph_posts",
		"commentable_id":   post.ID,
	})
	_, _ = orm.Create[morphComment](map[string]any{
		"body":             "great",
		"commentable_type": "morph_posts",
		"commentable_id":   post.ID,
	})

	comments, err := orm.MorphMany[morphPost, morphComment](post, "commentable_type", "commentable_id", "morph_posts")
	if err != nil || len(comments) != 2 {
		t.Fatalf("morphMany=%d err=%v", len(comments), err)
	}

	one, err := orm.MorphOne[morphPost, morphComment](post, "commentable_type", "commentable_id", "morph_posts")
	if err != nil || one == nil {
		t.Fatalf("morphOne=%+v err=%v", one, err)
	}

	parent, err := orm.MorphToByTable[morphComment, morphPost](&comments[0], "commentable_type", "commentable_id", "morph_posts")
	if err != nil || parent == nil || parent.ID != post.ID {
		t.Fatalf("morphTo=%+v err=%v", parent, err)
	}

	wrong, err := orm.MorphToByTable[morphComment, morphPost](&comments[0], "commentable_type", "commentable_id", "other")
	if err != nil || wrong != nil {
		t.Fatalf("morphTo wrong type=%+v err=%v", wrong, err)
	}
}
