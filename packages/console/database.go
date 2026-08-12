package console

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zatrano/framework/core"
	"github.com/zatrano/framework/packages/database"
)

func registerDatabaseCommands(console *Application, app *core.Application) {
	console.Register(
		&MigrateCommand{app: app},
		&MigrateRollbackCommand{app: app},
		&MigrateStatusCommand{app: app},
		&MigrateFreshCommand{app: app},
		&DBSeedCommand{app: app},
		&MakeModelCommand{app: app},
		&MakeMigrationCommand{app: app},
		&MakeSeederCommand{app: app},
	)
}

type MigrateCommand struct {
	app *core.Application
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
	app *core.Application
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
	app *core.Application
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
	app *core.Application
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

type DBSeedCommand struct {
	app *core.Application
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
	app *core.Application
}

func (c *MakeModelCommand) Name() string        { return "make:model" }
func (c *MakeModelCommand) Description() string { return "Create a new ORM model" }
func (c *MakeModelCommand) Handle(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("model name required")
	}
	name := args[0]
	withMigration := false
	for _, arg := range args[1:] {
		if arg == "-m" || arg == "--migration" {
			withMigration = true
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

	content := fmt.Sprintf(`package models

import "github.com/zatrano/framework/packages/orm"

type %s struct {
	orm.Model
	Name string `+"`"+`db:"name" json:"name"`+"`"+`
}

func (m *%s) TableName() string {
	return "%s"
}
`, name, name, toSnake(pluralize(name)))

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("Model created: %s\n", path)

	if withMigration {
		return (&MakeMigrationCommand{app: c.app}).Handle([]string{"create_" + toSnake(pluralize(name)) + "_table"})
	}
	return nil
}

type MakeMigrationCommand struct {
	app *core.Application
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

func (m *%s) Up(schema *schema.Builder) error {
	return schema.Create("table_name", func(table *schema.Blueprint) {
		table.ID()
		table.Timestamps()
	})
}

func (m *%s) Down(schema *schema.Builder) error {
	return schema.DropIfExists("table_name")
}
`, structName, structName, stamp, description, structName, structName)

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("Migration created: %s\n", path)
	fmt.Println("Remember to register it in database/migrations/migrations.go")
	return nil
}

type MakeSeederCommand struct {
	app *core.Application
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
	fmt.Println("Remember to register it in database/seeders/database_seeder.go")
	return nil
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
