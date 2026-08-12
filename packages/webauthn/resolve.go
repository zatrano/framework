package webauthn

// App is satisfied by *core.Application without importing the root module.
type App interface {
	Make(abstract string) (any, error)
}

// From resolves the WebAuthn manager from an application container.
func From(app App) *Manager {
	if app == nil {
		return nil
	}
	raw, err := app.Make("webauthn")
	if err != nil {
		return nil
	}
	m, _ := raw.(*Manager)
	return m
}
