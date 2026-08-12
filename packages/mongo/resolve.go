package mongo

// App is satisfied by *core.Application without importing the root module.
type App interface {
	Make(abstract string) (any, error)
}

// From resolves the Mongo client from an application container.
func From(app App) *Client {
	if app == nil {
		return nil
	}
	raw, err := app.Make("mongo")
	if err != nil {
		return nil
	}
	c, _ := raw.(*Client)
	return c
}
