package consolecore

import (
	"fmt"
	"os"
	"sort"
	"text/tabwriter"

	"github.com/zatrano/framework/v2/kernel"
)

// Command is a CLI command.
type Command interface {
	Name() string
	Description() string
	Handle(args []string) error
}

// Application is the console kernel.
type Application struct {
	Kernel   *kernel.Application
	Unknown  func(name string) error
	commands map[string]Command
}

// New returns an empty console application (commands are registered by the caller).
func New(app *kernel.Application) *Application {
	return &Application{
		Kernel:   app,
		commands: make(map[string]Command),
	}
}

// Register registers commands.
func (c *Application) Register(commands ...Command) {
	for _, command := range commands {
		c.commands[command.Name()] = command
	}
}

// Commands returns registered commands.
func (c *Application) Commands() map[string]Command {
	return c.commands
}

// Run executes the console application.
func (c *Application) Run(args []string) error {
	if len(args) == 0 {
		return c.commands["list"].Handle(nil)
	}
	name := args[0]
	switch name {
	case "--help", "-h", "help":
		return c.commands["list"].Handle(nil)
	case "--version", "-v":
		return c.commands["version"].Handle(nil)
	}
	command, ok := c.commands[name]
	if !ok {
		if c.Unknown != nil {
			return c.Unknown(name)
		}
		return fmt.Errorf("command [%s] not defined\nNext: run zatrano --help (or list) for available commands", name)
	}
	return command.Handle(args[1:])
}

// WriteCommandList prints registered command names and descriptions.
func (c *Application) WriteCommandList() error {
	fmt.Println("ZATRANO Console")
	fmt.Println()
	names := make([]string, 0, len(c.commands))
	for name := range c.commands {
		names = append(names, name)
	}
	sort.Strings(names)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, name := range names {
		command := c.commands[name]
		fmt.Fprintf(w, "  %s\t%s\n", name, command.Description())
	}
	return w.Flush()
}
