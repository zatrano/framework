package orm_test

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/zatrano/framework/packages/orm"
)

type persistModel struct {
	orm.Model
	Title string `db:"title"`
	Hits  int64  `db:"hits"`
}

func (persistModel) TableName() string { return "persist_models" }

func setupPersistDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE persist_models (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT,
		hits INTEGER DEFAULT 0,
		created_at DATETIME,
		updated_at DATETIME
	)`)
	if err != nil {
		t.Fatal(err)
	}
	orm.Configure(db, "sqlite")
	return db
}

func TestFirstOrCreateAndUpdateOrCreate(t *testing.T) {
	db := setupPersistDB(t)
	defer db.Close()

	first, created, err := orm.FirstOrCreate[persistModel](
		map[string]any{"title": "alpha"},
		map[string]any{"hits": int64(1)},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !created || first == nil || first.Title != "alpha" || first.Hits != 1 {
		t.Fatalf("first create: created=%v model=%+v", created, first)
	}

	again, created, err := orm.FirstOrCreate[persistModel](map[string]any{"title": "alpha"})
	if err != nil {
		t.Fatal(err)
	}
	if created || again.ID != first.ID {
		t.Fatalf("expected existing row, created=%v id=%d", created, again.ID)
	}

	draft, exists, err := orm.FirstOrNew[persistModel](
		map[string]any{"title": "brand-new"},
		map[string]any{"hits": int64(3)},
	)
	if err != nil || exists || draft.ID != 0 || draft.Title != "brand-new" || draft.Hits != 3 {
		t.Fatalf("first or new: exists=%v model=%+v err=%v", exists, draft, err)
	}
	existing, exists, err := orm.FirstOrNew[persistModel](map[string]any{"title": "alpha"})
	if err != nil || !exists || existing.ID != first.ID {
		t.Fatalf("first or new existing: exists=%v id=%d err=%v", exists, existing.ID, err)
	}

	titles, err := orm.Query[persistModel]().OrderBy("id").Pluck("title")
	if err != nil || len(titles) < 1 {
		t.Fatalf("pluck=%v err=%v", titles, err)
	}
	val, err := orm.Query[persistModel]().Where("title", "alpha").Value("hits")
	if err != nil {
		t.Fatal(err)
	}
	liked, err := orm.Query[persistModel]().WhereLike("title", "alp%").Get()
	if err != nil || len(liked) != 1 {
		t.Fatalf("like=%d err=%v", len(liked), err)
	}
	_ = val

	updated, created, err := orm.UpdateOrCreate[persistModel](
		map[string]any{"title": "alpha"},
		map[string]any{"hits": int64(9)},
	)
	if err != nil {
		t.Fatal(err)
	}
	if created || updated.Hits != 9 {
		t.Fatalf("update or create: created=%v hits=%d", created, updated.Hits)
	}

	fresh, created, err := orm.UpdateOrCreate[persistModel](
		map[string]any{"title": "beta"},
		map[string]any{"hits": int64(3)},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !created || fresh.Title != "beta" || fresh.Hits != 3 {
		t.Fatalf("create via update or create: %+v created=%v", fresh, created)
	}
}

func TestIncrementDecrementChunkSimplePaginate(t *testing.T) {
	db := setupPersistDB(t)
	defer db.Close()

	for i := 0; i < 5; i++ {
		if _, err := orm.Create[persistModel](map[string]any{
			"title": "item",
			"hits":  int64(i),
		}); err != nil {
			t.Fatal(err)
		}
	}

	affected, err := orm.Query[persistModel]().Where("title", "item").Where("hits", 0).Increment("hits", 2)
	if err != nil || affected != 1 {
		t.Fatalf("increment: affected=%d err=%v", affected, err)
	}
	row, err := orm.Query[persistModel]().Where("hits", 2).First()
	if err != nil || row == nil {
		t.Fatalf("after increment: %v", err)
	}
	affected, err = orm.Query[persistModel]().Where("id", row.ID).Decrement("hits")
	if err != nil || affected != 1 {
		t.Fatalf("decrement: affected=%d err=%v", affected, err)
	}

	batches := 0
	total := 0
	err = orm.Query[persistModel]().Where("title", "item").Chunk(2, func(items []persistModel) error {
		batches++
		total += len(items)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if batches != 3 || total != 5 {
		t.Fatalf("chunk batches=%d total=%d", batches, total)
	}

	page, err := orm.Query[persistModel]().Where("title", "item").OrderBy("id").SimplePaginate(1, 2, "/demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Data) != 2 || !page.HasMore || page.NextPageURL == "" {
		t.Fatalf("simple page 1: %+v", page)
	}
	page2, err := orm.Query[persistModel]().Where("title", "item").OrderBy("id").SimplePaginate(3, 2, "/demo")
	if err != nil {
		t.Fatal(err)
	}
	if len(page2.Data) != 1 || page2.HasMore {
		t.Fatalf("simple page 3: %+v", page2)
	}
}
