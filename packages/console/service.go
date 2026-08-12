package console

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zatrano/framework/core"
)

func registerServiceCommands(console *Application, app *core.Application) {
	console.Register(&MakeServiceCommand{app: app})
}

type MakeServiceCommand struct {
	app *core.Application
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
	dir := c.app.BasePath("app", "services")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, toSnake(name)+".go")
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
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("Service created: %s\n", path)
	return nil
}
