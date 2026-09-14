package console

import "github.com/zatrano/framework/v2/console/consolecore"

const currentRelease = consolecore.CurrentRelease

func productVersion() string               { return consolecore.ProductVersion() }
func productVersionAt(root string) string  { return consolecore.ProductVersionAt(root) }
func frameworkModuleRoot() (string, error) { return consolecore.FrameworkModuleRoot() }
func hasFlag(args []string, flags ...string) bool {
	return consolecore.HasFlag(args, flags...)
}
