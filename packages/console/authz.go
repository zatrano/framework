package console

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zatrano/framework/core"
)

func registerAuthzCommands(console *Application, app *core.Application) {
	console.Register(
		&MakePolicyCommand{app: app},
	)
}

type MakePolicyCommand struct {
	app *core.Application
}

func (c *MakePolicyCommand) Name() string        { return "make:policy" }
func (c *MakePolicyCommand) Description() string { return "Create a new policy class" }
func (c *MakePolicyCommand) Handle(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("policy name required")
	}
	name := args[0]
	if len(name) < 6 || name[len(name)-6:] != "Policy" {
		name += "Policy"
	}
	dir := c.app.BasePath("app", "policies")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, toSnake(name)+".go")
	content := fmt.Sprintf(`package policies

import (
	"github.com/zatrano/framework/packages/auth"
	"github.com/zatrano/framework/packages/authorization"
)

func New%s() *authorization.Policy {
	return authorization.NewPolicy().
		Define("view", func(user auth.Authenticatable, arguments ...any) bool {
			return user != nil
		}).
		Define("update", func(user auth.Authenticatable, arguments ...any) bool {
			return user != nil
		}).
		Define("delete", func(user auth.Authenticatable, arguments ...any) bool {
			return user != nil
		})
}
`, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("Policy created: %s\n", path)
	fmt.Println("Register it with authorization.From(app).Policy(\"name\", policies.New" + name + "())")
	return nil
}
