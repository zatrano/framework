package console

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zatrano/framework/v2/kernel"
)

func registerMakeProviderCommand(console *Application, app *kernel.Application) {
	console.Register(&MakeProviderCommand{app: app})
}

type MakeProviderCommand struct{ app *kernel.Application }

func (c *MakeProviderCommand) Name() string        { return "make:provider" }
func (c *MakeProviderCommand) Description() string { return "Create a new service provider" }
func (c *MakeProviderCommand) Handle(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("provider name required")
	}
	name := args[0]
	if !strings.HasSuffix(name, "ServiceProvider") {
		name += "ServiceProvider"
	}
	dir := c.app.BasePath("app", "providers")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, toSnake(name)+".go")
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("provider already exists: %s", path)
	}
	content := fmt.Sprintf(`package providers

import "github.com/zatrano/framework/v2/contracts"

// %s registers application services.
type %s struct{}

func (p *%s) Register(app contracts.App) error {
	return nil
}

func (p *%s) Boot(app contracts.App) error {
	return nil
}
`, name, name, name, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("Provider created: %s\n", path)
	fmt.Println("Remember to register it with bootstrap.WithProviders.")
	return nil
}
