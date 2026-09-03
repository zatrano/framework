package pulse

import (
	"github.com/zatrano/framework/bootstrap/addons"
	"github.com/zatrano/framework/kernel"
	"github.com/zatrano/framework/packages/inspector"
	"github.com/zatrano/framework/packages/search"
)

func init() {
	addons.Register(addons.Meta{
		Name:        "pulse",
		Key:         "pulse",
		Description: "Metrics pulse dashboard",
		Factory:     func() kernel.Provider { return &ServiceProvider{} },
	})
}

// ServiceProvider boots the pulse addon.
type ServiceProvider struct{}

func (p *ServiceProvider) Register(app *kernel.Application) error {
	if app.Metrics() == nil {
		return nil
	}
	dash := New(app.Metrics()).WithExtra(func() map[string]any {
		extra := map[string]any{}
		if insp := inspector.From(app); insp != nil {
			extra["inspector_entries"] = insp.Count()
		}
		if s := search.From(app); s != nil {
			extra["search_docs"] = s.Count()
		}
		return extra
	})
	app.Container().Instance("pulse", dash)
	return nil
}

func (p *ServiceProvider) Boot(app *kernel.Application) error { return nil }
