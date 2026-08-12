package console

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/zatrano/framework/core"
	"github.com/zatrano/framework/packages/openapi"
)

func registerOpenAPICommands(console *Application, app *core.Application) {
	console.Register(&OpenAPIGenerateCommand{app: app})
}

type OpenAPIGenerateCommand struct {
	app *core.Application
}

func (c *OpenAPIGenerateCommand) Name() string { return "openapi:generate" }
func (c *OpenAPIGenerateCommand) Description() string {
	return "Generate an OpenAPI document from registered routes"
}
func (c *OpenAPIGenerateCommand) Handle(args []string) error {
	if err := c.app.Bootstrap(); err != nil {
		return err
	}

	out := c.app.BasePath("storage", "app", "openapi.json")
	format := "json"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--out", "-o":
			if i+1 < len(args) {
				out = args[i+1]
				i++
			}
		case "--yaml":
			format = "yaml"
		}
	}
	if format == "yaml" && strings.HasSuffix(strings.ToLower(out), ".json") {
		out = strings.TrimSuffix(out, filepath.Ext(out)) + ".yaml"
	}

	spec := openapi.Generate(c.app.Router().Routes(), openapi.Options{
		Title:       c.app.Config().GetString("app.name", "ZATRANO") + " API",
		Description: "Auto-generated from route definitions",
		Version:     "1.0.0",
		ServerURL:   c.app.Config().GetString("app.url", "http://localhost:8080"),
	})

	var err error
	if format == "yaml" {
		err = openapi.WriteYAML(out, spec)
	} else {
		err = openapi.WriteJSON(out, spec)
	}
	if err != nil {
		return err
	}
	fmt.Printf("OpenAPI document written to %s (%d paths)\n", out, len(spec.Paths))
	return nil
}
