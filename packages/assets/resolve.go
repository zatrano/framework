package assets

// App is satisfied by *core.Application without importing the root module.
type App interface {
	Make(abstract string) (any, error)
}

// From resolves the assets from the application container.
func From(app App) *Manifest {
	if app == nil {
		return nil
	}
	raw, err := app.Make("assets")
	if err != nil {
		return nil
	}
	v, _ := raw.(*Manifest)
	return v
}
