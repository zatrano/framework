package scaffold

import (
	"github.com/zatrano/framework/v2/console/consolecore"
	"github.com/zatrano/framework/v2/console/describe"
	"github.com/zatrano/framework/v2/kernel"
)

type Application = consolecore.Application

func productVersion() string     { return consolecore.ProductVersion() }
func toSnake(name string) string { return consolecore.ToSnake(name) }
func consumerModule(app *kernel.Application) string {
	return consolecore.ConsumerModule(app)
}

func Register(cli *Application, app *kernel.Application, writeAgents func(string) (string, error), seedEnv func(string) error) {
	if writeAgents == nil {
		writeAgents = describe.WriteAgentsMarkdown
	}
	if seedEnv == nil {
		seedEnv = consolecore.SeedEnvFromExample
	}
	cli.Register(&NewCommand{app: app, writeAgents: writeAgents, seedEnv: seedEnv})
	registerMakeCommand(cli, app)
	registerMakeProviderCommand(cli, app)
}
