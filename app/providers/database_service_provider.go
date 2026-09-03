package providers

import (
	"github.com/zatrano/framework/kernel"
	"github.com/zatrano/framework/database/migrations"
	"github.com/zatrano/framework/database/seeders"
)

// DatabaseServiceProvider registers migrations and seeders.
type DatabaseServiceProvider struct{}

func (p *DatabaseServiceProvider) Register(app *kernel.Application) error {
	return nil
}

func (p *DatabaseServiceProvider) Boot(app *kernel.Application) error {
	app.SetMigrations(migrations.All())
	app.SetSeeders(seeders.All())
	return nil
}
