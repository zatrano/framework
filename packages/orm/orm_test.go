package orm_test

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/zatrano/framework/packages/database/query"
	"github.com/zatrano/framework/packages/events"
	"github.com/zatrano/framework/packages/orm"
)

type softModel struct {
	orm.Model
	orm.SoftDeletes
	Title string `db:"title"`
	Hits  int64  `db:"hits"`
}

func (softModel) TableName() string { return "soft_models" }

type fillModel struct {
	orm.Model
	Title  string `db:"title"`
	Secret string `db:"secret"`
}

func (fillModel) TableName() string { return "fill_models" }

func (fillModel) Fillable() []string { return []string{"title"} }

type childModel struct {
	orm.Model
	ParentID int64  `db:"parent_id"`
	Name     string `db:"name"`
}

func (childModel) TableName() string { return "child_models" }

type parentModel struct {
	orm.Model
	Name string `db:"name"`
}

func (parentModel) TableName() string { return "parent_models" }

func setupORMDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	statements := []string{
		`CREATE TABLE soft_models (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT,
			hits INTEGER DEFAULT 0,
			created_at DATETIME,
			updated_at DATETIME,
			deleted_at DATETIME
		)`,
		`CREATE TABLE fill_models (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT,
			secret TEXT,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE parent_models (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE child_models (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			parent_id INTEGER,
			name TEXT,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE parent_tag (
			parent_id INTEGER,
			tag_id INTEGER
		)`,
	}
	for _, sqlStr := range statements {
		if _, err := db.Exec(sqlStr); err != nil {
			t.Fatal(err)
		}
	}
	orm.Configure(db, "sqlite")
	return db
}

func TestSoftDeleteRestoreAndPrune(t *testing.T) {
	db := setupORMDB(t)
	defer db.Close()

	m, err := orm.Create[softModel](map[string]any{"title": "gone", "hits": int64(2)})
	if err != nil {
		t.Fatal(err)
	}
	if orm.Trashed(m) {
		t.Fatal("should not be trashed")
	}
	if _, err := orm.SoftDelete[softModel](m.ID); err != nil {
		t.Fatal(err)
	}
	found, err := orm.Find[softModel](m.ID)
	if err == nil && found != nil {
		t.Fatal("soft deleted row should be hidden")
	}
	trashed, err := orm.Query[softModel]().OnlyTrashed().Where("id", m.ID).First()
	if err != nil || trashed == nil || !orm.Trashed(trashed) {
		t.Fatalf("only trashed=%+v err=%v", trashed, err)
	}
	if _, err := orm.RestoreByID[softModel](m.ID); err != nil {
		t.Fatal(err)
	}
	restored, err := orm.Find[softModel](m.ID)
	if err != nil || restored == nil || orm.Trashed(restored) {
		t.Fatalf("restored=%+v err=%v", restored, err)
	}

	old := time.Now().Add(-48 * time.Hour)
	_, _ = db.Exec(`INSERT INTO soft_models (title, hits, created_at, updated_at, deleted_at) VALUES ('old', 1, ?, ?, ?)`, old, old, old)
	n, err := orm.Prune[softModel](24 * time.Hour)
	if err != nil || n < 1 {
		t.Fatalf("prune n=%d err=%v", n, err)
	}
}

func TestAggregatesDistinctOldestAndChunkById(t *testing.T) {
	db := setupORMDB(t)
	defer db.Close()

	_, _ = orm.InsertMany[softModel]([]map[string]any{
		{"title": "a", "hits": int64(1)},
		{"title": "b", "hits": int64(3)},
		{"title": "c", "hits": int64(5)},
	})

	sum, err := orm.Query[softModel]().Sum("hits")
	if err != nil || sum != 9 {
		t.Fatalf("sum=%v err=%v", sum, err)
	}
	avg, err := orm.Query[softModel]().Avg("hits")
	if err != nil || avg != 3 {
		t.Fatalf("avg=%v err=%v", avg, err)
	}
	minV, err := orm.Query[softModel]().Min("hits")
	if err != nil || fmtInt(minV) != 1 {
		t.Fatalf("min=%v err=%v", minV, err)
	}
	maxV, err := orm.Query[softModel]().Max("hits")
	if err != nil || fmtInt(maxV) != 5 {
		t.Fatalf("max=%v err=%v", maxV, err)
	}

	cloned := orm.Query[softModel]().Where("hits", ">", 1).Clone().Oldest("hits").Take(1)
	first, err := cloned.First()
	if err != nil || first.Title != "b" {
		t.Fatalf("oldest take=%+v err=%v", first, err)
	}

	var ids []int64
	err = orm.Query[softModel]().ChunkById(2, func(batch []softModel) error {
		for _, row := range batch {
			ids = append(ids, row.ID)
		}
		return nil
	})
	if err != nil || len(ids) != 3 {
		t.Fatalf("chunkById ids=%v err=%v", ids, err)
	}
}

func TestFillableMassAssignment(t *testing.T) {
	db := setupORMDB(t)
	defer db.Close()

	m, err := orm.Create[fillModel](map[string]any{
		"title":  "ok",
		"secret": "nope",
	})
	if err != nil {
		t.Fatal(err)
	}
	if m.Title != "ok" {
		t.Fatalf("title=%q", m.Title)
	}
	if m.Secret != "" {
		t.Fatalf("secret should be blocked, got %q", m.Secret)
	}
}

func TestWhereHasAndPivotSync(t *testing.T) {
	db := setupORMDB(t)
	defer db.Close()

	p1, _ := orm.Create[parentModel](map[string]any{"name": "p1"})
	p2, _ := orm.Create[parentModel](map[string]any{"name": "p2"})
	_, _ = orm.Create[childModel](map[string]any{"parent_id": p1.ID, "name": "c1"})

	withKids, err := orm.WhereHas[parentModel, childModel](orm.Query[parentModel](), "parent_id").Get()
	if err != nil || len(withKids) != 1 || withKids[0].ID != p1.ID {
		t.Fatalf("whereHas=%+v err=%v", withKids, err)
	}
	without, err := orm.DoesntHave[parentModel, childModel]("parent_id").Get()
	if err != nil || len(without) != 1 || without[0].ID != p2.ID {
		t.Fatalf("doesntHave=%+v err=%v", without, err)
	}

	if err := orm.Attach(p1, "parent_tag", "parent_id", "tag_id", []any{int64(10), int64(20)}); err != nil {
		t.Fatal(err)
	}
	if err := orm.Sync(p1, "parent_tag", "parent_id", "tag_id", []any{int64(20), int64(30)}); err != nil {
		t.Fatal(err)
	}
	rows, err := db.Query(`SELECT tag_id FROM parent_tag WHERE parent_id = ? ORDER BY tag_id`, p1.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var tags []int64
	for rows.Next() {
		var id int64
		_ = rows.Scan(&id)
		tags = append(tags, id)
	}
	if len(tags) != 2 || tags[0] != 20 || tags[1] != 30 {
		t.Fatalf("tags=%v", tags)
	}
}

func TestTransactionRollback(t *testing.T) {
	db := setupORMDB(t)
	defer db.Close()

	err := orm.Transaction(func(tx *sql.Tx) error {
		_, err := tx.Exec(`INSERT INTO soft_models (title, hits, created_at, updated_at) VALUES ('tx', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`)
		if err != nil {
			return err
		}
		return sql.ErrTxDone
	})
	if err == nil {
		t.Fatal("expected error")
	}
	n, _ := orm.Query[softModel]().Where("title", "tx").Count()
	if n != 0 {
		t.Fatalf("expected rollback count=%d", n)
	}
}

func TestModelEventsAndDirty(t *testing.T) {
	db := setupORMDB(t)
	defer db.Close()

	d := events.New()
	var created bool
	d.Listen("softmodel.created", func(event any) error {
		created = true
		return nil
	})
	orm.SetDispatcher(d)

	m, err := orm.Create[softModel](map[string]any{"title": "evt", "hits": int64(1)})
	if err != nil || !created {
		t.Fatalf("created event=%v err=%v", created, err)
	}
	if orm.IsDirty(m) {
		t.Fatal("fresh model should not be dirty")
	}
	m.Title = "changed"
	if !orm.IsDirty(m, "title") {
		t.Fatal("expected dirty title")
	}
	copy := orm.Replicate(m)
	if copy.ID != 0 || copy.Title != "changed" {
		t.Fatalf("replicate=%+v", copy)
	}
	fresh, err := orm.Fresh(m)
	if err != nil || fresh.Title != "evt" {
		t.Fatalf("fresh=%+v err=%v", fresh, err)
	}

	m.Title = "saved"
	if err := orm.Save(m); err != nil {
		t.Fatal(err)
	}
	if !orm.WasChanged(m, "title") {
		t.Fatalf("expected WasChanged title, changes=%v", orm.GetChanges(m))
	}
	if orm.IsDirty(m) {
		t.Fatal("after save should not be dirty")
	}
}

func TestQueryTxParticipates(t *testing.T) {
	db := setupORMDB(t)
	defer db.Close()

	err := orm.Transaction(func(tx *sql.Tx) error {
		now := time.Now()
		if _, err := query.New(tx, "sqlite", "soft_models").Insert(map[string]any{
			"title":      "tx-model",
			"hits":       int64(1),
			"created_at": now,
			"updated_at": now,
		}); err != nil {
			return err
		}
		n, err := orm.QueryTx[softModel](tx).Where("title", "tx-model").Count()
		if err != nil {
			return err
		}
		if n != 1 {
			return fmt.Errorf("tx count=%d", n)
		}
		return fmt.Errorf("force rollback")
	})
	if err == nil {
		t.Fatal("expected rollback error")
	}
	n, _ := orm.Query[softModel]().Where("title", "tx-model").Count()
	if n != 0 {
		t.Fatalf("tx row leaked count=%d", n)
	}
}

func fmtInt(v any) int64 {
	switch n := v.(type) {
	case int64:
		return n
	case int:
		return int64(n)
	case float64:
		return int64(n)
	default:
		return 0
	}
}
