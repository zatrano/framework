package console

import (
	"github.com/zatrano/framework/bootstrap/addons"
	"github.com/zatrano/framework/kernel"
)

func registerAddonCLI(console *Application, app *kernel.Application) {
	for _, meta := range addons.Available() {
		if meta.CLI == nil {
			continue
		}
		for _, cmd := range meta.CLI(app) {
			if cmd.Name == "" || cmd.Handle == nil {
				continue
			}
			console.Register(&addonCLICommand{cmd: cmd})
		}
	}
}

type addonCLICommand struct {
	cmd addons.CLICommand
}

func (c *addonCLICommand) Name() string        { return c.cmd.Name }
func (c *addonCLICommand) Description() string { return c.cmd.Description }
func (c *addonCLICommand) Handle(args []string) error {
	return c.cmd.Handle(args)
}
