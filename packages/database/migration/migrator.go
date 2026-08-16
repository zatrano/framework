package migration

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/zatrano/framework/packages/database/schema"
)

// Migration is a database migration.
type Migration interface {
	Name() string
	Up(schema *schema.Builder) error
	Down(schema *schema.Builder) error
}

// Migrator runs migrations.
type Migrator struct {
	db         *sql.DB
	driver     string
	schema     *schema.Builder
	repository *Repository
	migrations []Migration
}

// NewMigrator creates a migrator.
func NewMigrator(db *sql.DB, driver string, migrations []Migration) *Migrator {
	return &Migrator{
		db:         db,
		driver:     driver,
		schema:     schema.New(db, driver),
		repository: NewRepository(db, driver),
		migrations: migrations,
	}
}

// Migrate runs outstanding migrations.
func (m *Migrator) Migrate() error {
	if err := m.repository.CreateRepository(); err != nil {
		return err
	}

	ran, err := m.repository.Ran()
	if err != nil {
		return err
	}
	ranSet := make(map[string]bool, len(ran))
	for _, name := range ran {
		ranSet[name] = true
	}

	batch, err := m.repository.LastBatch()
	if err != nil {
		return err
	}
	batch++

	pending := 0
	for _, migration := range m.migrations {
		if ranSet[migration.Name()] {
			continue
		}
		fmt.Printf("Migrating: %s\n", migration.Name())
		if err := migration.Up(m.schema); err != nil {
			return fmt.Errorf("%s: %w", migration.Name(), err)
		}
		if err := m.repository.Log(migration.Name(), batch); err != nil {
			return err
		}
		fmt.Printf("Migrated:  %s\n", migration.Name())
		pending++
	}

	if pending == 0 {
		fmt.Println("Nothing to migrate.")
	}
	return nil
}

// Rollback rolls back the last batch.
func (m *Migrator) Rollback() error {
	if err := m.repository.CreateRepository(); err != nil {
		return err
	}
	batch, err := m.repository.LastBatch()
	if err != nil {
		return err
	}
	if batch == 0 {
		fmt.Println("Nothing to rollback.")
		return nil
	}

	names, err := m.repository.GetBatch(batch)
	if err != nil {
		return err
	}

	lookup := make(map[string]Migration, len(m.migrations))
	for _, migration := range m.migrations {
		lookup[migration.Name()] = migration
	}

	for i := len(names) - 1; i >= 0; i-- {
		name := names[i]
		migration, ok := lookup[name]
		if !ok {
			return fmt.Errorf("migration [%s] not found", name)
		}
		fmt.Printf("Rolling back: %s\n", name)
		if err := migration.Down(m.schema); err != nil {
			return err
		}
		if err := m.repository.Delete(name); err != nil {
			return err
		}
		fmt.Printf("Rolled back:  %s\n", name)
	}
	return nil
}

// Fresh drops all tables and re-runs migrations.
func (m *Migrator) Fresh() error {
	if err := m.repository.CreateRepository(); err != nil {
		return err
	}
	ran, err := m.repository.Ran()
	if err != nil {
		return err
	}
	lookup := make(map[string]Migration, len(m.migrations))
	for _, migration := range m.migrations {
		lookup[migration.Name()] = migration
	}
	for i := len(ran) - 1; i >= 0; i-- {
		if migration, ok := lookup[ran[i]]; ok {
			_ = migration.Down(m.schema)
		}
	}
	_ = m.schema.DropIfExists("migrations")
	return m.Migrate()
}

// Status prints migration status.
func (m *Migrator) Status() error {
	if err := m.repository.CreateRepository(); err != nil {
		return err
	}
	ran, err := m.repository.Ran()
	if err != nil {
		return err
	}
	ranSet := make(map[string]bool, len(ran))
	for _, name := range ran {
		ranSet[name] = true
	}

	fmt.Println("Migration name\tBatch\tStatus")
	batches, _ := m.repository.Batches()
	for _, migration := range m.migrations {
		status := "Pending"
		batch := 0
		if ranSet[migration.Name()] {
			status = "Ran"
			batch = batches[migration.Name()]
		}
		fmt.Printf("%s\t%d\t%s\n", migration.Name(), batch, status)
	}
	return nil
}

// Repository stores migration history.
type Repository struct {
	db     *sql.DB
	driver string
}

// NewRepository creates a migration repository.
func NewRepository(db *sql.DB, driver string) *Repository {
	return &Repository{db: db, driver: driver}
}

// CreateRepository creates the migrations table.
func (r *Repository) CreateRepository() error {
	var sqlStr string
	switch r.driver {
	case "mysql":
		sqlStr = `CREATE TABLE IF NOT EXISTS migrations (
			id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
			migration VARCHAR(255) NOT NULL,
			batch INT NOT NULL
		)`
	case "pgsql", "postgres", "postgresql":
		sqlStr = `CREATE TABLE IF NOT EXISTS migrations (
			id BIGSERIAL PRIMARY KEY,
			migration VARCHAR(255) NOT NULL,
			batch INT NOT NULL
		)`
	case "mssql", "sqlserver":
		sqlStr = `IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = 'migrations')
		CREATE TABLE migrations (
			id BIGINT IDENTITY(1,1) PRIMARY KEY,
			migration VARCHAR(255) NOT NULL,
			batch INT NOT NULL
		)`
	default:
		sqlStr = `CREATE TABLE IF NOT EXISTS migrations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			migration VARCHAR(255) NOT NULL,
			batch INTEGER NOT NULL
		)`
	}
	_, err := r.db.Exec(sqlStr)
	return err
}

// Ran returns ran migration names in order.
func (r *Repository) Ran() ([]string, error) {
	rows, err := r.db.Query(`SELECT migration FROM migrations ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	names := make([]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// LastBatch returns the last batch number.
func (r *Repository) LastBatch() (int, error) {
	var batch sql.NullInt64
	err := r.db.QueryRow(`SELECT MAX(batch) FROM migrations`).Scan(&batch)
	if err != nil {
		return 0, err
	}
	if !batch.Valid {
		return 0, nil
	}
	return int(batch.Int64), nil
}

// GetBatch returns migrations in a batch.
func (r *Repository) GetBatch(batch int) ([]string, error) {
	rows, err := r.db.Query(r.q(`SELECT migration FROM migrations WHERE batch = ? ORDER BY id ASC`), batch)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	names := make([]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}

// Batches returns migration name to batch map.
func (r *Repository) Batches() (map[string]int, error) {
	rows, err := r.db.Query(`SELECT migration, batch FROM migrations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]int)
	for rows.Next() {
		var name string
		var batch int
		if err := rows.Scan(&name, &batch); err != nil {
			return nil, err
		}
		out[name] = batch
	}
	return out, rows.Err()
}

// Log records a migration.
func (r *Repository) Log(name string, batch int) error {
	_, err := r.db.Exec(r.q(`INSERT INTO migrations (migration, batch) VALUES (?, ?)`), name, batch)
	return err
}

// Delete removes a migration record.
func (r *Repository) Delete(name string) error {
	_, err := r.db.Exec(r.q(`DELETE FROM migrations WHERE migration = ?`), name)
	return err
}

// q rewrites ? placeholders for PostgreSQL ($1…) and SQL Server (@p1…).
func (r *Repository) q(query string) string {
	switch r.driver {
	case "pgsql", "postgres", "postgresql":
		return rebindPlaceholders(query, "$%d")
	case "mssql", "sqlserver":
		return rebindPlaceholders(query, "@p%d")
	default:
		return query
	}
}

func rebindPlaceholders(query, format string) string {
	out := make([]byte, 0, len(query)+8)
	n := 1
	for i := 0; i < len(query); i++ {
		if query[i] == '?' {
			out = append(out, fmt.Sprintf(format, n)...)
			n++
			continue
		}
		out = append(out, query[i])
	}
	return string(out)
}

// GenerateName creates a timestamped migration name.
func GenerateName(description string) string {
	stamp := time.Now().Format("20060102_150405")
	return stamp + "_" + description
}
