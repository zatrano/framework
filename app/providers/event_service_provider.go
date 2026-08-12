package providers

import (
	"github.com/zatrano/framework/core"
)

// EventServiceProvider registers application events and listeners.
type EventServiceProvider struct{}

func (p *EventServiceProvider) Register(app *core.Application) error {
	return nil
}

func (p *EventServiceProvider) Boot(app *core.Application) error {
	// Register listeners here as your application grows.
	return nil
}
