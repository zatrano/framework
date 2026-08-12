package database

import (
	"fmt"

	"github.com/zatrano/framework/packages/database/migration"
	"github.com/zatrano/framework/packages/database/query"
	"github.com/zatrano/framework/packages/database/seeder"
)

// Bootable is satisfied by *core.Application for migrator/seeder helpers.
type Bootable interface {
	App
	Bootstrap() error
	Migrations() any
	Seeders() any
}

// Table starts a query builder on a table via the application DB binding.
func Table(app App, table string) (*query.Builder, error) {
	db := From(app)
	if db == nil {
		return nil, fmt.Errorf("database not configured")
	}
	return db.Table(table)
}

// Migrator creates a migrator for the application's registered migrations.
func Migrator(app Bootable) (*migration.Migrator, error) {
	if app == nil {
		return nil, fmt.Errorf("application unavailable")
	}
	if err := app.Bootstrap(); err != nil {
		return nil, err
	}
	db := From(app)
	if db == nil {
		return nil, fmt.Errorf("database not configured")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	driver, err := db.DriverName()
	if err != nil {
		return nil, err
	}
	items, err := migrationList(app.Migrations())
	if err != nil {
		return nil, err
	}
	return migration.NewMigrator(sqlDB, driver, items), nil
}

// SeederRunner builds a seeder runner from application registrations.
func SeederRunner(app Bootable) (*seeder.Runner, error) {
	if app == nil {
		return nil, fmt.Errorf("application unavailable")
	}
	if err := app.Bootstrap(); err != nil {
		return nil, err
	}
	items, err := seederList(app.Seeders())
	if err != nil {
		return nil, err
	}
	return seeder.NewRunner(items...), nil
}

func migrationList(raw any) ([]migration.Migration, error) {
	if raw == nil {
		return nil, nil
	}
	switch v := raw.(type) {
	case []migration.Migration:
		return v, nil
	case []any:
		out := make([]migration.Migration, 0, len(v))
		for _, item := range v {
			m, ok := item.(migration.Migration)
			if !ok {
				return nil, fmt.Errorf("invalid migration registration type %T", item)
			}
			out = append(out, m)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("invalid migrations registration type %T", raw)
	}
}

func seederList(raw any) ([]seeder.Seeder, error) {
	if raw == nil {
		return nil, nil
	}
	switch v := raw.(type) {
	case []seeder.Seeder:
		return v, nil
	case []any:
		out := make([]seeder.Seeder, 0, len(v))
		for _, item := range v {
			s, ok := item.(seeder.Seeder)
			if !ok {
				return nil, fmt.Errorf("invalid seeder registration type %T", item)
			}
			out = append(out, s)
		}
		return out, nil
	default:
		return nil, fmt.Errorf("invalid seeders registration type %T", raw)
	}
}
