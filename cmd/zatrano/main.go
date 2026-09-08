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
		os.Exit(console.CodeFromError(err))
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
	if strings.HasPrefix(name, "make:") {
		return true
	}
	switch name {
	case "db:setup", "new", "describe", "doctor", "agents:generate",
		"--help", "-h", "help", "--version", "-v", "version":
		return true
	default:
		return false
	}
}
