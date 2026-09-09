package console

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/zatrano/framework/v2/console/generator"
	"github.com/zatrano/framework/v2/kernel"
)

func registerServiceCommands(console *Application, app *kernel.Application) {
	console.Register(&MakeServiceCommand{app: app})
}

type MakeServiceCommand struct {
	app *kernel.Application
}

func (c *MakeServiceCommand) Name() string        { return "make:service" }
func (c *MakeServiceCommand) Description() string { return "Create an application service scaffold" }
func (c *MakeServiceCommand) Handle(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("service name required")
	}
	name := toExported(args[0])
	if !strings.HasSuffix(name, "Service") {
		name += "Service"
	}
	path := filepath.Join(c.app.BasePath("app", "services"), toSnake(name)+".go")
	content := fmt.Sprintf(`package services

// %s encapsulates application business logic.
type %s struct{}

// New%s creates a %s.
func New%s() *%s {
	return &%s{}
}

// Handle performs the primary service action.
func (s *%s) Handle() error {
	// TODO: implement service logic
	return nil
}
`, name, name, name, name, name, name, name, name)
	if err := generator.WriteFile(path, content); err != nil {
		return err
	}
	fmt.Printf("Service created: %s\n", path)
	return nil
}
