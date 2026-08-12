package view

// App is satisfied by *core.Application without importing the root module.
type App interface {
	Make(abstract string) (any, error)
}

// From resolves the view from the application container.
func From(app App) *Engine {
	if app == nil {
		return nil
	}
	raw, err := app.Make("view")
	if err != nil {
		return nil
	}
	v, _ := raw.(*Engine)
	return v
}
