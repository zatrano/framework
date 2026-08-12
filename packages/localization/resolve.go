package localization

// App is satisfied by *core.Application without importing the root module.
type App interface {
	Make(abstract string) (any, error)
}

// From resolves the translator from the application container.
func From(app App) *Translator {
	if app == nil {
		return nil
	}
	raw, err := app.Make("translator")
	if err != nil {
		return nil
	}
	v, _ := raw.(*Translator)
	return v
}
