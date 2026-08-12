package httpclient

// App is satisfied by *core.Application without importing the root module.
type App interface {
	Make(abstract string) (any, error)
}

// From resolves the http from the application container.
func From(app App) *Client {
	if app == nil {
		return nil
	}
	raw, err := app.Make("http")
	if err != nil {
		return nil
	}
	v, _ := raw.(*Client)
	return v
}
