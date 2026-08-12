package providers

import (
	"github.com/zatrano/framework/core"
	"github.com/zatrano/framework/database/migrations"
	"github.com/zatrano/framework/database/seeders"
)

// DatabaseServiceProvider registers migrations and seeders.
type DatabaseServiceProvider struct{}

func (p *DatabaseServiceProvider) Register(app *core.Application) error {
	return nil
}

func (p *DatabaseServiceProvider) Boot(app *core.Application) error {
	app.SetMigrations(migrations.All())
	app.SetSeeders(seeders.All())
	return nil
}
