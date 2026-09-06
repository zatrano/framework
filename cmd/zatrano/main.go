package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/zatrano/framework/v2/bootstrap"
	"github.com/zatrano/framework/v2/console"
	"github.com/zatrano/framework/v2/kernel"
)

func main() {
	args := os.Args[1:]
	app := bootForCLI(args)
	cli := console.New(app)
	if err := cli.Run(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func bootForCLI(args []string) *kernel.Application {
	if cliUsesCoreBoot(args) {
		return bootstrap.App()
	}
	return bootstrap.FromEnv("app")
}

func cliUsesCoreBoot(args []string) bool {
	if len(args) == 0 {
		return false
	}
	name := strings.TrimSpace(args[0])
	return strings.HasPrefix(name, "make:") || name == "db:setup" || name == "new" || name == "describe" || name == "doctor" || name == "agents:generate"
}
