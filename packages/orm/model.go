package orm

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/zatrano/framework/packages/database/query"
	"github.com/zatrano/framework/packages/pagination"
)

// DB is the active database connection used by ORM.
var DB *sql.DB

// Driver is the active database driver name.
var Driver string

// Model is the base model embedded into application models.
type Model struct {
	ID        int64      `db:"id" json:"id"`
	CreatedAt *time.Time `db:"created_at" json:"created_at,omitempty"`
	UpdatedAt *time.Time `db:"updated_at" json:"updated_at,omitempty"`
}

// SoftDeletes adds soft delete support.
type SoftDeletes struct {
	DeletedAt *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
}

// Querier builds model-aware queries.
type Querier[T any] struct {
	builder          *query.Builder
	table            string
	softDelete       bool
	softApplied      bool
	skipGlobalScopes bool
	globalsApplied   bool
	removedScopes    map[string]bool
	loaders          []func([]T) error
}

// Configure sets the global database connection for ORM.
func Configure(db *sql.DB, driver string) {
	DB = db
	Driver = driver
}

// Table resolves the table name for a model type.
func Table[T any]() string {
	var zero T
	rv := reflect.ValueOf(&zero).Elem()
	rt := rv.Type()

	// Prefer TableName() string method on value or pointer.
	if m := rv.MethodByName("TableName"); m.IsValid() {
		results := m.Call(nil)
		if len(results) == 1 {
			return results[0].String()
		}
	}
	ptr := reflect.New(rt)
	if m := ptr.MethodByName("TableName"); m.IsValid() {
		results := m.Call(nil)
		if len(results) == 1 {
			return results[0].String()
		}
	}

	name := rt.Name()
	return toSnake(pluralize(name))
}

// Query starts a new query for model T (honours Connection() on the model).
func Query[T any]() *Querier[T] {
	table := Table[T]()
	db, driver := dbAndDriver[T]()
	return &Querier[T]{
		builder:    query.New(db, driver, table),
		table:      table,
		softDelete: hasSoftDeletes[T](),
	}
}

// Where adds a where clause.
func Where[T any](column string, args ...any) *Querier[T] {
	return Query[T]().Where(column, args...)
}

// Find finds a model by primary key.
func Find[T any](id any) (*T, error) {
	return Query[T]().Where(KeyName[T](), id).First()
}

// FindOrFail finds a model by primary key or returns an error.
func FindOrFail[T any](id any) (*T, error) {
	model, err := Find[T](id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no query results for model [%s]", Table[T]())
		}
		return nil, err
	}
	return model, nil
}

// All returns all models.
func All[T any]() ([]T, error) {
	return Query[T]().Get()
}

// Create inserts a model from attributes and returns it.
func Create[T any](attrs map[string]any) (*T, error) {
	attrs = filterMassAssignment[T](attrs)
	now := time.Now()
	if _, ok := attrs["created_at"]; !ok {
		attrs["created_at"] = now
	}
	if _, ok := attrs["updated_at"]; !ok {
		attrs["updated_at"] = now
	}

	draft := attrsToModel[T](attrs)
	if err := dispatchModel("creating", draft); err != nil {
		return nil, err
	}

	id, err := tableQuery[T]().Insert(attrs)
	if err != nil {
		return nil, err
	}
	var model *T
	keyName := KeyName[T]()
	if id > 0 && keyName == "id" {
		model, err = Find[T](id)
	} else if keyVal, ok := attrs[keyName]; ok {
		model, err = Find[T](keyVal)
	} else {
		// Fallback for drivers without LastInsertId support.
		q := Query[T]()
		for column, value := range attrs {
			q.Where(column, value)
		}
		model, err = q.Latest(keyName).First()
	}
	if err != nil {
		return nil, err
	}
	SyncOriginal(model)
	recordChanges(model, nil)
	_ = dispatchModel("created", model)
	return model, nil
}

// Where adds a where clause.
func (q *Querier[T]) Where(column string, args ...any) *Querier[T] {
	q.builder.Where(column, args...)
	return q
}

// OrWhere adds an OR where clause.
func (q *Querier[T]) OrWhere(column string, args ...any) *Querier[T] {
	q.builder.OrWhere(column, args...)
	return q
}

// WhereAny applies the same comparison across columns with OR.
func (q *Querier[T]) WhereAny(columns []string, args ...any) *Querier[T] {
	q.builder.WhereAny(columns, args...)
	return q
}

// WhereAll applies the same comparison across columns with AND.
func (q *Querier[T]) WhereAll(columns []string, args ...any) *Querier[T] {
	q.builder.WhereAll(columns, args...)
	return q
}

// OrWhereAny is WhereAny with an outer OR.
func (q *Querier[T]) OrWhereAny(columns []string, args ...any) *Querier[T] {
	q.builder.OrWhereAny(columns, args...)
	return q
}

// OrWhereAll is WhereAll with an outer OR.
func (q *Querier[T]) OrWhereAll(columns []string, args ...any) *Querier[T] {
	q.builder.OrWhereAll(columns, args...)
	return q
}

// WhereIn adds a WHERE IN clause.
func (q *Querier[T]) WhereIn(column string, values []any) *Querier[T] {
	q.builder.WhereIn(column, values)
	return q
}

// OrWhereIn adds an OR WHERE IN clause.
func (q *Querier[T]) OrWhereIn(column string, values []any) *Querier[T] {
	q.builder.OrWhereIn(column, values)
	return q
}

// WhereNotIn adds a WHERE NOT IN clause.
func (q *Querier[T]) WhereNotIn(column string, values []any) *Querier[T] {
	q.builder.WhereNotIn(column, values)
	return q
}

// OrWhereNotIn adds an OR WHERE NOT IN clause.
func (q *Querier[T]) OrWhereNotIn(column string, values []any) *Querier[T] {
	q.builder.OrWhereNotIn(column, values)
	return q
}

// WhereBetween adds a WHERE BETWEEN clause.
func (q *Querier[T]) WhereBetween(column string, min, max any) *Querier[T] {
	q.builder.WhereBetween(column, min, max)
	return q
}

// OrWhereBetween adds an OR WHERE BETWEEN clause.
func (q *Querier[T]) OrWhereBetween(column string, min, max any) *Querier[T] {
	q.builder.OrWhereBetween(column, min, max)
	return q
}

// WhereNotBetween adds a WHERE NOT BETWEEN clause.
func (q *Querier[T]) WhereNotBetween(column string, min, max any) *Querier[T] {
	q.builder.WhereNotBetween(column, min, max)
	return q
}

// OrWhereNotBetween adds an OR WHERE NOT BETWEEN clause.
func (q *Querier[T]) OrWhereNotBetween(column string, min, max any) *Querier[T] {
	q.builder.OrWhereNotBetween(column, min, max)
	return q
}

// WhereDate compares DATE(column) to value.
func (q *Querier[T]) WhereDate(column string, value any) *Querier[T] {
	q.builder.WhereDate(column, value)
	return q
}

// WhereMonth compares the month of column.
func (q *Querier[T]) WhereMonth(column string, month any) *Querier[T] {
	q.builder.WhereMonth(column, month)
	return q
}

// WhereYear compares the year of column.
func (q *Querier[T]) WhereYear(column string, year any) *Querier[T] {
	q.builder.WhereYear(column, year)
	return q
}

// WhereDay compares the day-of-month of column.
func (q *Querier[T]) WhereDay(column string, day any) *Querier[T] {
	q.builder.WhereDay(column, day)
	return q
}

// WhereTime compares the time-of-day of column.
func (q *Querier[T]) WhereTime(column string, value any) *Querier[T] {
	q.builder.WhereTime(column, value)
	return q
}

// WhereDayOfWeek compares weekday of column (0=Sunday … 6=Saturday).
func (q *Querier[T]) WhereDayOfWeek(column string, day any) *Querier[T] {
	q.builder.WhereDayOfWeek(column, day)
	return q
}

// WhereHour compares the hour of column (0-23).
func (q *Querier[T]) WhereHour(column string, hour any) *Querier[T] {
	q.builder.WhereHour(column, hour)
	return q
}

// WhereNull adds WHERE IS NULL.
func (q *Querier[T]) WhereNull(column string) *Querier[T] {
	q.builder.WhereNull(column)
	return q
}

// WhereNotNull adds WHERE IS NOT NULL.
func (q *Querier[T]) WhereNotNull(column string) *Querier[T] {
	q.builder.WhereNotNull(column)
	return q
}

// OrWhereNull adds OR WHERE IS NULL.
func (q *Querier[T]) OrWhereNull(column string) *Querier[T] {
	q.builder.OrWhereNull(column)
	return q
}

// OrWhereNotNull adds OR WHERE IS NOT NULL.
func (q *Querier[T]) OrWhereNotNull(column string) *Querier[T] {
	q.builder.OrWhereNotNull(column)
	return q
}

// WhereColumn compares two columns.
func (q *Querier[T]) WhereColumn(first string, parts ...string) *Querier[T] {
	q.builder.WhereColumn(first, parts...)
	return q
}

// OrWhereColumn compares two columns with OR.
func (q *Querier[T]) OrWhereColumn(first string, parts ...string) *Querier[T] {
	q.builder.OrWhereColumn(first, parts...)
	return q
}

// WhereRaw adds a raw WHERE fragment.
func (q *Querier[T]) WhereRaw(sqlStr string, bindings ...any) *Querier[T] {
	q.builder.WhereRaw(sqlStr, bindings...)
	return q
}

// OrWhereRaw adds a raw OR WHERE fragment.
func (q *Querier[T]) OrWhereRaw(sqlStr string, bindings ...any) *Querier[T] {
	q.builder.OrWhereRaw(sqlStr, bindings...)
	return q
}

// WhereExists adds a WHERE EXISTS subquery.
func (q *Querier[T]) WhereExists(sub *query.Builder) *Querier[T] {
	q.builder.WhereExists(sub)
	return q
}

// WhereNotExists adds a WHERE NOT EXISTS subquery.
func (q *Querier[T]) WhereNotExists(sub *query.Builder) *Querier[T] {
	q.builder.WhereNotExists(sub)
	return q
}

// OrWhereExists adds an OR EXISTS subquery.
func (q *Querier[T]) OrWhereExists(sub *query.Builder) *Querier[T] {
	q.builder.OrWhereExists(sub)
	return q
}

// OrWhereNotExists adds an OR NOT EXISTS subquery.
func (q *Querier[T]) OrWhereNotExists(sub *query.Builder) *Querier[T] {
	q.builder.OrWhereNotExists(sub)
	return q
}

// WhereLike adds a WHERE LIKE clause.
func (q *Querier[T]) WhereLike(column string, pattern string) *Querier[T] {
	q.builder.WhereLike(column, pattern)
	return q
}

// WhereNotLike adds a WHERE NOT LIKE clause.
func (q *Querier[T]) WhereNotLike(column string, pattern string) *Querier[T] {
	q.builder.WhereNotLike(column, pattern)
	return q
}

// OrWhereLike adds an OR WHERE LIKE clause.
func (q *Querier[T]) OrWhereLike(column string, pattern string) *Querier[T] {
	q.builder.OrWhereLike(column, pattern)
	return q
}

// OrWhereNotLike adds an OR WHERE NOT LIKE clause.
func (q *Querier[T]) OrWhereNotLike(column string, pattern string) *Querier[T] {
	q.builder.OrWhereNotLike(column, pattern)
	return q
}

// Value returns a single column value from the first matching row.
func (q *Querier[T]) Value(column string) (any, error) {
	q.prepare()
	return q.builder.Value(column)
}

// Pluck returns a slice of values for the given column.
func (q *Querier[T]) Pluck(column string) ([]any, error) {
	q.prepare()
	return q.builder.Pluck(column)
}

// OrderBy adds ordering.
func (q *Querier[T]) OrderBy(column string, direction ...string) *Querier[T] {
	q.builder.OrderBy(column, direction...)
	return q
}

// OrderByRaw adds a raw ORDER BY expression.
func (q *Querier[T]) OrderByRaw(sqlStr string, bindings ...any) *Querier[T] {
	q.builder.OrderByRaw(sqlStr, bindings...)
	return q
}

// OrderByDesc orders by column descending.
func (q *Querier[T]) OrderByDesc(column string) *Querier[T] {
	q.builder.OrderByDesc(column)
	return q
}

// Reorder clears order clauses, then optionally adds a new OrderBy.
func (q *Querier[T]) Reorder(column ...string) *Querier[T] {
	q.builder.Reorder(column...)
	return q
}

// Select sets the columns to select.
func (q *Querier[T]) Select(columns ...string) *Querier[T] {
	q.builder.Select(columns...)
	return q
}

// AddSelect appends columns to the current select list.
func (q *Querier[T]) AddSelect(columns ...string) *Querier[T] {
	q.builder.AddSelect(columns...)
	return q
}

// SelectRaw appends a raw SELECT expression.
func (q *Querier[T]) SelectRaw(expression string, bindings ...any) *Querier[T] {
	q.builder.SelectRaw(expression, bindings...)
	return q
}

// GroupBy adds GROUP BY columns.
func (q *Querier[T]) GroupBy(columns ...string) *Querier[T] {
	q.builder.GroupBy(columns...)
	return q
}

// Having adds a HAVING clause.
func (q *Querier[T]) Having(column string, operator string, value any) *Querier[T] {
	q.builder.Having(column, operator, value)
	return q
}

// HavingRaw adds a raw HAVING fragment.
func (q *Querier[T]) HavingRaw(sqlStr string, bindings ...any) *Querier[T] {
	q.builder.HavingRaw(sqlStr, bindings...)
	return q
}

// OrHavingRaw adds a raw OR HAVING fragment.
func (q *Querier[T]) OrHavingRaw(sqlStr string, bindings ...any) *Querier[T] {
	q.builder.OrHavingRaw(sqlStr, bindings...)
	return q
}

// OrHaving adds an OR HAVING clause.
func (q *Querier[T]) OrHaving(column string, operator string, value any) *Querier[T] {
	q.builder.OrHaving(column, operator, value)
	return q
}

// Join adds an inner join.
func (q *Querier[T]) Join(table, first, operator, second string) *Querier[T] {
	q.builder.Join(table, first, operator, second)
	return q
}

// LeftJoin adds a left join.
func (q *Querier[T]) LeftJoin(table, first, operator, second string) *Querier[T] {
	q.builder.LeftJoin(table, first, operator, second)
	return q
}

// RightJoin adds a right join.
func (q *Querier[T]) RightJoin(table, first, operator, second string) *Querier[T] {
	q.builder.RightJoin(table, first, operator, second)
	return q
}

// CrossJoin adds a cross join.
func (q *Querier[T]) CrossJoin(table string) *Querier[T] {
	q.builder.CrossJoin(table)
	return q
}

// Latest orders descending.
func (q *Querier[T]) Latest(column ...string) *Querier[T] {
	q.builder.Latest(column...)
	return q
}

// Oldest orders ascending.
func (q *Querier[T]) Oldest(column ...string) *Querier[T] {
	q.builder.Oldest(column...)
	return q
}

// Distinct selects distinct rows.
func (q *Querier[T]) Distinct() *Querier[T] {
	q.builder.Distinct()
	return q
}

// Take limits the number of results.
func (q *Querier[T]) Take(n int) *Querier[T] {
	q.builder.Take(n)
	return q
}

// Skip offsets the results.
func (q *Querier[T]) Skip(n int) *Querier[T] {
	q.builder.Skip(n)
	return q
}

// Clone returns a copy of the querier.
func (q *Querier[T]) Clone() *Querier[T] {
	if q == nil {
		return Query[T]()
	}
	loaders := make([]func([]T) error, len(q.loaders))
	copy(loaders, q.loaders)
	return &Querier[T]{
		builder:          q.builder.Clone(),
		table:            q.table,
		softDelete:       q.softDelete,
		softApplied:      q.softApplied,
		skipGlobalScopes: q.skipGlobalScopes,
		globalsApplied:   q.globalsApplied,
		removedScopes:    copyBoolMap(q.removedScopes),
		loaders:          loaders,
	}
}

// InRandomOrder orders rows randomly.
func (q *Querier[T]) InRandomOrder() *Querier[T] {
	q.builder.InRandomOrder()
	return q
}

// Builder exposes the underlying query builder.
func (q *Querier[T]) Builder() *query.Builder {
	return q.builder
}

func copyBoolMap(in map[string]bool) map[string]bool {
	if in == nil {
		return nil
	}
	out := make(map[string]bool, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// Limit limits results.
func (q *Querier[T]) Limit(n int) *Querier[T] {
	q.builder.Limit(n)
	return q
}

// Offset offsets results.
func (q *Querier[T]) Offset(n int) *Querier[T] {
	q.builder.Offset(n)
	return q
}

// ForPage applies limit/offset for a 1-based page.
func (q *Querier[T]) ForPage(page, perPage int) *Querier[T] {
	q.builder.ForPage(page, perPage)
	return q
}

// Union appends a UNION subquery builder.
func (q *Querier[T]) Union(sub *query.Builder) *Querier[T] {
	q.builder.Union(sub)
	return q
}

// UnionAll appends a UNION ALL subquery builder.
func (q *Querier[T]) UnionAll(sub *query.Builder) *Querier[T] {
	q.builder.UnionAll(sub)
	return q
}

// LockForUpdate adds a pessimistic write lock.
func (q *Querier[T]) LockForUpdate() *Querier[T] {
	q.builder.LockForUpdate()
	return q
}

// ForUpdate is an alias for LockForUpdate.
func (q *Querier[T]) ForUpdate() *Querier[T] {
	return q.LockForUpdate()
}

// SharedLock adds a shared lock.
func (q *Querier[T]) SharedLock() *Querier[T] {
	q.builder.SharedLock()
	return q
}

// SkipLocked appends SKIP LOCKED to the lock clause.
func (q *Querier[T]) SkipLocked() *Querier[T] {
	q.builder.SkipLocked()
	return q
}

// NoWait appends NOWAIT to the lock clause.
func (q *Querier[T]) NoWait() *Querier[T] {
	q.builder.NoWait()
	return q
}

// WithTrashed includes soft deleted models.
func (q *Querier[T]) WithTrashed() *Querier[T] {
	q.softDelete = false
	return q
}

// OnlyTrashed returns only soft deleted models.
func (q *Querier[T]) OnlyTrashed() *Querier[T] {
	q.softDelete = false
	q.builder.WhereNotNull("deleted_at")
	return q
}

// Get returns all matching models.
func (q *Querier[T]) Get() ([]T, error) {
	q.prepare()
	rows, err := q.builder.Get()
	if err != nil {
		return nil, err
	}
	out := make([]T, 0, len(rows))
	for _, row := range rows {
		model, err := mapToModel[T](row)
		if err != nil {
			return nil, err
		}
		out = append(out, *model)
	}
	if err := q.runLoaders(out); err != nil {
		return nil, err
	}
	return out, nil
}

// First returns the first matching model.
func (q *Querier[T]) First() (*T, error) {
	q.prepare()
	row, err := q.builder.First()
	if err != nil {
		return nil, err
	}
	model, err := mapToModel[T](row)
	if err != nil {
		return nil, err
	}
	slice := []T{*model}
	if err := q.runLoaders(slice); err != nil {
		return nil, err
	}
	return &slice[0], nil
}

// Sole returns the single matching model or an error if none/multiple match.
func (q *Querier[T]) Sole() (*T, error) {
	q.prepare()
	rows, err := q.builder.Limit(2).Get()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, sql.ErrNoRows
	}
	if len(rows) > 1 {
		return nil, fmt.Errorf("multiple records found for sole query on [%s]", q.table)
	}
	model, err := mapToModel[T](rows[0])
	if err != nil {
		return nil, err
	}
	slice := []T{*model}
	if err := q.runLoaders(slice); err != nil {
		return nil, err
	}
	return &slice[0], nil
}

// Count returns matching count.
func (q *Querier[T]) Count() (int64, error) {
	q.prepare()
	return q.builder.Count()
}

// Sum returns the sum of a column.
func (q *Querier[T]) Sum(column string) (float64, error) {
	q.prepare()
	return q.builder.Sum(column)
}

// Avg returns the average of a column.
func (q *Querier[T]) Avg(column string) (float64, error) {
	q.prepare()
	return q.builder.Avg(column)
}

// Min returns the minimum of a column.
func (q *Querier[T]) Min(column string) (any, error) {
	q.prepare()
	return q.builder.Min(column)
}

// Max returns the maximum of a column.
func (q *Querier[T]) Max(column string) (any, error) {
	q.prepare()
	return q.builder.Max(column)
}

// Exists reports whether any match exists.
func (q *Querier[T]) Exists() (bool, error) {
	q.prepare()
	return q.builder.Exists()
}

// Paginate returns a length-aware page of models.
func (q *Querier[T]) Paginate(page, perPage int, path ...string) (*pagination.LengthAware[T], error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 15
	}

	q.prepare()
	total, err := q.builder.Count()
	if err != nil {
		return nil, err
	}

	q.builder.Limit(perPage).Offset((page - 1) * perPage)
	// Get would re-prepare; pull rows directly after prepare already done.
	rows, err := q.builder.Get()
	if err != nil {
		return nil, err
	}
	items := make([]T, 0, len(rows))
	for _, row := range rows {
		model, err := mapToModel[T](row)
		if err != nil {
			return nil, err
		}
		items = append(items, *model)
	}
	if err := q.runLoaders(items); err != nil {
		return nil, err
	}

	basePath := ""
	if len(path) > 0 {
		basePath = path[0]
	}
	return pagination.New(items, total, page, perPage, basePath), nil
}

// SimplePaginate returns a page of models without a total count.
func (q *Querier[T]) SimplePaginate(page, perPage int, path ...string) (*pagination.Simple[T], error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 15
	}
	q.prepare()
	rows, hasMore, err := q.builder.SimplePaginate(page, perPage)
	if err != nil {
		return nil, err
	}
	items := make([]T, 0, len(rows))
	for _, row := range rows {
		model, err := mapToModel[T](row)
		if err != nil {
			return nil, err
		}
		items = append(items, *model)
	}
	if err := q.runLoaders(items); err != nil {
		return nil, err
	}
	basePath := ""
	if len(path) > 0 {
		basePath = path[0]
	}
	return pagination.NewSimple(items, page, perPage, basePath, hasMore), nil
}

// Increment increments a numeric column on matching rows.
func (q *Querier[T]) Increment(column string, amount ...int64) (int64, error) {
	q.prepare()
	return q.builder.Increment(column, amount...)
}

// Decrement decrements a numeric column on matching rows.
func (q *Querier[T]) Decrement(column string, amount ...int64) (int64, error) {
	q.prepare()
	return q.builder.Decrement(column, amount...)
}

// Chunk iterates matching models in batches of size.
func (q *Querier[T]) Chunk(size int, callback func([]T) error) error {
	q.prepare()
	return q.builder.Chunk(size, func(rows []map[string]any) error {
		items := make([]T, 0, len(rows))
		for _, row := range rows {
			model, err := mapToModel[T](row)
			if err != nil {
				return err
			}
			items = append(items, *model)
		}
		if err := q.runLoaders(items); err != nil {
			return err
		}
		return callback(items)
	})
}

// ChunkById iterates models using keyset pagination on id.
func (q *Querier[T]) ChunkById(size int, callback func([]T) error, column ...string) error {
	q.prepare()
	return q.builder.ChunkById(size, func(rows []map[string]any) error {
		items := make([]T, 0, len(rows))
		for _, row := range rows {
			model, err := mapToModel[T](row)
			if err != nil {
				return err
			}
			items = append(items, *model)
		}
		if err := q.runLoaders(items); err != nil {
			return err
		}
		return callback(items)
	}, column...)
}

// InsertMany inserts multiple attribute maps and returns affected rows.
// Each row goes through mass assignment and fires creating/created when a dispatcher is set.
func InsertMany[T any](rows []map[string]any) (int64, error) {
	if len(rows) == 0 {
		return 0, fmt.Errorf("insert many values required")
	}
	now := time.Now()
	prepared := make([]map[string]any, 0, len(rows))
	for _, attrs := range rows {
		attrs = filterMassAssignment[T](attrs)
		if _, ok := attrs["created_at"]; !ok {
			attrs["created_at"] = now
		}
		if _, ok := attrs["updated_at"]; !ok {
			attrs["updated_at"] = now
		}
		draft := attrsToModel[T](attrs)
		if err := dispatchModel("creating", draft); err != nil {
			return 0, err
		}
		prepared = append(prepared, attrs)
	}
	n, err := tableQuery[T]().InsertBatch(prepared)
	if err != nil {
		return 0, err
	}
	for _, attrs := range prepared {
		_ = dispatchModel("created", attrsToModel[T](attrs))
	}
	return n, nil
}

// Update updates matching rows.
func (q *Querier[T]) Update(attrs map[string]any) (int64, error) {
	attrs = filterMassAssignment[T](attrs)
	if _, ok := attrs["updated_at"]; !ok {
		now := time.Now()
		attrs["updated_at"] = now
	}
	q.prepare()
	return q.builder.Update(attrs)
}

// Delete deletes matching rows (soft delete when available).
func (q *Querier[T]) Delete() (int64, error) {
	if hasSoftDeletes[T]() {
		now := time.Now()
		q.prepare()
		return q.builder.Update(map[string]any{"deleted_at": now, "updated_at": now})
	}
	q.prepare()
	return q.builder.Delete()
}

// ForceDelete permanently deletes matching rows.
func (q *Querier[T]) ForceDelete() (int64, error) {
	q.softDelete = false
	q.prepare()
	return q.builder.Delete()
}

// Restore clears deleted_at on soft-deleted rows.
func (q *Querier[T]) Restore() (int64, error) {
	if !hasSoftDeletes[T]() {
		return 0, fmt.Errorf("model does not use soft deletes")
	}
	q.softDelete = false
	now := time.Now()
	return q.builder.Update(map[string]any{
		"deleted_at": nil,
		"updated_at": now,
	})
}

// SoftDelete soft-deletes a model by id.
func SoftDelete[T any](id any) (int64, error) {
	return Destroy[T](id)
}

// RestoreByID restores a soft-deleted model by id.
func RestoreByID[T any](id any) (int64, error) {
	model, _ := Query[T]().OnlyTrashed().Where(KeyName[T](), id).First()
	if model != nil {
		_ = dispatchModel("restoring", model)
	}
	n, err := Query[T]().OnlyTrashed().Where(KeyName[T](), id).Restore()
	if err != nil {
		return 0, err
	}
	if model != nil {
		_ = dispatchModel("restored", model)
	}
	return n, nil
}

// ForceDeleteByID permanently deletes a model by id (including trashed).
func ForceDeleteByID[T any](id any) (int64, error) {
	model, _ := Query[T]().WithTrashed().Where(KeyName[T](), id).First()
	if model != nil {
		if err := dispatchModel("deleting", model); err != nil {
			return 0, err
		}
	}
	n, err := Query[T]().WithTrashed().Where(KeyName[T](), id).ForceDelete()
	if err != nil {
		return 0, err
	}
	if model != nil {
		_ = dispatchModel("forceDeleted", model)
	}
	return n, nil
}

// Prune permanently deletes soft-deleted rows older than the given duration.
func Prune[T any](olderThan time.Duration) (int64, error) {
	if !hasSoftDeletes[T]() {
		return 0, fmt.Errorf("model does not use soft deletes")
	}
	if olderThan < 0 {
		olderThan = 0
	}
	cutoff := time.Now().Add(-olderThan)
	return Query[T]().OnlyTrashed().Where("deleted_at", "<", cutoff).ForceDelete()
}

// Trashed reports whether a model instance is soft deleted.
func Trashed(model any) bool {
	rv := reflect.ValueOf(model)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	field := rv.FieldByName("DeletedAt")
	if !field.IsValid() {
		for i := 0; i < rv.NumField(); i++ {
			f := rv.Type().Field(i)
			if f.Anonymous && f.Type == reflect.TypeOf(SoftDeletes{}) {
				field = rv.Field(i).FieldByName("DeletedAt")
				break
			}
		}
	}
	if !field.IsValid() {
		return false
	}
	if field.Kind() == reflect.Ptr {
		return !field.IsNil()
	}
	return !field.IsZero()
}

// Save inserts or updates a model pointer.
func Save[T any](model *T) error {
	rv := reflect.ValueOf(model).Elem()
	attrs := filterMassAssignment[T](modelToMap(rv))
	keyName := KeyName[T]()
	keyVal, keyErr := KeyValue(model)
	now := time.Now()

	if keyErr == nil && keyVal != nil && !isZeroAny(keyVal) {
		delete(attrs, keyName)
		attrs["updated_at"] = now
		before := snapshotOriginal(model)
		if err := dispatchModel("updating", model); err != nil {
			return err
		}
		_, err := tableQuery[T]().Where(keyName, keyVal).Update(attrs)
		if err != nil {
			return err
		}
		setTimeField(rv, "UpdatedAt", now)
		recordChanges(model, before)
		SyncOriginal(model)
		_ = dispatchModel("updated", model)
		return nil
	}

	attrs["created_at"] = now
	attrs["updated_at"] = now
	delete(attrs, keyName)
	if err := dispatchModel("creating", model); err != nil {
		return err
	}
	id, err := tableQuery[T]().Insert(attrs)
	if err != nil {
		return err
	}
	idField, _ := fieldValueByColumn(rv, keyName)
	if idField.IsValid() && idField.CanSet() && id > 0 && keyName == "id" {
		_ = setField(idField, id)
	} else if v, ok := attrs[keyName]; ok {
		setKeyField(rv, keyName, v)
	}
	setTimeField(rv, "CreatedAt", now)
	setTimeField(rv, "UpdatedAt", now)
	recordChanges(model, nil)
	SyncOriginal(model)
	_ = dispatchModel("created", model)
	return nil
}

func snapshotOriginal[T any](model *T) map[string]any {
	if model == nil {
		return nil
	}
	key := modelKey(model)
	originalsMu.Lock()
	snap := originals[key]
	originalsMu.Unlock()
	if snap == nil {
		return nil
	}
	out := make(map[string]any, len(snap))
	for k, v := range snap {
		out[k] = v
	}
	return out
}

// Destroy deletes a model by id.
func Destroy[T any](id any) (int64, error) {
	model, _ := Find[T](id)
	if model != nil {
		if err := dispatchModel("deleting", model); err != nil {
			return 0, err
		}
	}
	n, err := Query[T]().Where(KeyName[T](), id).Delete()
	if err != nil {
		return 0, err
	}
	if model != nil {
		_ = dispatchModel("deleted", model)
	}
	return n, nil
}

func (q *Querier[T]) applySoftDelete() {
	if q.softDelete && !q.softApplied {
		q.builder.WhereNull("deleted_at")
		q.softApplied = true
	}
}

func hasSoftDeletes[T any]() bool {
	var zero T
	rt := reflect.TypeOf(zero)
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		if field.Anonymous && field.Type == reflect.TypeOf(SoftDeletes{}) {
			return true
		}
		if field.Name == "DeletedAt" {
			return true
		}
	}
	return false
}

func attrsToModel[T any](attrs map[string]any) *T {
	model, err := mapToModel[T](attrs)
	if err != nil || model == nil {
		var zero T
		return &zero
	}
	return model
}

func mapToModel[T any](row map[string]any) (*T, error) {
	var model T
	rv := reflect.ValueOf(&model).Elem()
	if err := hydrate(rv, row); err != nil {
		return nil, err
	}
	SyncOriginal(&model)
	return &model, nil
}

func hydrate(rv reflect.Value, row map[string]any) error {
	rt := rv.Type()
	casts := castsOf(rv)
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		fv := rv.Field(i)
		if field.Anonymous {
			if err := hydrate(fv, row); err != nil {
				return err
			}
			continue
		}
		column := columnName(field)
		value, ok := row[column]
		if !ok || value == nil {
			continue
		}
		if cast, exists := casts[column]; exists {
			casted, err := castIncoming(cast, value)
			if err != nil {
				return fmt.Errorf("field %s: %w", field.Name, err)
			}
			value = casted
		}
		if err := setField(fv, value); err != nil {
			return fmt.Errorf("field %s: %w", field.Name, err)
		}
	}
	return nil
}

func modelToMap(rv reflect.Value) map[string]any {
	out := make(map[string]any)
	collectAttrs(rv, out)
	return out
}

func collectAttrs(rv reflect.Value, out map[string]any) {
	rt := rv.Type()
	casts := castsOf(rv)
	for i := 0; i < rt.NumField(); i++ {
		field := rt.Field(i)
		fv := rv.Field(i)
		if field.Anonymous {
			collectAttrs(fv, out)
			continue
		}
		if tag := field.Tag.Get("db"); tag == "-" {
			continue
		}
		column := columnName(field)
		if !fv.IsValid() {
			continue
		}
		var value any
		if fv.Kind() == reflect.Ptr {
			if fv.IsNil() {
				out[column] = nil
				continue
			}
			value = fv.Elem().Interface()
		} else {
			value = fv.Interface()
		}
		if cast, ok := casts[column]; ok {
			value = castOutgoing(cast, value)
		}
		out[column] = value
	}
}

func columnName(field reflect.StructField) string {
	if tag := field.Tag.Get("db"); tag != "" && tag != "-" {
		parts := strings.Split(tag, ",")
		return parts[0]
	}
	return toSnake(field.Name)
}

func setField(fv reflect.Value, value any) error {
	if !fv.CanSet() {
		return nil
	}

	if fv.Kind() == reflect.Ptr {
		if fv.IsNil() {
			fv.Set(reflect.New(fv.Type().Elem()))
		}
		return setField(fv.Elem(), value)
	}

	switch fv.Kind() {
	case reflect.String:
		fv.SetString(fmt.Sprint(value))
	case reflect.Bool:
		switch v := value.(type) {
		case bool:
			fv.SetBool(v)
		case int64:
			fv.SetBool(v != 0)
		case []byte:
			fv.SetBool(string(v) == "1" || string(v) == "true")
		default:
			fv.SetBool(fmt.Sprint(value) == "1" || fmt.Sprint(value) == "true")
		}
	case reflect.Int, reflect.Int64, reflect.Int32:
		switch v := value.(type) {
		case int64:
			fv.SetInt(v)
		case int:
			fv.SetInt(int64(v))
		case []byte:
			var n int64
			_, err := fmt.Sscan(string(v), &n)
			if err != nil {
				return err
			}
			fv.SetInt(n)
		default:
			var n int64
			_, err := fmt.Sscan(fmt.Sprint(v), &n)
			if err != nil {
				return err
			}
			fv.SetInt(n)
		}
	case reflect.Float32, reflect.Float64:
		switch v := value.(type) {
		case float64:
			fv.SetFloat(v)
		case float32:
			fv.SetFloat(float64(v))
		default:
			var n float64
			_, err := fmt.Sscan(fmt.Sprint(v), &n)
			if err != nil {
				return err
			}
			fv.SetFloat(n)
		}
	case reflect.Struct:
		if fv.Type() == reflect.TypeOf(time.Time{}) {
			switch v := value.(type) {
			case time.Time:
				fv.Set(reflect.ValueOf(v))
			case string:
				parsed, err := parseTime(v)
				if err != nil {
					return err
				}
				fv.Set(reflect.ValueOf(parsed))
			case []byte:
				parsed, err := parseTime(string(v))
				if err != nil {
					return err
				}
				fv.Set(reflect.ValueOf(parsed))
			}
		}
	case reflect.Map:
		rv := reflect.ValueOf(value)
		if rv.IsValid() && rv.Type().AssignableTo(fv.Type()) {
			fv.Set(rv)
			return nil
		}
		if m, ok := value.(map[string]any); ok && fv.Type().Key().Kind() == reflect.String {
			out := reflect.MakeMap(fv.Type())
			for k, v := range m {
				out.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v))
			}
			fv.Set(out)
			return nil
		}
		if s, ok := value.(string); ok {
			var dest map[string]any
			if err := json.Unmarshal([]byte(s), &dest); err == nil {
				return setField(fv, dest)
			}
		}
	}
	return nil
}

func setTimeField(rv reflect.Value, name string, value time.Time) {
	field := rv.FieldByName(name)
	if !field.IsValid() || !field.CanSet() {
		// Look in embedded Model.
		for i := 0; i < rv.NumField(); i++ {
			f := rv.Type().Field(i)
			if f.Anonymous {
				setTimeField(rv.Field(i), name, value)
			}
		}
		return
	}
	if field.Kind() == reflect.Ptr {
		t := value
		field.Set(reflect.ValueOf(&t))
		return
	}
	field.Set(reflect.ValueOf(value))
}

func parseTime(value string) (time.Time, error) {
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, format := range formats {
		if t, err := time.Parse(format, value); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time: %s", value)
}

func toSnake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		b.WriteRune(r)
	}
	return strings.ToLower(b.String())
}

func pluralize(name string) string {
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, "y") && len(name) > 1 {
		return name[:len(name)-1] + "ies"
	}
	if strings.HasSuffix(lower, "s") {
		return name + "es"
	}
	return name + "s"
}
