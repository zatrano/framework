package schedule

// App is satisfied by *core.Application without importing the root module.
type App interface {
	Make(abstract string) (any, error)
}

// From resolves the scheduler from the application container.
func From(app App) *Scheduler {
	if app == nil {
		return nil
	}
	raw, err := app.Make("scheduler")
	if err != nil {
		return nil
	}
	v, _ := raw.(*Scheduler)
	return v
}
