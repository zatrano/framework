package console

import (
	"github.com/zatrano/framework/core"
	coreconsole "github.com/zatrano/framework/packages/console"
)

// Register registers application console commands.
func Register(cli *coreconsole.Application, app *core.Application) {
	_ = app
	// Register app commands here, e.g. cli.Register(&commands.MyCommand{App: app})
}
