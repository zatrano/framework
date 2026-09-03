package console

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zatrano/framework/kernel"
)

func registerFactoryCommands(console *Application, app *kernel.Application) {
	console.Register(
		&MakeResourceCommand{app: app},
		&MakeFactoryCommand{app: app},
	)
}

type MakeResourceCommand struct {
	app *kernel.Application
}

func (c *MakeResourceCommand) Name() string        { return "make:resource" }
func (c *MakeResourceCommand) Description() string { return "Create a new API resource" }
func (c *MakeResourceCommand) Handle(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("resource name required")
	}
	name := args[0]
	dir := c.app.BasePath("app", "http", "resources")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, toSnake(name)+"_resource.go")
	content := fmt.Sprintf(`package resources

import "github.com/zatrano/framework/app/models"

// %s transforms a models.%s into an API resource array.
// Use with github.com/zatrano/packages/resources:
//
//	resources.JSON(model, %s)
func %s(model models.%s) map[string]any {
	return map[string]any{
		"id": model.ID,
	}
}
`, name, name, name, name, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("Resource created: %s\n", path)
	return nil
}

type MakeFactoryCommand struct {
	app *kernel.Application
}

func (c *MakeFactoryCommand) Name() string        { return "make:factory" }
func (c *MakeFactoryCommand) Description() string { return "Create a new model factory" }
func (c *MakeFactoryCommand) Handle(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("factory model name required")
	}
	name := args[0]
	dir := c.app.BasePath("database", "factories")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, toSnake(name)+"_factory.go")
	content := fmt.Sprintf(`package factories

import (
	"github.com/zatrano/framework/app/models"
	"github.com/zatrano/packages/factory"
)

func init() {
	factory.For[models.%s](func() map[string]any {
		return map[string]any{
			"name": factory.FakeName(),
		}
	})
}
`, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("Factory created: %s\n", path)
	fmt.Println("Import _ \"your/module/database/factories\" from your seeder/tests to register it.")
	return nil
}
