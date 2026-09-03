package console

import (
	"github.com/zatrano/framework/kernel"
	coreconsole "github.com/zatrano/framework/packages/console"
)

// Register registers application console commands.
func Register(cli *coreconsole.Application, app *kernel.Application) {
	_ = app
	// Register app commands here, e.g. cli.Register(&commands.MyCommand{App: app})
}
