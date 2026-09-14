package console

import "github.com/zatrano/framework/v2/console/consolecore"

func seedEnvFromExample(root string) error {
	return consolecore.SeedEnvFromExample(root)
}
