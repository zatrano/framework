package pkgmanager

import (
	"github.com/zatrano/framework/v2/console/consolecore"
	"github.com/zatrano/framework/v2/console/describe"
	"github.com/zatrano/framework/v2/kernel"
)

type Application = consolecore.Application

func hasFlag(args []string, flags ...string) bool { return consolecore.HasFlag(args, flags...) }
func formatFromArgs(args []string) (string, error) {
	return consolecore.FormatFromArgs(args)
}
func cliErr(code int, err error) error {
	return consolecore.CliErr(code, err)
}
func cliFailed(code int, action, target string, err error, next string) error {
	return consolecore.CliFailed(code, action, target, err, next)
}
func classifyContextError(err error) error { return consolecore.ClassifyContextError(err) }
func CodeFromError(err error) int          { return consolecore.CodeFromError(err) }
func parseEnabledAddons(src string) []string {
	return consolecore.ParseEnabledAddons(src)
}
func modulePath(root string) (string, error) { return consolecore.ModulePath(root) }
func catalogLookup(name string) (kernel.PackageInfo, bool) {
	return describe.Lookup(name)
}
func catalogLibraries() []kernel.PackageInfo { return describe.Libraries() }

const (
	ExitSuccess     = consolecore.ExitSuccess
	ExitGeneral     = consolecore.ExitGeneral
	ExitUsage       = consolecore.ExitUsage
	ExitResolution  = consolecore.ExitResolution
	ExitPlanning    = consolecore.ExitPlanning
	ExitAcquisition = consolecore.ExitAcquisition
	ExitEnablement  = consolecore.ExitEnablement
	ExitCanceled    = consolecore.ExitCanceled
)

func Register(cli *Application, app *kernel.Application) {
	registerPackageCommands(cli, app)
}
