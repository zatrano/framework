package backup

import (
	"github.com/zatrano/framework/bootstrap/addons"
	"github.com/zatrano/framework/core"
)

func init() {
	addons.Register(addons.Meta{
		Name:        "backup",
		Key:         "backup",
		Description: "Database backup/restore (SQLite + native dump tools)",
		Factory:     func() core.Provider { return &ServiceProvider{} },
	})
}

// ServiceProvider boots the backup addon.
type ServiceProvider struct{}

func (p *ServiceProvider) Register(app *core.Application) error {
	mgr, err := ManagerFromApp(app, "")
	if err != nil {
		return err
	}
	app.Container().Instance("backup", mgr)
	return nil
}

func (p *ServiceProvider) Boot(app *core.Application) error { return nil }
