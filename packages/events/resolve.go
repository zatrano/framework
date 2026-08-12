package events

// App is satisfied by *core.Application without importing the root module.
type App interface {
	Make(abstract string) (any, error)
}

// From resolves the events from the application container.
func From(app App) *Dispatcher {
	if app == nil {
		return nil
	}
	raw, err := app.Make("events")
	if err != nil {
		return nil
	}
	v, _ := raw.(*Dispatcher)
	return v
}
