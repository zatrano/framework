package query

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
)

// DBTX is satisfied by *sql.DB and *sql.Tx.
type DBTX interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

// Builder builds SQL queries fluently.
type Builder struct {
	db             DBTX
	driver         string
	table          string
	columns        []string
	selectBindings []any
	wheres         []whereClause
	orders         []orderClause
	groups         []string
	havings        []whereClause
	joins          []string
	unions         []unionClause
	limitN         int
	offsetN        int
	distinct       bool
	lockMode       string // "", "update", "share"
	lockSkip       bool
	lockNoWait     bool
	err            error
}

type whereClause struct {
	boolean string
	sql     string
	args    []any
}

type orderClause struct {
	sql  string
	args []any
}

type unionClause struct {
	all   bool
	query *Builder
}

// New creates a query builder.
func New(db DBTX, driver, table string) *Builder {
	return &Builder{
		db:     db,
		driver: driver,
		table:  table,
	}
}

// Table sets the table name.
func (b *Builder) Table(table string) *Builder {
	b.table = table
	return b
}

// Select sets the columns to select.
func (b *Builder) Select(columns ...string) *Builder {
	b.columns = columns
	b.selectBindings = nil
	return b
}

// AddSelect appends columns to the current select list.
func (b *Builder) AddSelect(columns ...string) *Builder {
	b.columns = append(b.columns, columns...)
	return b
}

// SelectRaw appends a raw SELECT expression with optional bindings.
func (b *Builder) SelectRaw(expression string, bindings ...any) *Builder {
	b.columns = append(b.columns, expression)
	b.selectBindings = append(b.selectBindings, bindings...)
	return b
}

// Distinct enables SELECT DISTINCT.
func (b *Builder) Distinct() *Builder {
	b.distinct = true
	return b
}

// Where adds a basic where clause.
func (b *Builder) Where(column string, args ...any) *Builder {
	return b.addWhere("and", column, args...)
}

// OrWhere adds an OR where clause.
func (b *Builder) OrWhere(column string, args ...any) *Builder {
	return b.addWhere("or", column, args...)
}

// WhereAny applies the same comparison to columns with OR (grouped).
func (b *Builder) WhereAny(columns []string, args ...any) *Builder {
	return b.addWhereMulti("and", "or", columns, args...)
}

// WhereAll applies the same comparison to columns with AND (grouped).
func (b *Builder) WhereAll(columns []string, args ...any) *Builder {
	return b.addWhereMulti("and", "and", columns, args...)
}

// OrWhereAny is WhereAny combined with OR at the outer level.
func (b *Builder) OrWhereAny(columns []string, args ...any) *Builder {
	return b.addWhereMulti("or", "or", columns, args...)
}

// OrWhereAll is WhereAll combined with OR at the outer level.
func (b *Builder) OrWhereAll(columns []string, args ...any) *Builder {
	return b.addWhereMulti("or", "and", columns, args...)
}

// WhereIn adds a WHERE IN clause.
func (b *Builder) WhereIn(column string, values []any) *Builder {
	return b.addWhereIn("and", column, values, false)
}

// OrWhereIn adds an OR WHERE IN clause.
func (b *Builder) OrWhereIn(column string, values []any) *Builder {
	return b.addWhereIn("or", column, values, false)
}

// WhereNotIn adds a WHERE NOT IN clause.
func (b *Builder) WhereNotIn(column string, values []any) *Builder {
	return b.addWhereIn("and", column, values, true)
}

// OrWhereNotIn adds an OR WHERE NOT IN clause.
func (b *Builder) OrWhereNotIn(column string, values []any) *Builder {
	return b.addWhereIn("or", column, values, true)
}

// WhereBetween adds a WHERE BETWEEN clause.
func (b *Builder) WhereBetween(column string, min, max any) *Builder {
	return b.addRawWhere("and", column+" BETWEEN ? AND ?", min, max)
}

// OrWhereBetween adds an OR WHERE BETWEEN clause.
func (b *Builder) OrWhereBetween(column string, min, max any) *Builder {
	return b.addRawWhere("or", column+" BETWEEN ? AND ?", min, max)
}

// WhereNotBetween adds a WHERE NOT BETWEEN clause.
func (b *Builder) WhereNotBetween(column string, min, max any) *Builder {
	return b.addRawWhere("and", column+" NOT BETWEEN ? AND ?", min, max)
}

// OrWhereNotBetween adds an OR WHERE NOT BETWEEN clause.
func (b *Builder) OrWhereNotBetween(column string, min, max any) *Builder {
	return b.addRawWhere("or", column+" NOT BETWEEN ? AND ?", min, max)
}

// WhereLike adds a WHERE LIKE clause.
func (b *Builder) WhereLike(column string, pattern string) *Builder {
	return b.addRawWhere("and", column+" LIKE ?", pattern)
}

// WhereNotLike adds a WHERE NOT LIKE clause.
func (b *Builder) WhereNotLike(column string, pattern string) *Builder {
	return b.addRawWhere("and", column+" NOT LIKE ?", pattern)
}

// OrWhereLike adds an OR WHERE LIKE clause.
func (b *Builder) OrWhereLike(column string, pattern string) *Builder {
	return b.addRawWhere("or", column+" LIKE ?", pattern)
}

// OrWhereNotLike adds an OR WHERE NOT LIKE clause.
func (b *Builder) OrWhereNotLike(column string, pattern string) *Builder {
	return b.addRawWhere("or", column+" NOT LIKE ?", pattern)
}

// WhereNull adds WHERE column IS NULL.
func (b *Builder) WhereNull(column string) *Builder {
	return b.addRawWhere("and", column+" IS NULL")
}

// WhereNotNull adds WHERE column IS NOT NULL.
func (b *Builder) WhereNotNull(column string) *Builder {
	return b.addRawWhere("and", column+" IS NOT NULL")
}

// OrWhereNull adds OR WHERE column IS NULL.
func (b *Builder) OrWhereNull(column string) *Builder {
	return b.addRawWhere("or", column+" IS NULL")
}

// OrWhereNotNull adds OR WHERE column IS NOT NULL.
func (b *Builder) OrWhereNotNull(column string) *Builder {
	return b.addRawWhere("or", column+" IS NOT NULL")
}

// WhereColumn compares two columns (WhereColumn("a", "b") or WhereColumn("a", ">", "b")).
func (b *Builder) WhereColumn(first string, parts ...string) *Builder {
	return b.addColumnWhere("and", first, parts...)
}

// OrWhereColumn compares two columns with OR.
func (b *Builder) OrWhereColumn(first string, parts ...string) *Builder {
	return b.addColumnWhere("or", first, parts...)
}

// WhereRaw adds a raw WHERE fragment with bindings.
func (b *Builder) WhereRaw(sqlStr string, bindings ...any) *Builder {
	return b.addRawWhere("and", sqlStr, bindings...)
}

// OrWhereRaw adds a raw OR WHERE fragment with bindings.
func (b *Builder) OrWhereRaw(sqlStr string, bindings ...any) *Builder {
	return b.addRawWhere("or", sqlStr, bindings...)
}

// WhereExists adds a WHERE EXISTS (subquery) clause.
func (b *Builder) WhereExists(sub *Builder) *Builder {
	return b.addExists("and", false, sub)
}

// WhereNotExists adds a WHERE NOT EXISTS (subquery) clause.
func (b *Builder) WhereNotExists(sub *Builder) *Builder {
	return b.addExists("and", true, sub)
}

// OrWhereExists adds an OR EXISTS (subquery) clause.
func (b *Builder) OrWhereExists(sub *Builder) *Builder {
	return b.addExists("or", false, sub)
}

// OrWhereNotExists adds an OR NOT EXISTS (subquery) clause.
func (b *Builder) OrWhereNotExists(sub *Builder) *Builder {
	return b.addExists("or", true, sub)
}

func (b *Builder) addExists(boolean string, not bool, sub *Builder) *Builder {
	if sub == nil {
		return b
	}
	clone := sub.clone()
	if len(clone.columns) == 0 {
		clone.Select("1")
	}
	clone.orders = nil
	clone.lockMode = ""
	clone.lockSkip = false
	clone.lockNoWait = false
	sqlStr, args := clone.ToSQL()
	keyword := "EXISTS"
	if not {
		keyword = "NOT EXISTS"
	}
	return b.addRawWhere(boolean, keyword+" ("+sqlStr+")", args...)
}

// When applies fn when condition is true.
func (b *Builder) When(condition bool, fn func(*Builder) *Builder) *Builder {
	if condition && fn != nil {
		return fn(b)
	}
	return b
}

// Unless applies fn when condition is false.
func (b *Builder) Unless(condition bool, fn func(*Builder) *Builder) *Builder {
	return b.When(!condition, fn)
}

// Tap runs fn with the builder and returns the builder unchanged for chaining.
func (b *Builder) Tap(fn func(*Builder)) *Builder {
	if fn != nil {
		fn(b)
	}
	return b
}

// GetBindings returns the bound arguments for the compiled SQL.
func (b *Builder) GetBindings() []any {
	_, args := b.ToSQL()
	return args
}

// WhereDate compares DATE(column) to value (driver-aware).
func (b *Builder) WhereDate(column string, value any) *Builder {
	switch {
	case strings.Contains(b.driver, "sqlite"):
		return b.addRawWhere("and", "date("+column+") = ?", value)
	default:
		return b.addRawWhere("and", "DATE("+column+") = ?", value)
	}
}

// WhereMonth compares the month of column to month (1-12, driver-aware).
func (b *Builder) WhereMonth(column string, month any) *Builder {
	switch {
	case strings.Contains(b.driver, "sqlite"):
		return b.addRawWhere("and", "CAST(strftime('%m', "+column+") AS INTEGER) = ?", month)
	case b.driver == "pgsql" || b.driver == "postgres" || b.driver == "postgresql":
		return b.addRawWhere("and", "EXTRACT(MONTH FROM "+column+") = ?", month)
	default:
		return b.addRawWhere("and", "MONTH("+column+") = ?", month)
	}
}

// WhereYear compares the year of column to year (driver-aware).
func (b *Builder) WhereYear(column string, year any) *Builder {
	switch {
	case strings.Contains(b.driver, "sqlite"):
		return b.addRawWhere("and", "CAST(strftime('%Y', "+column+") AS INTEGER) = ?", year)
	case b.driver == "pgsql" || b.driver == "postgres" || b.driver == "postgresql":
		return b.addRawWhere("and", "EXTRACT(YEAR FROM "+column+") = ?", year)
	default:
		return b.addRawWhere("and", "YEAR("+column+") = ?", year)
	}
}

// WhereDay compares the day-of-month of column (1-31, driver-aware).
func (b *Builder) WhereDay(column string, day any) *Builder {
	switch {
	case strings.Contains(b.driver, "sqlite"):
		return b.addRawWhere("and", "CAST(strftime('%d', "+column+") AS INTEGER) = ?", day)
	case b.driver == "pgsql" || b.driver == "postgres" || b.driver == "postgresql":
		return b.addRawWhere("and", "EXTRACT(DAY FROM "+column+") = ?", day)
	default:
		return b.addRawWhere("and", "DAY("+column+") = ?", day)
	}
}

// WhereTime compares the time-of-day of column (driver-aware).
func (b *Builder) WhereTime(column string, value any) *Builder {
	switch {
	case strings.Contains(b.driver, "sqlite"):
		return b.addRawWhere("and", "time("+column+") = ?", value)
	case b.driver == "pgsql" || b.driver == "postgres" || b.driver == "postgresql":
		return b.addRawWhere("and", "CAST("+column+" AS time) = ?", value)
	default:
		return b.addRawWhere("and", "TIME("+column+") = ?", value)
	}
}

// WhereDayOfWeek compares weekday of column (0=Sunday … 6=Saturday, driver-aware).
func (b *Builder) WhereDayOfWeek(column string, day any) *Builder {
	switch {
	case strings.Contains(b.driver, "sqlite"):
		return b.addRawWhere("and", "CAST(strftime('%w', "+column+") AS INTEGER) = ?", day)
	case b.driver == "pgsql" || b.driver == "postgres" || b.driver == "postgresql":
		return b.addRawWhere("and", "EXTRACT(DOW FROM "+column+") = ?", day)
	default:
		// MySQL DAYOFWEEK is 1=Sunday … 7=Saturday; normalize to 0-6.
		return b.addRawWhere("and", "DAYOFWEEK("+column+") - 1 = ?", day)
	}
}

// WhereHour compares the hour of column (0-23, driver-aware).
func (b *Builder) WhereHour(column string, hour any) *Builder {
	switch {
	case strings.Contains(b.driver, "sqlite"):
		return b.addRawWhere("and", "CAST(strftime('%H', "+column+") AS INTEGER) = ?", hour)
	case b.driver == "pgsql" || b.driver == "postgres" || b.driver == "postgresql":
		return b.addRawWhere("and", "EXTRACT(HOUR FROM "+column+") = ?", hour)
	default:
		return b.addRawWhere("and", "HOUR("+column+") = ?", hour)
	}
}

// OrderBy adds an order by clause.
// Column must be a safe SQL identifier; direction is restricted to asc/desc.
func (b *Builder) OrderBy(column string, direction ...string) *Builder {
	col, err := sanitizeIdentifier(column)
	if err != nil {
		b.err = fmt.Errorf("order by: %w", err)
		return b
	}
	dir := "asc"
	if len(direction) > 0 && direction[0] != "" {
		dir = sanitizeOrderDirection(direction[0])
	}
	b.orders = append(b.orders, orderClause{
		sql: fmt.Sprintf("%s %s", col, dir),
	})
	return b
}

// OrderByRaw adds a raw ORDER BY expression with optional bindings.
func (b *Builder) OrderByRaw(sqlStr string, bindings ...any) *Builder {
	b.orders = append(b.orders, orderClause{sql: sqlStr, args: bindings})
	return b
}

// OrderByDesc orders by column descending.
func (b *Builder) OrderByDesc(column string) *Builder {
	return b.OrderBy(column, "desc")
}

// Reorder clears existing order clauses, then optionally adds a new OrderBy.
func (b *Builder) Reorder(column ...string) *Builder {
	b.orders = nil
	if len(column) == 0 || column[0] == "" {
		return b
	}
	dir := "asc"
	if len(column) > 1 && column[1] != "" {
		dir = column[1]
	}
	return b.OrderBy(column[0], dir)
}

// Latest orders by column descending (default created_at).
func (b *Builder) Latest(column ...string) *Builder {
	col := "created_at"
	if len(column) > 0 && column[0] != "" {
		col = column[0]
	}
	return b.OrderBy(col, "desc")
}

// Oldest orders by column ascending.
func (b *Builder) Oldest(column ...string) *Builder {
	col := "created_at"
	if len(column) > 0 && column[0] != "" {
		col = column[0]
	}
	return b.OrderBy(col, "asc")
}

// InRandomOrder orders rows randomly (driver-aware).
func (b *Builder) InRandomOrder() *Builder {
	switch strings.ToLower(b.driver) {
	case "mysql":
		return b.OrderByRaw("RAND()")
	case "mssql", "sqlserver":
		return b.OrderByRaw("NEWID()")
	case "oracle", "ora":
		return b.OrderByRaw("DBMS_RANDOM.VALUE")
	default:
		return b.OrderByRaw("RANDOM()")
	}
}

// GroupBy adds group by columns.
func (b *Builder) GroupBy(columns ...string) *Builder {
	for _, column := range columns {
		col, err := sanitizeIdentifier(column)
		if err != nil {
			b.err = fmt.Errorf("group by: %w", err)
			return b
		}
		b.groups = append(b.groups, col)
	}
	return b
}

// Having adds a having clause.
func (b *Builder) Having(column string, operator string, value any) *Builder {
	col, err := sanitizeIdentifier(column)
	if err != nil {
		b.err = fmt.Errorf("having: %w", err)
		return b
	}
	op, err := sanitizeOperator(operator)
	if err != nil {
		b.err = fmt.Errorf("having: %w", err)
		return b
	}
	b.havings = append(b.havings, whereClause{
		boolean: "and",
		sql:     fmt.Sprintf("%s %s ?", col, op),
		args:    []any{value},
	})
	return b
}

// OrHaving adds an OR having clause.
func (b *Builder) OrHaving(column string, operator string, value any) *Builder {
	col, err := sanitizeIdentifier(column)
	if err != nil {
		b.err = fmt.Errorf("or having: %w", err)
		return b
	}
	op, err := sanitizeOperator(operator)
	if err != nil {
		b.err = fmt.Errorf("or having: %w", err)
		return b
	}
	b.havings = append(b.havings, whereClause{
		boolean: "or",
		sql:     fmt.Sprintf("%s %s ?", col, op),
		args:    []any{value},
	})
	return b
}

// HavingRaw adds a raw HAVING fragment with bindings.
func (b *Builder) HavingRaw(sqlStr string, bindings ...any) *Builder {
	b.havings = append(b.havings, whereClause{
		boolean: "and",
		sql:     sqlStr,
		args:    bindings,
	})
	return b
}

// OrHavingRaw adds a raw OR HAVING fragment with bindings.
func (b *Builder) OrHavingRaw(sqlStr string, bindings ...any) *Builder {
	b.havings = append(b.havings, whereClause{
		boolean: "or",
		sql:     sqlStr,
		args:    bindings,
	})
	return b
}

// Join adds an inner join.
func (b *Builder) Join(table, first, operator, second string) *Builder {
	t, err := sanitizeIdentifier(table)
	if err != nil {
		b.err = fmt.Errorf("join: %w", err)
		return b
	}
	f, err := sanitizeIdentifier(first)
	if err != nil {
		b.err = fmt.Errorf("join: %w", err)
		return b
	}
	op, err := sanitizeOperator(operator)
	if err != nil {
		b.err = fmt.Errorf("join: %w", err)
		return b
	}
	s, err := sanitizeIdentifier(second)
	if err != nil {
		b.err = fmt.Errorf("join: %w", err)
		return b
	}
	b.joins = append(b.joins, fmt.Sprintf("INNER JOIN %s ON %s %s %s", t, f, op, s))
	return b
}

// LeftJoin adds a left join.
func (b *Builder) LeftJoin(table, first, operator, second string) *Builder {
	t, err := sanitizeIdentifier(table)
	if err != nil {
		b.err = fmt.Errorf("left join: %w", err)
		return b
	}
	f, err := sanitizeIdentifier(first)
	if err != nil {
		b.err = fmt.Errorf("left join: %w", err)
		return b
	}
	op, err := sanitizeOperator(operator)
	if err != nil {
		b.err = fmt.Errorf("left join: %w", err)
		return b
	}
	s, err := sanitizeIdentifier(second)
	if err != nil {
		b.err = fmt.Errorf("left join: %w", err)
		return b
	}
	b.joins = append(b.joins, fmt.Sprintf("LEFT JOIN %s ON %s %s %s", t, f, op, s))
	return b
}

// RightJoin adds a right join.
func (b *Builder) RightJoin(table, first, operator, second string) *Builder {
	t, err := sanitizeIdentifier(table)
	if err != nil {
		b.err = fmt.Errorf("right join: %w", err)
		return b
	}
	f, err := sanitizeIdentifier(first)
	if err != nil {
		b.err = fmt.Errorf("right join: %w", err)
		return b
	}
	op, err := sanitizeOperator(operator)
	if err != nil {
		b.err = fmt.Errorf("right join: %w", err)
		return b
	}
	s, err := sanitizeIdentifier(second)
	if err != nil {
		b.err = fmt.Errorf("right join: %w", err)
		return b
	}
	b.joins = append(b.joins, fmt.Sprintf("RIGHT JOIN %s ON %s %s %s", t, f, op, s))
	return b
}

// CrossJoin adds a cross join.
func (b *Builder) CrossJoin(table string) *Builder {
	b.joins = append(b.joins, fmt.Sprintf("CROSS JOIN %s", table))
	return b
}

// Limit sets the limit.
func (b *Builder) Limit(n int) *Builder {
	b.limitN = n
	return b
}

// Offset sets the offset.
func (b *Builder) Offset(n int) *Builder {
	b.offsetN = n
	return b
}

// Take is an alias for Limit.
func (b *Builder) Take(n int) *Builder {
	return b.Limit(n)
}

// Skip is an alias for Offset.
func (b *Builder) Skip(n int) *Builder {
	return b.Offset(n)
}

// ForPage applies limit/offset for a 1-based page.
func (b *Builder) ForPage(page, perPage int) *Builder {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 15
	}
	return b.Offset((page - 1) * perPage).Limit(perPage)
}

// Union appends a UNION query.
func (b *Builder) Union(other *Builder) *Builder {
	if other == nil {
		return b
	}
	b.unions = append(b.unions, unionClause{all: false, query: other.clone()})
	return b
}

// UnionAll appends a UNION ALL query.
func (b *Builder) UnionAll(other *Builder) *Builder {
	if other == nil {
		return b
	}
	b.unions = append(b.unions, unionClause{all: true, query: other.clone()})
	return b
}

// Clone returns a deep copy of the builder.
func (b *Builder) Clone() *Builder {
	return b.clone()
}

// LockForUpdate adds a pessimistic write lock (FOR UPDATE).
func (b *Builder) LockForUpdate() *Builder {
	b.lockMode = "update"
	return b
}

// ForUpdate is an alias for LockForUpdate.
func (b *Builder) ForUpdate() *Builder {
	return b.LockForUpdate()
}

// SharedLock adds a shared lock (FOR SHARE / LOCK IN SHARE MODE).
func (b *Builder) SharedLock() *Builder {
	b.lockMode = "share"
	return b
}

// SkipLocked appends SKIP LOCKED to the lock clause.
func (b *Builder) SkipLocked() *Builder {
	b.lockSkip = true
	b.lockNoWait = false
	return b
}

// NoWait appends NOWAIT to the lock clause.
func (b *Builder) NoWait() *Builder {
	b.lockNoWait = true
	b.lockSkip = false
	return b
}

// ToSQL returns the SELECT SQL and args.
func (b *Builder) ToSQL() (string, []any) {
	coreSQL, args := b.compileSelectCore()

	var sb strings.Builder
	sb.WriteString(coreSQL)
	for _, u := range b.unions {
		part := u.query.clone()
		part.orders = nil
		part.limitN = 0
		part.offsetN = 0
		part.lockMode = ""
		part.unions = nil
		partSQL, partArgs := part.compileSelectCore()
		if u.all {
			sb.WriteString(" UNION ALL ")
		} else {
			sb.WriteString(" UNION ")
		}
		sb.WriteString(partSQL)
		args = append(args, partArgs...)
	}

	if len(b.orders) > 0 {
		orderParts := make([]string, 0, len(b.orders))
		for _, order := range b.orders {
			orderParts = append(orderParts, order.sql)
			args = append(args, order.args...)
		}
		sb.WriteString(" ORDER BY ")
		sb.WriteString(strings.Join(orderParts, ", "))
	}

	if b.limitN > 0 {
		sb.WriteString(fmt.Sprintf(" LIMIT %d", b.limitN))
	}
	if b.offsetN > 0 {
		sb.WriteString(fmt.Sprintf(" OFFSET %d", b.offsetN))
	}
	if lock := b.compileLock(); lock != "" {
		sb.WriteString(lock)
	}

	return b.rebind(sb.String()), args
}

// compileSelectCore builds SELECT..HAVING without ORDER/LIMIT/LOCK/UNION.
func (b *Builder) compileSelectCore() (string, []any) {
	columns := "*"
	if len(b.columns) > 0 {
		columns = strings.Join(b.columns, ", ")
	}
	if b.distinct {
		columns = "DISTINCT " + columns
	}

	var sb strings.Builder
	sb.WriteString("SELECT ")
	sb.WriteString(columns)
	sb.WriteString(" FROM ")
	sb.WriteString(b.table)

	for _, join := range b.joins {
		sb.WriteString(" ")
		sb.WriteString(join)
	}

	args := make([]any, 0)
	args = append(args, b.selectBindings...)

	whereSQL, whereArgs := b.compileWheres(b.wheres)
	if whereSQL != "" {
		sb.WriteString(" WHERE ")
		sb.WriteString(whereSQL)
		args = append(args, whereArgs...)
	}

	if len(b.groups) > 0 {
		sb.WriteString(" GROUP BY ")
		sb.WriteString(strings.Join(b.groups, ", "))
	}

	havingSQL, havingArgs := b.compileWheres(b.havings)
	if havingSQL != "" {
		sb.WriteString(" HAVING ")
		sb.WriteString(havingSQL)
		args = append(args, havingArgs...)
	}

	return sb.String(), args
}

// Get executes the select query and returns maps.
func (b *Builder) Get() ([]map[string]any, error) {
	if b.err != nil {
		return nil, b.err
	}
	sqlStr, args := b.ToSQL()
	rows, err := b.db.Query(sqlStr, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRows(rows)
}

// First returns the first matching row.
func (b *Builder) First() (map[string]any, error) {
	b.Limit(1)
	rows, err := b.Get()
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, sql.ErrNoRows
	}
	return rows[0], nil
}

// Value returns a single column value.
func (b *Builder) Value(column string) (any, error) {
	b.Select(column)
	row, err := b.First()
	if err != nil {
		return nil, err
	}
	return row[column], nil
}

// Pluck returns all values for a column.
func (b *Builder) Pluck(column string) ([]any, error) {
	clone := b.clone()
	clone.Select(column)
	rows, err := clone.Get()
	if err != nil {
		return nil, err
	}
	out := make([]any, 0, len(rows))
	for _, row := range rows {
		out = append(out, row[column])
	}
	return out, nil
}

// Count returns the number of matching rows.
func (b *Builder) Count() (int64, error) {
	return b.aggregateInt("COUNT(*)")
}

// Sum returns the sum of a column.
func (b *Builder) Sum(column string) (float64, error) {
	return b.aggregateFloat(fmt.Sprintf("SUM(%s)", column))
}

// Avg returns the average of a column.
func (b *Builder) Avg(column string) (float64, error) {
	return b.aggregateFloat(fmt.Sprintf("AVG(%s)", column))
}

// Min returns the minimum of a column.
func (b *Builder) Min(column string) (any, error) {
	return b.aggregateValue(fmt.Sprintf("MIN(%s)", column))
}

// Max returns the maximum of a column.
func (b *Builder) Max(column string) (any, error) {
	return b.aggregateValue(fmt.Sprintf("MAX(%s)", column))
}

func (b *Builder) aggregateInt(expression string) (int64, error) {
	v, err := b.aggregateValue(expression)
	if err != nil {
		return 0, err
	}
	return toInt64(v)
}

func (b *Builder) aggregateFloat(expression string) (float64, error) {
	v, err := b.aggregateValue(expression)
	if err != nil {
		return 0, err
	}
	if v == nil {
		return 0, nil
	}
	switch n := v.(type) {
	case float64:
		return n, nil
	case float32:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case int:
		return float64(n), nil
	case []byte:
		var f float64
		_, err := fmt.Sscan(string(n), &f)
		return f, err
	default:
		var f float64
		_, err := fmt.Sscan(fmt.Sprint(n), &f)
		return f, err
	}
}

func (b *Builder) aggregateValue(expression string) (any, error) {
	clone := b.clone()
	clone.columns = []string{expression + " as aggregate"}
	clone.selectBindings = nil
	clone.orders = nil
	clone.limitN = 0
	clone.offsetN = 0
	row, err := clone.First()
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return row["aggregate"], nil
}

func toInt64(v any) (int64, error) {
	if v == nil {
		return 0, nil
	}
	switch n := v.(type) {
	case int64:
		return n, nil
	case int:
		return int64(n), nil
	case float64:
		return int64(n), nil
	case []byte:
		var out int64
		_, err := fmt.Sscan(string(n), &out)
		return out, err
	default:
		var out int64
		_, err := fmt.Sscan(fmt.Sprint(n), &out)
		return out, err
	}
}

// Exists reports whether any matching row exists.
func (b *Builder) Exists() (bool, error) {
	count, err := b.Count()
	return count > 0, err
}

// Paginate returns a page of results with total count metadata.
func (b *Builder) Paginate(page, perPage int) (items []map[string]any, total int64, err error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 15
	}

	total, err = b.Count()
	if err != nil {
		return nil, 0, err
	}

	clone := b.clone()
	clone.Limit(perPage).Offset((page - 1) * perPage)
	items, err = clone.Get()
	return items, total, err
}

// Insert inserts a row and returns last insert id when available.
func (b *Builder) Insert(values map[string]any) (int64, error) {
	if len(values) == 0 {
		return 0, fmt.Errorf("insert values required")
	}
	columns := make([]string, 0, len(values))
	for column := range values {
		columns = append(columns, column)
	}
	sort.Strings(columns)
	placeholders := make([]string, 0, len(columns))
	args := make([]any, 0, len(columns))
	for _, column := range columns {
		placeholders = append(placeholders, "?")
		args = append(args, values[column])
	}
	sqlStr := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", b.table, strings.Join(columns, ", "), strings.Join(placeholders, ", "))
	if isPostgresDriver(b.driver) {
		sqlStr += " RETURNING id"
		var id int64
		err := b.db.QueryRow(b.rebind(sqlStr), args...).Scan(&id)
		return id, err
	}
	if isSQLServerDriver(b.driver) {
		sqlStr += " OUTPUT INSERTED.id"
		var id int64
		err := b.db.QueryRow(b.rebind(sqlStr), args...).Scan(&id)
		return id, err
	}
	result, err := b.db.Exec(b.rebind(sqlStr), args...)
	if err != nil {
		return 0, err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

// InsertGetID is an alias for Insert.
func (b *Builder) InsertGetID(values map[string]any) (int64, error) {
	return b.Insert(values)
}

// Upsert inserts a row or updates columns on unique key conflict.
// uniqueBy lists conflict target columns; update lists columns to update (default: all non-unique keys).
func (b *Builder) Upsert(values map[string]any, uniqueBy []string, update ...string) (int64, error) {
	if len(values) == 0 {
		return 0, fmt.Errorf("upsert values required")
	}
	if len(uniqueBy) == 0 {
		return 0, fmt.Errorf("upsert unique columns required")
	}

	columns := make([]string, 0, len(values))
	placeholders := make([]string, 0, len(values))
	args := make([]any, 0, len(values))
	for column, value := range values {
		columns = append(columns, column)
		placeholders = append(placeholders, "?")
		args = append(args, value)
	}

	updateCols := update
	if len(updateCols) == 0 {
		unique := make(map[string]bool, len(uniqueBy))
		for _, col := range uniqueBy {
			unique[col] = true
		}
		for _, col := range columns {
			if !unique[col] {
				updateCols = append(updateCols, col)
			}
		}
	}
	if len(updateCols) == 0 {
		return 0, fmt.Errorf("upsert update columns required")
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", b.table, strings.Join(columns, ", "), strings.Join(placeholders, ", ")))

	driver := strings.ToLower(b.driver)
	switch {
	case strings.Contains(driver, "mysql"):
		sets := make([]string, 0, len(updateCols))
		for _, col := range updateCols {
			sets = append(sets, fmt.Sprintf("%s = VALUES(%s)", col, col))
		}
		sb.WriteString(" ON DUPLICATE KEY UPDATE ")
		sb.WriteString(strings.Join(sets, ", "))
	case isSQLServerDriver(driver):
		// MERGE … USING (VALUES …) — uniqueBy is the match key.
		srcCols := make([]string, 0, len(columns))
		for i, col := range columns {
			srcCols = append(srcCols, fmt.Sprintf("? AS %s", col))
			_ = i
		}
		onParts := make([]string, 0, len(uniqueBy))
		for _, col := range uniqueBy {
			onParts = append(onParts, fmt.Sprintf("target.%s = source.%s", col, col))
		}
		setParts := make([]string, 0, len(updateCols))
		for _, col := range updateCols {
			setParts = append(setParts, fmt.Sprintf("target.%s = source.%s", col, col))
		}
		insertCols := strings.Join(columns, ", ")
		insertVals := make([]string, 0, len(columns))
		for _, col := range columns {
			insertVals = append(insertVals, "source."+col)
		}
		merge := fmt.Sprintf(
			"MERGE INTO %s AS target USING (SELECT %s) AS source ON %s WHEN MATCHED THEN UPDATE SET %s WHEN NOT MATCHED THEN INSERT (%s) VALUES (%s);",
			b.table,
			strings.Join(srcCols, ", "),
			strings.Join(onParts, " AND "),
			strings.Join(setParts, ", "),
			insertCols,
			strings.Join(insertVals, ", "),
		)
		result, err := b.db.Exec(b.rebind(merge), args...)
		if err != nil {
			return 0, err
		}
		return result.RowsAffected()
	default:
		// sqlite / postgres
		sets := make([]string, 0, len(updateCols))
		for _, col := range updateCols {
			sets = append(sets, fmt.Sprintf("%s = excluded.%s", col, col))
		}
		sb.WriteString(" ON CONFLICT (")
		sb.WriteString(strings.Join(uniqueBy, ", "))
		sb.WriteString(") DO UPDATE SET ")
		sb.WriteString(strings.Join(sets, ", "))
	}

	result, err := b.db.Exec(b.rebind(sb.String()), args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Update updates matching rows.
func (b *Builder) Update(values map[string]any) (int64, error) {
	if len(values) == 0 {
		return 0, fmt.Errorf("update values required")
	}
	sets := make([]string, 0, len(values))
	args := make([]any, 0)
	for column, value := range values {
		sets = append(sets, column+" = ?")
		args = append(args, value)
	}
	var sb strings.Builder
	sb.WriteString("UPDATE ")
	sb.WriteString(b.table)
	sb.WriteString(" SET ")
	sb.WriteString(strings.Join(sets, ", "))

	whereSQL, whereArgs := b.compileWheres(b.wheres)
	if whereSQL != "" {
		sb.WriteString(" WHERE ")
		sb.WriteString(whereSQL)
		args = append(args, whereArgs...)
	}

	result, err := b.db.Exec(b.rebind(sb.String()), args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Delete deletes matching rows.
func (b *Builder) Delete() (int64, error) {
	var sb strings.Builder
	sb.WriteString("DELETE FROM ")
	sb.WriteString(b.table)

	args := make([]any, 0)
	whereSQL, whereArgs := b.compileWheres(b.wheres)
	if whereSQL != "" {
		sb.WriteString(" WHERE ")
		sb.WriteString(whereSQL)
		args = append(args, whereArgs...)
	}

	result, err := b.db.Exec(b.rebind(sb.String()), args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Increment increments a column.
func (b *Builder) Increment(column string, amount ...int64) (int64, error) {
	n := int64(1)
	if len(amount) > 0 {
		n = amount[0]
	}
	return b.adjustColumn(column, n)
}

// Decrement decrements a column.
func (b *Builder) Decrement(column string, amount ...int64) (int64, error) {
	n := int64(1)
	if len(amount) > 0 {
		n = amount[0]
	}
	return b.adjustColumn(column, -n)
}

func (b *Builder) adjustColumn(column string, delta int64) (int64, error) {
	var sb strings.Builder
	sb.WriteString("UPDATE ")
	sb.WriteString(b.table)
	sb.WriteString(" SET ")
	if delta < 0 {
		sb.WriteString(fmt.Sprintf("%s = %s - %d", column, column, -delta))
	} else {
		sb.WriteString(fmt.Sprintf("%s = %s + %d", column, column, delta))
	}

	args := make([]any, 0)
	whereSQL, whereArgs := b.compileWheres(b.wheres)
	if whereSQL != "" {
		sb.WriteString(" WHERE ")
		sb.WriteString(whereSQL)
		args = append(args, whereArgs...)
	}

	result, err := b.db.Exec(b.rebind(sb.String()), args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Chunk iterates matching rows in batches of size, invoking callback for each batch.
// Stops early if callback returns an error. Orders by id when no order is set.
func (b *Builder) Chunk(size int, callback func([]map[string]any) error) error {
	if size < 1 {
		size = 100
	}
	base := b.clone()
	if len(base.orders) == 0 {
		base.OrderBy("id")
	}
	offset := 0
	for {
		clone := base.clone()
		clone.Limit(size).Offset(offset)
		rows, err := clone.Get()
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		if err := callback(rows); err != nil {
			return err
		}
		if len(rows) < size {
			return nil
		}
		offset += size
	}
}

// ChunkById iterates rows using keyset pagination on the id column (or custom column).
func (b *Builder) ChunkById(size int, callback func([]map[string]any) error, column ...string) error {
	if size < 1 {
		size = 100
	}
	col := "id"
	if len(column) > 0 && column[0] != "" {
		col = column[0]
	}
	var last any
	for {
		clone := b.clone()
		clone.orders = nil
		if last != nil {
			clone.Where(col, ">", last)
		}
		clone.OrderBy(col).Limit(size)
		rows, err := clone.Get()
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		if err := callback(rows); err != nil {
			return err
		}
		if len(rows) < size {
			return nil
		}
		last = rows[len(rows)-1][col]
		if last == nil {
			return nil
		}
	}
}

// InsertBatch inserts multiple rows sharing the same column set.
func (b *Builder) InsertBatch(rows []map[string]any) (int64, error) {
	if len(rows) == 0 {
		return 0, fmt.Errorf("insert batch values required")
	}
	columns := make([]string, 0, len(rows[0]))
	for column := range rows[0] {
		columns = append(columns, column)
	}
	sort.Strings(columns)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("INSERT INTO %s (%s) VALUES ", b.table, strings.Join(columns, ", ")))
	args := make([]any, 0, len(rows)*len(columns))
	for i, row := range rows {
		if i > 0 {
			sb.WriteString(", ")
		}
		placeholders := make([]string, len(columns))
		for j, column := range columns {
			placeholders[j] = "?"
			args = append(args, row[column])
		}
		sb.WriteString("(" + strings.Join(placeholders, ", ") + ")")
	}
	result, err := b.db.Exec(b.rebind(sb.String()), args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Truncate deletes all rows from the table.
func (b *Builder) Truncate() error {
	_, err := b.db.Exec(fmt.Sprintf("DELETE FROM %s", b.table))
	return err
}

// SimplePaginate returns a page of results without counting the total.
// Fetches one extra row to detect whether another page exists.
func (b *Builder) SimplePaginate(page, perPage int) (items []map[string]any, hasMore bool, err error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 15
	}
	clone := b.clone()
	clone.Limit(perPage + 1).Offset((page - 1) * perPage)
	rows, err := clone.Get()
	if err != nil {
		return nil, false, err
	}
	hasMore = len(rows) > perPage
	if hasMore {
		rows = rows[:perPage]
	}
	return rows, hasMore, nil
}

func (b *Builder) addWhereIn(boolean, column string, values []any, not bool) *Builder {
	col, err := sanitizeIdentifier(column)
	if err != nil {
		b.err = fmt.Errorf("where in: %w", err)
		return b
	}
	if len(values) == 0 {
		if not {
			return b
		}
		return b.addRawWhere(boolean, "0 = 1")
	}
	placeholders := make([]string, len(values))
	for i := range values {
		placeholders[i] = "?"
	}
	op := "IN"
	if not {
		op = "NOT IN"
	}
	return b.addRawWhere(boolean, fmt.Sprintf("%s %s (%s)", col, op, strings.Join(placeholders, ", ")), values...)
}

func (b *Builder) addWhere(boolean, column string, args ...any) *Builder {
	switch len(args) {
	case 0:
		return b.addRawWhere(boolean, column)
	case 1:
		col, err := sanitizeIdentifier(column)
		if err != nil {
			b.err = fmt.Errorf("where: %w", err)
			return b
		}
		return b.addRawWhere(boolean, col+" = ?", args[0])
	default:
		col, err := sanitizeIdentifier(column)
		if err != nil {
			b.err = fmt.Errorf("where: %w", err)
			return b
		}
		op, err := sanitizeOperator(fmt.Sprint(args[0]))
		if err != nil {
			b.err = fmt.Errorf("where: %w", err)
			return b
		}
		return b.addRawWhere(boolean, fmt.Sprintf("%s %s ?", col, op), args[1])
	}
}

func (b *Builder) addWhereMulti(outerBoolean, innerBoolean string, columns []string, args ...any) *Builder {
	if len(columns) == 0 {
		return b
	}
	parts := make([]string, 0, len(columns))
	allArgs := make([]any, 0, len(columns))
	for _, column := range columns {
		switch len(args) {
		case 0:
			parts = append(parts, column)
		case 1:
			parts = append(parts, column+" = ?")
			allArgs = append(allArgs, args[0])
		default:
			operator := fmt.Sprint(args[0])
			parts = append(parts, fmt.Sprintf("%s %s ?", column, operator))
			allArgs = append(allArgs, args[1])
		}
	}
	joined := strings.Join(parts, " "+strings.ToUpper(innerBoolean)+" ")
	return b.addRawWhere(outerBoolean, "("+joined+")", allArgs...)
}

func (b *Builder) addColumnWhere(boolean, first string, parts ...string) *Builder {
	operator, second := "=", ""
	switch len(parts) {
	case 1:
		second = parts[0]
	case 2:
		operator, second = parts[0], parts[1]
	default:
		return b
	}
	return b.addRawWhere(boolean, fmt.Sprintf("%s %s %s", first, operator, second))
}

func (b *Builder) addRawWhere(boolean, sqlStr string, args ...any) *Builder {
	b.wheres = append(b.wheres, whereClause{
		boolean: boolean,
		sql:     sqlStr,
		args:    args,
	})
	return b
}

func (b *Builder) compileWheres(clauses []whereClause) (string, []any) {
	if len(clauses) == 0 {
		return "", nil
	}
	parts := make([]string, 0, len(clauses))
	args := make([]any, 0)
	for i, clause := range clauses {
		part := clause.sql
		if i > 0 {
			part = strings.ToUpper(clause.boolean) + " " + clause.sql
		}
		parts = append(parts, part)
		args = append(args, clause.args...)
	}
	return strings.Join(parts, " "), args
}

func (b *Builder) rebind(sqlStr string) string {
	switch {
	case isPostgresDriver(b.driver):
		return rebindIndexed(sqlStr, "$%d")
	case isSQLServerDriver(b.driver):
		return rebindIndexed(sqlStr, "@p%d")
	case isOracleDriver(b.driver):
		return rebindIndexed(sqlStr, ":%d")
	default:
		return sqlStr
	}
}

func rebindIndexed(sqlStr, format string) string {
	var sb strings.Builder
	arg := 1
	for i := 0; i < len(sqlStr); i++ {
		if sqlStr[i] == '?' {
			sb.WriteString(fmt.Sprintf(format, arg))
			arg++
			continue
		}
		sb.WriteByte(sqlStr[i])
	}
	return sb.String()
}

func (b *Builder) compileLock() string {
	if b.lockMode == "" || strings.Contains(b.driver, "sqlite") {
		return ""
	}
	var clause string
	switch b.lockMode {
	case "update":
		clause = "FOR UPDATE"
	case "share":
		if b.driver == "pgsql" || b.driver == "postgres" || b.driver == "postgresql" || strings.Contains(b.driver, "mysql") {
			clause = "FOR SHARE"
		} else {
			clause = "LOCK IN SHARE MODE"
		}
	default:
		return ""
	}
	if b.lockSkip {
		clause += " SKIP LOCKED"
	} else if b.lockNoWait {
		clause += " NOWAIT"
	}
	return " " + clause
}

func (b *Builder) clone() *Builder {
	cp := *b
	cp.columns = append([]string{}, b.columns...)
	cp.selectBindings = append([]any{}, b.selectBindings...)
	cp.wheres = append([]whereClause{}, b.wheres...)
	cp.orders = append([]orderClause{}, b.orders...)
	cp.groups = append([]string{}, b.groups...)
	cp.havings = append([]whereClause{}, b.havings...)
	cp.joins = append([]string{}, b.joins...)
	cp.unions = make([]unionClause, len(b.unions))
	for i, u := range b.unions {
		cp.unions[i] = unionClause{all: u.all, query: u.query.clone()}
	}
	return &cp
}

func scanRows(rows *sql.Rows) ([]map[string]any, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	results := make([]map[string]any, 0)
	for rows.Next() {
		values := make([]any, len(columns))
		pointers := make([]any, len(columns))
		for i := range values {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			return nil, err
		}
		row := make(map[string]any, len(columns))
		for i, column := range columns {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[column] = string(b)
			} else {
				row[column] = val
			}
		}
		results = append(results, row)
	}
	return results, rows.Err()
}

func isPostgresDriver(driver string) bool {
	switch strings.ToLower(driver) {
	case "pgsql", "postgres", "postgresql":
		return true
	default:
		return false
	}
}

func isSQLServerDriver(driver string) bool {
	switch strings.ToLower(driver) {
	case "mssql", "sqlserver":
		return true
	default:
		return false
	}
}

func isOracleDriver(driver string) bool {
	switch strings.ToLower(driver) {
	case "oracle", "ora":
		return true
	default:
		return false
	}
}
