package providers

import "github.com/zatrano/framework/core"

// AuthServiceProvider registers authorization gates and policies.
type AuthServiceProvider struct{}

func (p *AuthServiceProvider) Register(app *core.Application) error {
	return nil
}

func (p *AuthServiceProvider) Boot(app *core.Application) error {
	// Register gates and policies here, e.g.:
	// authorization.From(app).Policy("user", policies.NewUserPolicy())
	return nil
}
