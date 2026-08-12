package mail

// App is satisfied by *core.Application without importing the root module.
type App interface {
	Make(abstract string) (any, error)
}

// From resolves the mail from the application container.
func From(app App) *Manager {
	if app == nil {
		return nil
	}
	raw, err := app.Make("mail")
	if err != nil {
		return nil
	}
	v, _ := raw.(*Manager)
	return v
}
