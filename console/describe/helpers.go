package describe

import (
	"github.com/zatrano/framework/v2/console/consolecore"
	"github.com/zatrano/framework/v2/kernel"
)

type Application = consolecore.Application

func hasFlag(args []string, flags ...string) bool { return consolecore.HasFlag(args, flags...) }
func formatFromArgs(args []string) (string, error) {
	return consolecore.FormatFromArgs(args)
}
func productVersion() string { return consolecore.ProductVersion() }
func productVersionAt(root string) string {
	return consolecore.ProductVersionAt(root)
}
func frameworkModuleRoot() (string, error)   { return consolecore.FrameworkModuleRoot() }
func modulePath(root string) (string, error) { return consolecore.ModulePath(root) }

// Lookup is the consumer catalog lookup (kernel primitives + ecosystem).
func Lookup(name string) (kernel.PackageInfo, bool) { return catalogLookup(name) }

func Ecosystem() []kernel.PackageInfo {
	return append([]kernel.PackageInfo(nil), ecosystemCatalog...)
}

func ByLayer(layer kernel.Layer) []kernel.PackageInfo { return catalogByLayer(layer) }

func Libraries() []kernel.PackageInfo { return catalogLibraries() }

func Register(cli *Application, app *kernel.Application) {
	registerDescribeCommand(cli, app)
}
