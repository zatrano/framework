package schema

import (
	"database/sql"
	"fmt"
	"strings"
)

// Builder builds and executes schema statements.
type Builder struct {
	db     *sql.DB
	driver string
}

// New creates a schema builder.
func New(db *sql.DB, driver string) *Builder {
	return &Builder{db: db, driver: driver}
}

// Create creates a table.
func (b *Builder) Create(table string, callback func(*Blueprint)) error {
	bp := NewBlueprint(table, b.driver)
	callback(bp)
	sqlStr, err := bp.ToCreateSQL()
	if err != nil {
		return err
	}
	_, err = b.db.Exec(sqlStr)
	return err
}

// Table alters a table.
func (b *Builder) Table(table string, callback func(*Blueprint)) error {
	bp := NewBlueprint(table, b.driver)
	bp.altering = true
	callback(bp)
	statements, err := bp.ToAlterSQL()
	if err != nil {
		return err
	}
	for _, sqlStr := range statements {
		if _, err := b.db.Exec(sqlStr); err != nil {
			return err
		}
	}
	return nil
}

// Drop drops a table.
func (b *Builder) Drop(table string) error {
	_, err := b.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s", table))
	return err
}

// DropIfExists drops a table if it exists.
func (b *Builder) DropIfExists(table string) error {
	return b.Drop(table)
}

// Rename renames a table.
func (b *Builder) Rename(from, to string) error {
	switch b.driver {
	case "mysql":
		_, err := b.db.Exec(fmt.Sprintf("RENAME TABLE %s TO %s", from, to))
		return err
	case "mssql", "sqlserver":
		_, err := b.db.Exec(fmt.Sprintf("EXEC sp_rename '%s', '%s'", from, to))
		return err
	case "sqlite", "sqlite3", "pgsql", "postgres", "postgresql":
		_, err := b.db.Exec(fmt.Sprintf("ALTER TABLE %s RENAME TO %s", from, to))
		return err
	default:
		_, err := b.db.Exec(fmt.Sprintf("ALTER TABLE %s RENAME TO %s", from, to))
		return err
	}
}

// HasTable reports whether a table exists.
func (b *Builder) HasTable(table string) (bool, error) {
	var query string
	switch b.driver {
	case "sqlite", "sqlite3":
		query = `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`
	case "mysql":
		query = `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name = ?`
	case "pgsql", "postgres", "postgresql":
		query = `SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name = $1`
	case "mssql", "sqlserver":
		query = `SELECT COUNT(*) FROM information_schema.tables WHERE table_name = @p1`
	default:
		return false, fmt.Errorf("unsupported driver: %s", b.driver)
	}

	var count int
	err := b.db.QueryRow(query, table).Scan(&count)
	return count > 0, err
}

// HasColumn reports whether a column exists on a table.
func (b *Builder) HasColumn(table, column string) (bool, error) {
	var query string
	switch b.driver {
	case "sqlite", "sqlite3":
		rows, err := b.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
		if err != nil {
			return false, err
		}
		defer rows.Close()
		for rows.Next() {
			var cid int
			var name, ctype string
			var notnull, pk int
			var dflt sql.NullString
			if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
				return false, err
			}
			if strings.EqualFold(name, column) {
				return true, nil
			}
		}
		return false, rows.Err()
	case "mysql":
		query = `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = ? AND column_name = ?`
	case "pgsql", "postgres", "postgresql":
		query = `SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = 'public' AND table_name = $1 AND column_name = $2`
	case "mssql", "sqlserver":
		query = `SELECT COUNT(*) FROM information_schema.columns WHERE table_name = @p1 AND column_name = @p2`
	default:
		return false, fmt.Errorf("unsupported driver: %s", b.driver)
	}
	var count int
	err := b.db.QueryRow(query, table, column).Scan(&count)
	return count > 0, err
}

// Blueprint describes a table definition.
type Blueprint struct {
	table    string
	driver   string
	columns  []*Column
	altering bool
}

// Column describes a table column.
type Column struct {
	Name          string
	Type          string
	Length        int
	IsNullable    bool
	Primary       bool
	AutoIncrement bool
	IsUnique      bool
	DefaultValue  any
	Comment       string
	Change        string // add|modify|drop
	References    string // referenced table for inline FK (CREATE TABLE)
	OnDelete      string // e.g. CASCADE
}

// NewBlueprint creates a blueprint.
func NewBlueprint(table, driver string) *Blueprint {
	return &Blueprint{table: table, driver: driver}
}

// ID adds a big integer auto-incrementing primary key named id.
func (b *Blueprint) ID(name ...string) *Column {
	colName := "id"
	if len(name) > 0 && name[0] != "" {
		colName = name[0]
	}
	col := &Column{
		Name:          colName,
		Type:          "id",
		Primary:       true,
		AutoIncrement: true,
		Change:        "add",
	}
	b.columns = append(b.columns, col)
	return col
}

// BigIncrements adds an auto-incrementing unsigned big integer.
func (b *Blueprint) BigIncrements(name string) *Column {
	return b.ID(name)
}

// String adds a varchar column.
func (b *Blueprint) String(name string, length ...int) *Column {
	l := 255
	if len(length) > 0 {
		l = length[0]
	}
	col := &Column{Name: name, Type: "string", Length: l, Change: "add"}
	b.columns = append(b.columns, col)
	return col
}

// Text adds a text column.
func (b *Blueprint) Text(name string) *Column {
	col := &Column{Name: name, Type: "text", Change: "add"}
	b.columns = append(b.columns, col)
	return col
}

// Integer adds an integer column.
func (b *Blueprint) Integer(name string) *Column {
	col := &Column{Name: name, Type: "integer", Change: "add"}
	b.columns = append(b.columns, col)
	return col
}

// BigInteger adds a big integer column.
func (b *Blueprint) BigInteger(name string) *Column {
	col := &Column{Name: name, Type: "biginteger", Change: "add"}
	b.columns = append(b.columns, col)
	return col
}

// Boolean adds a boolean column.
func (b *Blueprint) Boolean(name string) *Column {
	col := &Column{Name: name, Type: "boolean", Change: "add"}
	b.columns = append(b.columns, col)
	return col
}

// Timestamp adds a timestamp column.
func (b *Blueprint) Timestamp(name string) *Column {
	col := &Column{Name: name, Type: "timestamp", IsNullable: true, Change: "add"}
	b.columns = append(b.columns, col)
	return col
}

// Timestamps adds created_at and updated_at.
func (b *Blueprint) Timestamps() {
	b.Timestamp("created_at")
	b.Timestamp("updated_at")
}

// SoftDeletes adds deleted_at.
func (b *Blueprint) SoftDeletes() *Column {
	return b.Timestamp("deleted_at")
}

// Decimal adds a decimal column.
func (b *Blueprint) Decimal(name string, precision, scale int) *Column {
	col := &Column{
		Name:   name,
		Type:   fmt.Sprintf("decimal:%d:%d", precision, scale),
		Change: "add",
	}
	b.columns = append(b.columns, col)
	return col
}

// JSON adds a JSON column.
func (b *Blueprint) JSON(name string) *Column {
	col := &Column{Name: name, Type: "json", Change: "add"}
	b.columns = append(b.columns, col)
	return col
}

// ForeignID adds an unsigned big integer foreign id column (matches ID() on MySQL).
// Use Constrained(table) to append REFERENCES on CREATE TABLE; CascadeOnDelete sets ON DELETE CASCADE.
func (b *Blueprint) ForeignID(name string) *Column {
	col := &Column{Name: name, Type: "foreignid", Change: "add"}
	b.columns = append(b.columns, col)
	return col
}

// Constrained registers an inline foreign key REFERENCES clause for Create (references table.id).
func (c *Column) Constrained(table string) *Column {
	c.References = table
	return c
}

// CascadeOnDelete sets ON DELETE CASCADE on the column's REFERENCES clause.
func (c *Column) CascadeOnDelete() *Column {
	c.OnDelete = "CASCADE"
	return c
}

// DropColumn drops columns (alter mode).
func (b *Blueprint) DropColumn(columns ...string) {
	for _, column := range columns {
		b.columns = append(b.columns, &Column{Name: column, Change: "drop"})
	}
}

// Unique marks the column as unique.
func (c *Column) Unique() *Column {
	c.IsUnique = true
	return c
}

// Nullable marks the column as nullable.
func (c *Column) Nullable() *Column {
	c.IsNullable = true
	return c
}

// Default sets a default value.
func (c *Column) Default(value any) *Column {
	c.DefaultValue = value
	return c
}

// ToCreateSQL compiles CREATE TABLE SQL.
func (b *Blueprint) ToCreateSQL() (string, error) {
	if len(b.columns) == 0 {
		return "", fmt.Errorf("no columns defined for table %s", b.table)
	}
	defs := make([]string, 0, len(b.columns))
	for _, col := range b.columns {
		if col.Change == "drop" {
			continue
		}
		def, err := b.columnSQL(col)
		if err != nil {
			return "", err
		}
		defs = append(defs, def)
	}
	return fmt.Sprintf("CREATE TABLE %s (%s)", b.table, strings.Join(defs, ", ")), nil
}

// ToAlterSQL compiles ALTER TABLE statements.
func (b *Blueprint) ToAlterSQL() ([]string, error) {
	statements := make([]string, 0)
	for _, col := range b.columns {
		switch col.Change {
		case "drop":
			statements = append(statements, fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s", b.table, col.Name))
		default:
			def, err := b.columnSQL(col)
			if err != nil {
				return nil, err
			}
			add := "ADD"
			switch b.driver {
			case "sqlite", "sqlite3":
				add = "ADD COLUMN"
			case "mssql", "sqlserver":
				add = "ADD"
			}
			statements = append(statements, fmt.Sprintf("ALTER TABLE %s %s %s", b.table, add, def))
		}
	}
	return statements, nil
}

func (b *Blueprint) columnSQL(col *Column) (string, error) {
	typeSQL, err := b.typeSQL(col)
	if err != nil {
		return "", err
	}

	parts := []string{col.Name, typeSQL}

	if col.AutoIncrement {
		switch b.driver {
		case "mysql":
			parts = append(parts, "AUTO_INCREMENT")
		case "sqlite", "sqlite3":
			// INTEGER PRIMARY KEY is autoincrement in SQLite.
		case "pgsql", "postgres", "postgresql":
			// SERIAL handled in typeSQL
		case "mssql", "sqlserver":
			// IDENTITY handled in typeSQL
		}
	}

	if !col.IsNullable && !col.Primary {
		parts = append(parts, "NOT NULL")
	}
	if col.Primary {
		parts = append(parts, "PRIMARY KEY")
	}
	if col.IsUnique {
		parts = append(parts, "UNIQUE")
	}
	if col.DefaultValue != nil {
		parts = append(parts, fmt.Sprintf("DEFAULT %s", formatDefault(col.DefaultValue)))
	}
	if col.References != "" {
		ref := fmt.Sprintf("REFERENCES %s(id)", col.References)
		if col.OnDelete != "" {
			ref += " ON DELETE " + col.OnDelete
		}
		parts = append(parts, ref)
	}

	return strings.Join(parts, " "), nil
}

func (b *Blueprint) typeSQL(col *Column) (string, error) {
	switch col.Type {
	case "id":
		switch b.driver {
		case "mysql":
			return "BIGINT UNSIGNED", nil
		case "pgsql", "postgres", "postgresql":
			return "BIGSERIAL", nil
		case "mssql", "sqlserver":
			return "BIGINT IDENTITY(1,1)", nil
		default:
			return "INTEGER", nil
		}
	case "foreignid":
		switch b.driver {
		case "mysql":
			return "BIGINT UNSIGNED", nil
		default:
			return "BIGINT", nil
		}
	case "string":
		return fmt.Sprintf("VARCHAR(%d)", col.Length), nil
	case "text":
		switch b.driver {
		case "mssql", "sqlserver":
			return "NVARCHAR(MAX)", nil
		default:
			return "TEXT", nil
		}
	case "integer":
		return "INTEGER", nil
	case "biginteger":
		return "BIGINT", nil
	case "boolean":
		switch b.driver {
		case "mysql":
			return "TINYINT(1)", nil
		case "mssql", "sqlserver":
			return "BIT", nil
		default:
			return "BOOLEAN", nil
		}
	case "timestamp":
		switch b.driver {
		case "pgsql", "postgres", "postgresql":
			return "TIMESTAMP", nil
		case "mssql", "sqlserver":
			return "DATETIME2", nil
		default:
			return "DATETIME", nil
		}
	case "json":
		switch b.driver {
		case "pgsql", "postgres", "postgresql":
			return "JSONB", nil
		case "mysql":
			return "JSON", nil
		case "mssql", "sqlserver":
			return "NVARCHAR(MAX)", nil
		default:
			return "TEXT", nil
		}
	default:
		if strings.HasPrefix(col.Type, "decimal:") {
			parts := strings.Split(col.Type, ":")
			return fmt.Sprintf("DECIMAL(%s,%s)", parts[1], parts[2]), nil
		}
		return "", fmt.Errorf("unknown column type: %s", col.Type)
	}
}

func formatDefault(value any) string {
	switch v := value.(type) {
	case string:
		return fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "''"))
	case bool:
		if v {
			return "TRUE"
		}
		return "FALSE"
	default:
		return fmt.Sprint(v)
	}
}
