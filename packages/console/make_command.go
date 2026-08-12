package console

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zatrano/framework/core"
)

func registerMakeCommand(console *Application, app *core.Application) {
	console.Register(&MakeCommandCommand{app: app})
}

type MakeCommandCommand struct {
	app *core.Application
}

func (c *MakeCommandCommand) Name() string        { return "make:command" }
func (c *MakeCommandCommand) Description() string { return "Create a new console command" }
func (c *MakeCommandCommand) Handle(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("command name required")
	}

	raw := args[0]
	structName := strings.TrimSuffix(raw, "Command") + "Command"
	signature := toCommandSignature(raw)
	dir := c.app.BasePath("app", "console", "commands")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, toSnake(structName)+".go")
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("command already exists: %s", path)
	}

	content := fmt.Sprintf(`package commands

import (
	"fmt"

	"github.com/zatrano/framework/core"
)

// %s is an application console command.
type %s struct {
	App *core.Application
}

func (c *%s) Name() string        { return "%s" }
func (c *%s) Description() string { return "TODO: describe %s" }
func (c *%s) Handle(args []string) error {
	if err := c.App.Bootstrap(); err != nil {
		return err
	}
	fmt.Println("%s running...")
	return nil
}
`, structName, structName, structName, signature, structName, signature, structName, signature)

	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}

	kernel := c.app.BasePath("app", "console", "kernel.go")
	if _, err := os.Stat(kernel); os.IsNotExist(err) {
		kernelContent := fmt.Sprintf(`package console

import (
	"github.com/zatrano/framework/app/console/commands"
	"github.com/zatrano/framework/core"
	coreconsole "github.com/zatrano/framework/packages/console"
)

// Register registers application console commands.
func Register(cli *coreconsole.Application, app *core.Application) {
	cli.Register(
		&commands.%s{App: app},
	)
}
`, structName)
		if err := os.WriteFile(kernel, []byte(kernelContent), 0o644); err != nil {
			return err
		}
		fmt.Printf("created %s\n", path)
		fmt.Printf("registered in app/console/kernel.go\n")
		return nil
	}

	fmt.Printf("created %s\n", path)
	fmt.Printf("register it in app/console/kernel.go: &commands.%s{App: app}\n", structName)
	return nil
}

func toCommandSignature(name string) string {
	name = strings.TrimSuffix(name, "Command")
	snake := toSnake(name)
	return strings.ReplaceAll(snake, "_", ":")
}
