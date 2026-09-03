package providers

import (
	"github.com/zatrano/framework/kernel"
)

// EventServiceProvider registers application events and listeners.
type EventServiceProvider struct{}

func (p *EventServiceProvider) Register(app *kernel.Application) error {
	return nil
}

func (p *EventServiceProvider) Boot(app *kernel.Application) error {
	// Register listeners here as your application grows.
	return nil
}
