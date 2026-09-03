package console

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zatrano/framework/kernel"
)

func registerObserverCommands(console *Application, app *kernel.Application) {
	console.Register(&MakeObserverCommand{app: app})
}

type MakeObserverCommand struct {
	app *kernel.Application
}

func (c *MakeObserverCommand) Name() string        { return "make:observer" }
func (c *MakeObserverCommand) Description() string { return "Create a model event observer scaffold" }
func (c *MakeObserverCommand) Handle(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("observer name required")
	}
	name := args[0]
	structName := toExported(name)
	if !strings.HasSuffix(strings.ToLower(structName), "observer") {
		structName += "Observer"
	}
	subject := strings.TrimSuffix(strings.ToLower(toSnake(structName)), "_observer")
	dir := c.app.BasePath("app", "observers")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, toSnake(structName)+".go")
	content := fmt.Sprintf(`package observers

import (
	"fmt"

	"github.com/zatrano/framework/packages/events"
)

// %s observes "%s.*" lifecycle events.
type %s struct{}

var _ events.ModelObserver = (*%s)(nil)

func (o *%s) Created(event any) error {
	fmt.Printf("%s created: %%v\n", event)
	return nil
}

func (o *%s) Updated(event any) error {
	fmt.Printf("%s updated: %%v\n", event)
	return nil
}

func (o *%s) Deleted(event any) error {
	fmt.Printf("%s deleted: %%v\n", event)
	return nil
}

// Register%s attaches the observer to the dispatcher.
func Register%s(d *events.Dispatcher) {
	d.ObserveModel(%q, &%s{})
}
`, structName, subject, structName, structName, structName, subject, structName, subject, structName, subject, structName, structName, subject, structName)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("Observer created: %s\n", path)
	fmt.Printf("Call observers.Register%s(events.From(app)) during boot.\n", structName)
	return nil
}
