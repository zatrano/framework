package console

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/zatrano/framework/kernel"
)

func registerSubscriberCommands(console *Application, app *kernel.Application) {
	console.Register(&MakeSubscriberCommand{app: app})
}

type MakeSubscriberCommand struct {
	app *kernel.Application
}

func (c *MakeSubscriberCommand) Name() string        { return "make:subscriber" }
func (c *MakeSubscriberCommand) Description() string { return "Create an event subscriber scaffold" }
func (c *MakeSubscriberCommand) Handle(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("subscriber name required")
	}
	name := toExported(args[0])
	dir := c.app.BasePath("app", "subscribers")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, toSnake(name)+".go")
	content := fmt.Sprintf(`package subscribers

import "github.com/zatrano/framework/packages/events"

// %s registers related event listeners.
type %s struct{}

// Subscribe wires listeners for %s.
func (s *%s) Subscribe(d *events.Dispatcher) {
	d.Listen("example.event", s.HandleExample)
}

// HandleExample handles the example.event event.
func (s *%s) HandleExample(event any) error {
	// TODO: handle event
	_ = event
	return nil
}
`, name, name, name, name, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("Subscriber created: %s\n", path)
	fmt.Printf("Register with events.Dispatcher.Register(&subscribers.%s{})\n", name)
	return nil
}
