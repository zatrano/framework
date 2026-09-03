package graphql

import (
	"github.com/zatrano/framework/bootstrap/addons"
	"github.com/zatrano/framework/core"
	"github.com/zatrano/framework/packages/features"
)

func init() {
	addons.Register(addons.Meta{
		Name:        "graphql",
		Key:         "graphql",
		Description: "GraphQL schema and queries",
		Factory:     func() core.Provider { return &ServiceProvider{} },
	})
}

// ServiceProvider boots the GraphQL addon.
type ServiceProvider struct{}

func (p *ServiceProvider) Register(app *core.Application) error {
	schema := NewSchema()
	schema.Query("health", func(args map[string]any) (any, error) {
		return "ok", nil
	})
	schema.Query("echo", func(args map[string]any) (any, error) {
		msg, _ := args["message"].(string)
		if msg == "" {
			msg = "hello"
		}
		return msg, nil
	})
	schema.Query("feature", func(args map[string]any) (any, error) {
		f := features.From(app)
		if f == nil {
			return false, nil
		}
		name, _ := args["name"].(string)
		return f.Active(name), nil
	})
	schema.Mutation("ping", func(args map[string]any) (any, error) {
		return map[string]any{"pong": true}, nil
	})
	app.Container().Instance("graphql", schema)
	return nil
}

func (p *ServiceProvider) Boot(app *core.Application) error { return nil }
