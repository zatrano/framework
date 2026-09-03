package circuit

import (
	"time"

	"github.com/zatrano/framework/bootstrap/addons"
	"github.com/zatrano/framework/kernel"
	"github.com/zatrano/framework/packages/env"
)

func init() {
	addons.Register(addons.Meta{
		Name:        "circuit",
		Key:         "circuit",
		Description: "Circuit breaker",
		Factory:     func() kernel.Provider { return &ServiceProvider{} },
	})
}

// ServiceProvider boots the circuit addon.
type ServiceProvider struct{}

func (p *ServiceProvider) Register(app *kernel.Application) error {
	app.Container().Instance("circuit", New(Settings{
		FailureThreshold: env.GetInt("CIRCUIT_FAILURE_THRESHOLD", 5),
		SuccessThreshold: env.GetInt("CIRCUIT_SUCCESS_THRESHOLD", 2),
		Timeout:          time.Duration(env.GetInt("CIRCUIT_TIMEOUT_SECONDS", 30)) * time.Second,
	}))
	return nil
}

func (p *ServiceProvider) Boot(app *kernel.Application) error { return nil }
