package providers

import "github.com/zatrano/framework/core"

// ScheduleServiceProvider registers scheduled tasks.
type ScheduleServiceProvider struct{}

func (p *ScheduleServiceProvider) Register(app *core.Application) error {
	return nil
}

func (p *ScheduleServiceProvider) Boot(app *core.Application) error {
	// Register scheduled commands here, e.g.:
	// schedule.From(app).Command("reports", fn).Daily()
	return nil
}
