package doctor

import (
	"github.com/zatrano/framework/v2/console/consolecore"
	"github.com/zatrano/framework/v2/kernel"
)

type Application = consolecore.Application

func hasFlag(args []string, flags ...string) bool { return consolecore.HasFlag(args, flags...) }
func cliErr(code int, err error) error            { return consolecore.CliErr(code, err) }
func parseEnabledAddons(src string) []string      { return consolecore.ParseEnabledAddons(src) }
func modulePath(root string) (string, error)      { return consolecore.ModulePath(root) }
func frameworkModuleRoot() (string, error)        { return consolecore.FrameworkModuleRoot() }

const (
	ExitGeneral = consolecore.ExitGeneral
	ExitUsage   = consolecore.ExitUsage
)

func CodeFromError(err error) int { return consolecore.CodeFromError(err) }

func Register(cli *Application, app *kernel.Application) {
	registerDoctorCommand(cli, app)
}
