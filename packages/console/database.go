package console

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zatrano/framework/kernel"
	"github.com/zatrano/framework/packages/database"
	"github.com/zatrano/framework/packages/env"
)

func registerDatabaseCommands(console *Application, app *kernel.Application) {
	console.Register(
		&MigrateCommand{app: app},
		&MigrateRollbackCommand{app: app},
		&MigrateStatusCommand{app: app},
		&MigrateFreshCommand{app: app},
		&DBCreateCommand{app: app},
		&DBSeedCommand{app: app},
		&MakeModelCommand{app: app},
		&MakeMigrationCommand{app: app},
		&MakeSeederCommand{app: app},
	)
	registerDBSetupCommand(console, app)
}

type MigrateCommand struct {
	app *kernel.Application
}

func (c *MigrateCommand) Name() string        { return "migrate" }
func (c *MigrateCommand) Description() string { return "Run the database migrations" }
func (c *MigrateCommand) Handle(args []string) error {
	if err := c.app.Bootstrap(); err != nil {
		return err
	}
	migrator, err := database.Migrator(c.app)
	if err != nil {
		return err
	}
	return migrator.Migrate()
}

type MigrateRollbackCommand struct {
	app *kernel.Application
}

func (c *MigrateRollbackCommand) Name() string        { return "migrate:rollback" }
func (c *MigrateRollbackCommand) Description() string { return "Rollback the last database migration" }
func (c *MigrateRollbackCommand) Handle(args []string) error {
	if err := c.app.Bootstrap(); err != nil {
		return err
	}
	migrator, err := database.Migrator(c.app)
	if err != nil {
		return err
	}
	return migrator.Rollback()
}

type MigrateStatusCommand struct {
	app *kernel.Application
}

func (c *MigrateStatusCommand) Name() string        { return "migrate:status" }
func (c *MigrateStatusCommand) Description() string { return "Show the status of each migration" }
func (c *MigrateStatusCommand) Handle(args []string) error {
	if err := c.app.Bootstrap(); err != nil {
		return err
	}
	migrator, err := database.Migrator(c.app)
	if err != nil {
		return err
	}
	return migrator.Status()
}

type MigrateFreshCommand struct {
	app *kernel.Application
}

func (c *MigrateFreshCommand) Name() string { return "migrate:fresh" }
func (c *MigrateFreshCommand) Description() string {
	return "Drop all tables and re-run all migrations"
}
func (c *MigrateFreshCommand) Handle(args []string) error {
	if err := c.app.Bootstrap(); err != nil {
		return err
	}
	migrator, err := database.Migrator(c.app)
	if err != nil {
		return err
	}
	return migrator.Fresh()
}

// DBCreateCommand creates the configured MySQL/PostgreSQL database.
type DBCreateCommand struct {
	app *kernel.Application
}

func (c *DBCreateCommand) Name() string { return "db:create" }
func (c *DBCreateCommand) Description() string {
	return "Create the application database (MySQL/PostgreSQL/SQL Server/Oracle)"
}
func (c *DBCreateCommand) Handle(args []string) error {
	_ = env.Load(c.app.BasePath(".env"))

	driver := strings.ToLower(strings.TrimSpace(env.Get("DB_CONNECTION")))
	if driver == "" {
		return fmt.Errorf("no database configured; run db:setup --drivers=sqlite (or mysql, pgsql, …)")
	}
	switch database.NormalizeDriverName(driver) {
	case "sqlite":
		fmt.Println("SQLite does not require db:create; the database file is created on first connection.")
		return nil
	case "mysql":
		return createMySQLDatabase()
	case "pgsql":
		return createPostgresDatabase()
	case "mssql":
		return createMSSQLDatabase()
	case "oracle":
		return createOracleDatabase()
	default:
		return fmt.Errorf("db:create does not support driver [%s]", driver)
	}
}

func createMySQLDatabase() error {
	name := env.GetNonEmpty("DB_DATABASE", "zatrano")
	if name == "" {
		return fmt.Errorf("DB_DATABASE is empty")
	}
	host := env.Get("DB_HOST", "127.0.0.1")
	port := env.GetNonEmpty("DB_PORT", "3306")
	user := env.GetNonEmpty("DB_USERNAME", "root")
	pass := env.Get("DB_PASSWORD", "")
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/?parseTime=true&loc=Local", user, pass, host, port)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		return err
	}
	quoted := "`" + strings.ReplaceAll(name, "`", "``") + "`"
	if _, err := db.Exec("CREATE DATABASE IF NOT EXISTS " + quoted); err != nil {
		return err
	}
	fmt.Printf("Database [%s] created (or already exists).\n", name)
	return nil
}

func createPostgresDatabase() error {
	name := env.GetNonEmpty("DB_DATABASE", "zatrano")
	if name == "" {
		return fmt.Errorf("DB_DATABASE is empty")
	}
	host := env.Get("DB_HOST", "127.0.0.1")
	port := env.GetNonEmpty("DB_PORT", "5432")
	user := env.GetNonEmpty("DB_USERNAME", "postgres")
	pass := env.Get("DB_PASSWORD", "")
	ssl := env.GetNonEmpty("DB_SSLMODE", "disable")
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=%s", host, port, user, pass, ssl)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		return err
	}
	var exists bool
	if err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", name).Scan(&exists); err != nil {
		return err
	}
	if exists {
		fmt.Printf("Database [%s] already exists.\n", name)
		return nil
	}
	if strings.ContainsAny(name, "\"';\\") {
		return fmt.Errorf("invalid database name [%s]", name)
	}
	if _, err := db.Exec(`CREATE DATABASE "` + name + `"`); err != nil {
		return err
	}
	fmt.Printf("Database [%s] created.\n", name)
	return nil
}

func createMSSQLDatabase() error {
	name := env.GetNonEmpty("DB_DATABASE", "zatrano")
	if name == "" {
		return fmt.Errorf("DB_DATABASE is empty")
	}
	if strings.ContainsAny(name, "';\\[]") {
		return fmt.Errorf("invalid database name [%s]", name)
	}
	host := env.Get("DB_HOST", "127.0.0.1")
	port := env.GetNonEmpty("DB_PORT", "1433")
	user := env.GetNonEmpty("DB_USERNAME", "sa")
	pass := env.Get("DB_PASSWORD", "")
	dsn := fmt.Sprintf("sqlserver://%s:%s@%s:%s?database=master",
		url.QueryEscape(user), url.QueryEscape(pass), host, port)
	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		return err
	}
	var exists int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sys.databases WHERE name = @p1`, name).Scan(&exists); err != nil {
		return err
	}
	if exists > 0 {
		fmt.Printf("Database [%s] already exists.\n", name)
		return nil
	}
	if _, err := db.Exec("CREATE DATABASE [" + name + "]"); err != nil {
		return err
	}
	fmt.Printf("Database [%s] created.\n", name)
	return nil
}

func createOracleDatabase() error {
	fmt.Println("Oracle: create the pluggable/service database with your DBA tools (db:create is a no-op).")
	fmt.Println("Set DB_HOST, DB_PORT=1521, DB_SERVICE (or DB_DATABASE), DB_USERNAME, DB_PASSWORD.")
	return nil
}

type DBSeedCommand struct {
	app *kernel.Application
}

func (c *DBSeedCommand) Name() string        { return "db:seed" }
func (c *DBSeedCommand) Description() string { return "Seed the database with records" }
func (c *DBSeedCommand) Handle(args []string) error {
	runner, err := database.SeederRunner(c.app)
	if err != nil {
		return err
	}
	if err := runner.Call(); err != nil {
		return err
	}
	fmt.Println("Database seeding completed successfully.")
	return nil
}

type MakeModelCommand struct {
	app *kernel.Application
}

func (c *MakeModelCommand) Name() string { return "make:model" }
func (c *MakeModelCommand) Description() string {
	return "Create a new ORM model (--connection=, --translation, -m)"
}
func (c *MakeModelCommand) Handle(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("model name required")
	}
	name := args[0]
	withMigration := false
	translation := ""
	connection := ""
	for _, arg := range args[1:] {
		switch {
		case arg == "-m" || arg == "--migration":
			withMigration = true
		case arg == "--translation" || arg == "-t":
			translation = "columns"
		case strings.HasPrefix(arg, "--translation="):
			mode := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(arg, "--translation=")))
			switch mode {
			case "", "columns", "true", "1":
				translation = "columns"
			case "json":
				translation = "json"
			default:
				return fmt.Errorf("unknown --translation mode %q (want columns|json)", mode)
			}
		case strings.HasPrefix(arg, "--connection="):
			connection = database.NormalizeDriverName(strings.TrimPrefix(arg, "--connection="))
		case arg == "--connection" || arg == "-c":
			return fmt.Errorf("use --connection=pgsql (value required)")
		}
	}

	dir := c.app.BasePath("app", "models")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, toSnake(name)+".go")
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("model already exists: %s", path)
	}

	table := toSnake(pluralize(name))
	content := modelStub(name, table, translation, connection)

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("Model created: %s\n", path)

	if withMigration {
		return writeModelMigration(c.app, name, table, translation)
	}
	return nil
}

func modelStub(name, table, translation, connection string) string {
	connMethod := ""
	if connection != "" {
		connMethod = fmt.Sprintf(`

func (m *%s) Connection() string {
	return %q
}
`, name, connection)
	}
	switch translation {
	case "json":
		return fmt.Sprintf(`package models

import "github.com/zatrano/framework/packages/orm"

type %s struct {
	orm.Model
	Translations map[string]string `+"`"+`db:"translations" json:"translations"`+"`"+`
}

func (m *%s) TableName() string {
	return "%s"
}

func (m *%s) Casts() map[string]string {
	return map[string]string{"translations": "json"}
}
%s`, name, name, table, name, connMethod)
	case "columns":
		return fmt.Sprintf(`package models

import "github.com/zatrano/framework/packages/orm"

type %s struct {
	orm.Model
	NameTr string `+"`"+`db:"name_tr" json:"name_tr"`+"`"+`
	NameEn string `+"`"+`db:"name_en" json:"name_en"`+"`"+`
}

func (m *%s) TableName() string {
	return "%s"
}
%s`, name, name, table, connMethod)
	default:
		return fmt.Sprintf(`package models

import "github.com/zatrano/framework/packages/orm"

type %s struct {
	orm.Model
	Name string `+"`"+`db:"name" json:"name"`+"`"+`
}

func (m *%s) TableName() string {
	return "%s"
}
%s`, name, name, table, connMethod)
	}
}

func writeModelMigration(app *kernel.Application, modelName, table, translation string) error {
	description := "create_" + table + "_table"
	stamp := time.Now().Format("20060102_150405")
	structName := toExported(description)
	fileName := stamp + "_" + description + ".go"

	dir := app.BasePath("database", "migrations")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, fileName)

	columns := `		table.String("name")`
	switch translation {
	case "json":
		columns = `		table.Text("translations")`
	case "columns":
		columns = `		table.String("name_tr")
		table.String("name_en")`
	}

	content := fmt.Sprintf(`package migrations

import "github.com/zatrano/framework/packages/database/schema"

type %s struct{}

func (m *%s) Name() string {
	return "%s_%s"
}

func (m *%s) Up(s *schema.Builder) error {
	return s.Create("%s", func(table *schema.Blueprint) {
		table.ID()
%s
		table.Timestamps()
	})
}

func (m *%s) Down(s *schema.Builder) error {
	return s.DropIfExists("%s")
}
`, structName, structName, stamp, description, structName, table, columns, structName, table)

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("Migration created: %s\n", path)
	regPath := filepath.Join(dir, "migrations.go")
	if registered, err := appendToAllSlice(regPath, "&"+structName+"{}"); err != nil {
		return err
	} else if registered {
		fmt.Printf("Registered in %s\n", regPath)
	} else {
		fmt.Println("Remember to register it in database/migrations/migrations.go")
	}
	_ = modelName
	return nil
}

type MakeMigrationCommand struct {
	app *kernel.Application
}

func (c *MakeMigrationCommand) Name() string        { return "make:migration" }
func (c *MakeMigrationCommand) Description() string { return "Create a new migration file" }
func (c *MakeMigrationCommand) Handle(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("migration name required")
	}
	description := toSnake(args[0])
	stamp := time.Now().Format("20060102_150405")
	structName := toExported(description)
	fileName := stamp + "_" + description + ".go"

	dir := c.app.BasePath("database", "migrations")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, fileName)
	content := fmt.Sprintf(`package migrations

import "github.com/zatrano/framework/packages/database/schema"

type %s struct{}

func (m *%s) Name() string {
	return "%s_%s"
}

func (m *%s) Up(s *schema.Builder) error {
	return s.Create("table_name", func(table *schema.Blueprint) {
		table.ID()
		table.Timestamps()
	})
}

func (m *%s) Down(s *schema.Builder) error {
	return s.DropIfExists("table_name")
}
`, structName, structName, stamp, description, structName, structName)

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("Migration created: %s\n", path)
	regPath := filepath.Join(dir, "migrations.go")
	if registered, err := appendToAllSlice(regPath, "&"+structName+"{}"); err != nil {
		return err
	} else if registered {
		fmt.Printf("Registered in %s\n", regPath)
	} else {
		fmt.Println("Remember to register it in database/migrations/migrations.go")
	}
	return nil
}

type MakeSeederCommand struct {
	app *kernel.Application
}

func (c *MakeSeederCommand) Name() string        { return "make:seeder" }
func (c *MakeSeederCommand) Description() string { return "Create a new seeder" }
func (c *MakeSeederCommand) Handle(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("seeder name required")
	}
	name := args[0]
	if !strings.HasSuffix(name, "Seeder") {
		name += "Seeder"
	}
	dir := c.app.BasePath("database", "seeders")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, toSnake(name)+".go")
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("seeder already exists: %s", path)
	}
	content := fmt.Sprintf(`package seeders

import (
	"fmt"
)

type %s struct{}

func (s *%s) Run() error {
	fmt.Println("Running %s...")
	return nil
}
`, name, name, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("Seeder created: %s\n", path)
	regPath := filepath.Join(dir, "database_seeder.go")
	if registered, err := appendToAllSlice(regPath, "&"+name+"{}"); err != nil {
		return err
	} else if registered {
		fmt.Printf("Registered in %s\n", regPath)
	} else {
		fmt.Println("Remember to register it in database/seeders/database_seeder.go")
	}
	return nil
}

// appendToAllSlice inserts entry into a file's `func All()` return slice when present.
// Returns true when the entry was appended (or already present).
func appendToAllSlice(path, entry string) (bool, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	src := string(body)
	allIdx := strings.Index(src, "func All()")
	if allIdx < 0 {
		return false, nil
	}
	if strings.Contains(src, entry) {
		return true, nil
	}
	rest := src[allIdx:]
	retIdx := strings.Index(rest, "return")
	if retIdx < 0 {
		return false, nil
	}
	braceOpen := strings.Index(rest[retIdx:], "{")
	if braceOpen < 0 {
		return false, nil
	}
	braceOpen += retIdx
	depth := 0
	closeAt := -1
	for i := braceOpen; i < len(rest); i++ {
		switch rest[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				closeAt = i
			}
		}
		if closeAt >= 0 {
			break
		}
	}
	if closeAt < 0 {
		return false, nil
	}
	absClose := allIdx + closeAt
	out := src[:absClose] + "\t\t" + entry + ",\n" + src[absClose:]
	if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
		return false, err
	}
	return true, nil
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

func toExported(name string) string {
	parts := strings.Split(name, "_")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, "")
}
