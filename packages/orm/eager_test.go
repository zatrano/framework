package orm_test

import (
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/zatrano/framework/packages/orm"
)

type codeModel struct {
	Code      string     `db:"code"`
	Title     string     `db:"title"`
	CreatedAt *time.Time `db:"created_at"`
	UpdatedAt *time.Time `db:"updated_at"`
}

func (codeModel) TableName() string  { return "code_models" }
func (codeModel) PrimaryKey() string { return "code" }

func setupCodeDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE code_models (
		code TEXT PRIMARY KEY,
		title TEXT,
		created_at DATETIME,
		updated_at DATETIME
	)`)
	if err != nil {
		t.Fatal(err)
	}
	orm.Configure(db, "sqlite")
	return db
}

func TestCustomPrimaryKey(t *testing.T) {
	db := setupCodeDB(t)
	defer db.Close()

	created, err := orm.Create[codeModel](map[string]any{
		"code":  "ABC",
		"title": "alpha",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Code != "ABC" {
		t.Fatalf("code=%q", created.Code)
	}

	found, err := orm.Find[codeModel]("ABC")
	if err != nil || found.Title != "alpha" {
		t.Fatalf("find=%+v err=%v", found, err)
	}

	created.Title = "beta"
	if err := orm.Save(created); err != nil {
		t.Fatal(err)
	}
	refreshed, err := orm.Find[codeModel]("ABC")
	if err != nil || refreshed.Title != "beta" {
		t.Fatalf("save update=%+v err=%v", refreshed, err)
	}

	if _, err := orm.Destroy[codeModel]("ABC"); err != nil {
		t.Fatal(err)
	}
	if _, err := orm.Find[codeModel]("ABC"); err == nil {
		t.Fatal("expected not found after destroy")
	}
}

type eagerParent struct {
	orm.Model
	Name     string           `db:"name"`
	Children []eagerChildLoad `db:"-"`
}

func (eagerParent) TableName() string { return "eager_parents" }

type eagerChildLoad struct {
	orm.Model
	ParentID int64  `db:"parent_id"`
	Name     string `db:"name"`
}

func (eagerChildLoad) TableName() string { return "eager_child_loads" }

func setupEagerDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	for _, sqlStr := range []string{
		`CREATE TABLE eager_parents (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE eager_child_loads (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			parent_id INTEGER,
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
	return db
}

func TestEagerLoadHasManyAndWith(t *testing.T) {
	db := setupEagerDB(t)
	defer db.Close()

	p1, _ := orm.Create[eagerParent](map[string]any{"name": "p1"})
	p2, _ := orm.Create[eagerParent](map[string]any{"name": "p2"})
	_, _ = orm.Create[eagerChildLoad](map[string]any{"parent_id": p1.ID, "name": "c1"})
	_, _ = orm.Create[eagerChildLoad](map[string]any{"parent_id": p1.ID, "name": "c2"})
	_, _ = orm.Create[eagerChildLoad](map[string]any{"parent_id": p2.ID, "name": "c3"})

	parents, err := orm.Query[eagerParent]().OrderBy("id").Get()
	if err != nil {
		t.Fatal(err)
	}
	if err := orm.LoadHasMany[eagerParent, eagerChildLoad](&parents, "Children", "parent_id"); err != nil {
		t.Fatal(err)
	}
	if len(parents[0].Children) != 2 || len(parents[1].Children) != 1 {
		t.Fatalf("load has many: p1=%d p2=%d", len(parents[0].Children), len(parents[1].Children))
	}

	withParents, err := orm.Query[eagerParent]().
		With(orm.EagerHasMany[eagerParent, eagerChildLoad]("Children", "parent_id")).
		OrderBy("id").
		Get()
	if err != nil {
		t.Fatal(err)
	}
	if len(withParents[0].Children) != 2 {
		t.Fatalf("with eager: %d children", len(withParents[0].Children))
	}

	single, err := orm.Query[eagerParent]().
		Where("id", p1.ID).
		With(orm.EagerHasMany[eagerParent, eagerChildLoad]("Children", "parent_id")).
		First()
	if err != nil || len(single.Children) != 2 {
		t.Fatalf("first with=%+v err=%v", single, err)
	}
}

func TestWhereHasFn(t *testing.T) {
	db := setupEagerDB(t)
	defer db.Close()

	p1, _ := orm.Create[eagerParent](map[string]any{"name": "p1"})
	p2, _ := orm.Create[eagerParent](map[string]any{"name": "p2"})
	_, _ = orm.Create[eagerChildLoad](map[string]any{"parent_id": p1.ID, "name": "active"})
	_, _ = orm.Create[eagerChildLoad](map[string]any{"parent_id": p2.ID, "name": "inactive"})

	filtered, err := orm.WhereHasFn(
		orm.Query[eagerParent](),
		"parent_id",
		func(q *orm.Querier[eagerChildLoad]) {
			q.Where("name", "active")
		},
	).Get()
	if err != nil || len(filtered) != 1 || filtered[0].ID != p1.ID {
		t.Fatalf("whereHasFn=%+v err=%v", filtered, err)
	}
}

type throughMembership struct {
	orm.Model
	ParentID int64 `db:"parent_id"`
	RoleID   int64 `db:"role_id"`
}

func (throughMembership) TableName() string { return "memberships" }

type throughRole struct {
	orm.Model
	Name string `db:"name"`
}

func (throughRole) TableName() string { return "through_roles" }

func setupThroughDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	for _, sqlStr := range []string{
		`CREATE TABLE eager_parents (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE memberships (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			parent_id INTEGER,
			role_id INTEGER,
			created_at DATETIME,
			updated_at DATETIME
		)`,
		`CREATE TABLE through_roles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
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
	return db
}

func TestHasManyThrough(t *testing.T) {
	db := setupThroughDB(t)
	defer db.Close()

	parent, _ := orm.Create[eagerParent](map[string]any{"name": "owner"})
	role1, _ := orm.Create[throughRole](map[string]any{"name": "admin"})
	role2, _ := orm.Create[throughRole](map[string]any{"name": "editor"})
	_, _ = orm.Create[throughMembership](map[string]any{"parent_id": parent.ID, "role_id": role1.ID})
	_, _ = orm.Create[throughMembership](map[string]any{"parent_id": parent.ID, "role_id": role2.ID})

	roles, err := orm.HasManyThrough[eagerParent, throughMembership, throughRole](
		parent, "parent_id", "role_id",
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(roles) != 2 {
		t.Fatalf("through roles=%d", len(roles))
	}

	one, err := orm.HasOneThrough[eagerParent, throughMembership, throughRole](
		parent, "parent_id", "role_id",
	)
	if err != nil || one == nil {
		t.Fatalf("hasOneThrough=%+v err=%v", one, err)
	}
}

func TestReplicateCustomPrimaryKey(t *testing.T) {
	db := setupCodeDB(t)
	defer db.Close()

	created, err := orm.Create[codeModel](map[string]any{"code": "REP", "title": "copy-me"})
	if err != nil {
		t.Fatal(err)
	}
	copy := orm.Replicate(created)
	if copy.Code != "" {
		t.Fatalf("expected empty code, got %q", copy.Code)
	}
	if copy.Title != "copy-me" {
		t.Fatalf("title=%q", copy.Title)
	}
}

func TestWithCountBatchAndToggle(t *testing.T) {
	db := setupORMDB(t)
	defer db.Close()

	p1, _ := orm.Create[parentModel](map[string]any{"name": "wc1"})
	p2, _ := orm.Create[parentModel](map[string]any{"name": "wc2"})
	_, _ = orm.Create[childModel](map[string]any{"parent_id": p1.ID, "name": "a"})
	_, _ = orm.Create[childModel](map[string]any{"parent_id": p1.ID, "name": "b"})

	counts, err := orm.WithCount[parentModel, childModel]([]parentModel{*p1, *p2}, "parent_id")
	if err != nil {
		t.Fatal(err)
	}
	if counts[p1.ID] != 2 || counts[p2.ID] != 0 {
		t.Fatalf("counts=%v", counts)
	}

	if err := orm.Attach(p1, "parent_tag", "parent_id", "tag_id", []any{int64(1), int64(2)}); err != nil {
		t.Fatal(err)
	}
	if err := orm.Toggle(p1, "parent_tag", "parent_id", "tag_id", []any{int64(2), int64(3)}); err != nil {
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
	if len(tags) != 2 || tags[0] != 1 || tags[1] != 3 {
		t.Fatalf("toggle tags=%v", tags)
	}
}

func TestMorphRegister(t *testing.T) {
	db := setupCodeDB(t)
	defer db.Close()

	created, _ := orm.Create[codeModel](map[string]any{"code": "M1", "title": "morph"})
	orm.RegisterMorph("codes", func(id any) (any, error) {
		return orm.Find[codeModel](id)
	})

	type morphChild struct {
		Type string `db:"type"`
		ID   string `db:"id"`
	}
	child := &morphChild{Type: "codes", ID: created.Code}
	got, err := orm.MorphTo(child, "type", "id")
	if err != nil {
		t.Fatal(err)
	}
	m, ok := got.(*codeModel)
	if !ok || m.Title != "morph" {
		t.Fatalf("morph=%T %+v", got, got)
	}
}
