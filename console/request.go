package console

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zatrano/framework/v2/kernel"
)

func registerRequestCommands(console *Application, app *kernel.Application) {
	console.Register(&MakeRequestCommand{app: app})
}

type MakeRequestCommand struct {
	app *kernel.Application
}

func (c *MakeRequestCommand) Name() string        { return "make:request" }
func (c *MakeRequestCommand) Description() string { return "Create a new form request class" }
func (c *MakeRequestCommand) Handle(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("request name required")
	}
	name := args[0]
	if len(name) < 7 || name[len(name)-7:] != "Request" {
		name += "Request"
	}
	dir := c.app.BasePath("app", "http", "requests")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, toSnake(name)+".go")
	content := fmt.Sprintf(`package requests

import (
	. "github.com/zatrano/framework/v2/kernel/http"
	"github.com/zatrano/packages/validation"
)

type %s struct {
	validation.Base
}

func (r %s) Rules() map[string]string {
	return map[string]string{
		"name": "required|min:2",
	}
}

func (r %s) Messages() map[string]string {
	return map[string]string{}
}

func (r %s) Authorize(req *Request) bool {
	return true
}
`, name, name, name, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("Request created: %s\n", path)
	return nil
}
