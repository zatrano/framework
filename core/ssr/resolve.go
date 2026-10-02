package ssr

import "github.com/zatrano/canvas"

// App is satisfied by *kernel.Application without importing the root module cycle.
type App interface {
	Make(abstract string) (any, error)
}

// ContainerKey is the DI key for the Canvas engine (*canvas.Engine).
// Enable name stays "template"; the bound value is always Canvas.
const ContainerKey = "template"

// From resolves the Canvas engine from the application container.
func From(app App) *canvas.Engine {
	if app == nil {
		return nil
	}
	raw, err := app.Make(ContainerKey)
	if err != nil {
		return nil
	}
	e, _ := raw.(*canvas.Engine)
	return e
}
