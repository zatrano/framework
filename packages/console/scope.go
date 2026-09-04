package console

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zatrano/framework/kernel"
)

func registerScopeCommands(console *Application, app *kernel.Application) {
	console.Register(&MakeScopeCommand{app: app})
}

type MakeScopeCommand struct {
	app *kernel.Application
}

func (c *MakeScopeCommand) Name() string        { return "make:scope" }
func (c *MakeScopeCommand) Description() string { return "Create a ORM query scope scaffold" }
func (c *MakeScopeCommand) Handle(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("scope name required")
	}
	name := args[0]
	model := "User"
	global := false
	for _, arg := range args[1:] {
		if strings.HasPrefix(arg, "--model=") {
			model = toExported(strings.TrimPrefix(arg, "--model="))
		}
		if arg == "--global" {
			global = true
		}
	}
	structName := toExported(name)
	scopeKey := strings.ToLower(toSnake(structName))
	scopeKey = strings.TrimSuffix(scopeKey, "_scope")
	mod := consumerModule(c.app)
	dir := c.app.BasePath("app", "scopes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, toSnake(structName)+".go")
	registerCall := fmt.Sprintf("orm.RegisterScope[models.%s](%q, Scope%s)", model, scopeKey, structName)
	if global {
		registerCall = fmt.Sprintf("orm.AddGlobalScope[models.%s](%q, Scope%s)", model, scopeKey, structName)
	}
	content := fmt.Sprintf(`package scopes

import (
	"%s/app/models"
	"github.com/zatrano/framework/packages/orm"
)

// Register%s registers the "%s" scope for %s.
func Register%s() {
	%s
}

// Scope%s is a query scope for models.%s.
func Scope%s(q *orm.Querier[models.%s]) *orm.Querier[models.%s] {
	// TODO: apply filters for "%s".
	return q
}
`, mod, structName, scopeKey, model, structName, registerCall, structName, model, structName, model, model, scopeKey)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("Scope created: %s\n", path)
	fmt.Printf("Call scopes.Register%s() during boot (e.g. AppServiceProvider).\n", structName)
	return nil
}
