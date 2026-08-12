package authorization

// App is satisfied by *core.Application without importing the root module.
type App interface {
	Make(abstract string) (any, error)
}

// From resolves the gate from the application container.
func From(app App) *Gate {
	if app == nil {
		return nil
	}
	raw, err := app.Make("gate")
	if err != nil {
		return nil
	}
	v, _ := raw.(*Gate)
	return v
}
