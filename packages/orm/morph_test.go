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

type morphTag struct {
	orm.Model
	Name  string      `db:"name"`
	Posts []morphPost `db:"-"`
}

func (morphTag) TableName() string { return "morph_tags" }

type morphPostWithTags struct {
	orm.Model
	Title string     `db:"title"`
	Tags  []morphTag `db:"-"`
}

func (morphPostWithTags) TableName() string { return "morph_posts" }

func TestMorphToManyAttachAndEager(t *testing.T) {
	db := setupMorphDB(t)
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE taggables (
		tag_id INTEGER,
		taggable_id INTEGER,
		taggable_type TEXT
	)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE morph_tags (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		created_at DATETIME,
		updated_at DATETIME
	)`); err != nil {
		t.Fatal(err)
	}

	post, _ := orm.Create[morphPostWithTags](map[string]any{"title": "tagged"})
	tag1, _ := orm.Create[morphTag](map[string]any{"name": "go"})
	tag2, _ := orm.Create[morphTag](map[string]any{"name": "orm"})

	if err := orm.AttachMorph(post, "taggables", "tag_id", "taggable_type", "taggable_id", "morph_posts", []any{tag1.ID, tag2.ID}); err != nil {
		t.Fatal(err)
	}

	tags, err := orm.MorphToMany[morphPostWithTags, morphTag](post, "taggables", "tag_id", "taggable_type", "taggable_id", "morph_posts")
	if err != nil || len(tags) != 2 {
		t.Fatalf("morphToMany=%d err=%v", len(tags), err)
	}

	posts, err := orm.MorphedByMany[morphTag, morphPostWithTags](tag1, "taggables", "tag_id", "taggable_type", "taggable_id", "morph_posts")
	if err != nil || len(posts) != 1 || posts[0].ID != post.ID {
		t.Fatalf("morphedByMany=%+v err=%v", posts, err)
	}

	loaded, err := orm.Query[morphPostWithTags]().
		With(orm.EagerMorphToMany[morphPostWithTags, morphTag]("Tags", "taggables", "tag_id", "taggable_type", "taggable_id", "morph_posts")).
		Get()
	if err != nil || len(loaded) != 1 || len(loaded[0].Tags) != 2 {
		t.Fatalf("eager morphToMany=%+v err=%v", loaded, err)
	}

	if err := orm.SyncMorph(post, "taggables", "tag_id", "taggable_type", "taggable_id", "morph_posts", []any{tag1.ID}); err != nil {
		t.Fatal(err)
	}
	tags, err = orm.MorphToMany[morphPostWithTags, morphTag](post, "taggables", "tag_id", "taggable_type", "taggable_id", "morph_posts")
	if err != nil || len(tags) != 1 || tags[0].ID != tag1.ID {
		t.Fatalf("sync morph=%+v err=%v", tags, err)
	}
}

func TestWhereHasMorph(t *testing.T) {
	db := setupMorphDB(t)
	defer db.Close()

	p1, _ := orm.Create[morphPost](map[string]any{"title": "with"})
	p2, _ := orm.Create[morphPost](map[string]any{"title": "without"})
	_, _ = orm.Create[morphComment](map[string]any{
		"body": "x", "commentable_type": "morph_posts", "commentable_id": p1.ID,
	})

	found, err := orm.WhereHasMorph[morphPost, morphComment](
		orm.Query[morphPost](), "commentable_type", "commentable_id", "morph_posts",
	).Get()
	if err != nil || len(found) != 1 || found[0].ID != p1.ID {
		t.Fatalf("whereHasMorph=%+v err=%v", found, err)
	}

	none, err := orm.WhereDoesntHaveMorph[morphPost, morphComment](
		orm.Query[morphPost](), "commentable_type", "commentable_id", "morph_posts",
	).Get()
	if err != nil || len(none) != 1 || none[0].ID != p2.ID {
		t.Fatalf("whereDoesntHaveMorph=%+v err=%v", none, err)
	}
}
