package ssr

// App is satisfied by *kernel.Application without importing the root module cycle.
type App interface {
	Make(abstract string) (any, error)
}

// ContainerKey is the DI key for the SSR Engine.
const ContainerKey = "template"

// From resolves the SSR Engine from the application container.
func From(app App) Engine {
	if app == nil {
		return nil
	}
	raw, err := app.Make(ContainerKey)
	if err != nil {
		return nil
	}
	e, _ := raw.(Engine)
	return e
}
