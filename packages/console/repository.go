package console

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zatrano/framework/core"
)

func registerRepositoryCommands(console *Application, app *core.Application) {
	console.Register(&MakeRepositoryCommand{app: app})
}

type MakeRepositoryCommand struct {
	app *core.Application
}

func (c *MakeRepositoryCommand) Name() string        { return "make:repository" }
func (c *MakeRepositoryCommand) Description() string { return "Create a model repository scaffold" }
func (c *MakeRepositoryCommand) Handle(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("repository name required")
	}
	name := toExported(args[0])
	model := strings.TrimSuffix(name, "Repository")
	if model == "" || model == name {
		model = name
	}
	if !strings.HasSuffix(name, "Repository") {
		name += "Repository"
	}
	for _, arg := range args[1:] {
		if strings.HasPrefix(arg, "--model=") {
			model = toExported(strings.TrimPrefix(arg, "--model="))
		}
	}
	dir := c.app.BasePath("app", "repositories")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, toSnake(name)+".go")
	content := fmt.Sprintf(`package repositories

import (
	"github.com/zatrano/framework/app/models"
	"github.com/zatrano/framework/packages/orm"
)

// %s provides data access for models.%s.
type %s struct{}

// New%s creates a %s.
func New%s() *%s {
	return &%s{}
}

// All returns all %s records.
func (r *%s) All() ([]models.%s, error) {
	return orm.All[models.%s]()
}

// Find finds a %s by id.
func (r *%s) Find(id any) (*models.%s, error) {
	return orm.Find[models.%s](id)
}

// Create persists a new %s from attributes.
func (r *%s) Create(attrs map[string]any) (*models.%s, error) {
	return orm.Create[models.%s](attrs)
}
`, name, model, name, name, name, name, name, name, model, name, model, model, model, name, model, model, model, name, model, model)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("Repository created: %s\n", path)
	return nil
}
