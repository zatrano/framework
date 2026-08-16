package main

import (
	"fmt"
	"os"
	"strings"

	appconsole "github.com/zatrano/framework/app/console"
	"github.com/zatrano/framework/bootstrap"
	"github.com/zatrano/framework/core"
	"github.com/zatrano/framework/packages/console"
)

func main() {
	args := os.Args[1:]
	app := bootForCLI(args)
	cli := console.New(app)
	appconsole.Register(cli, app)

	if err := cli.Run(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// bootForCLI uses CoreApp for make:* (scaffold needs BasePath only — no DB/session).
// Other commands follow APP_BOOT via FromEnv("app").
func bootForCLI(args []string) *core.Application {
	if cliUsesCoreBoot(args) {
		return bootstrap.CoreApp()
	}
	return bootstrap.FromEnv("app")
}

func cliUsesCoreBoot(args []string) bool {
	if len(args) == 0 {
		return false
	}
	name := strings.TrimSpace(args[0])
	return strings.HasPrefix(name, "make:")
}
