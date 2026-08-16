package main

import (
	"fmt"
	"os"

	appconsole "github.com/zatrano/framework/app/console"
	"github.com/zatrano/framework/bootstrap"
	"github.com/zatrano/framework/packages/console"
)

func main() {
	// APP_BOOT selects the stack: app|api|web|minimal|core|demo
	// CLI defaults to app (lean). Use APP_BOOT=demo for the full exploration stack.
	app := bootstrap.FromEnv("app")
	cli := console.New(app)
	appconsole.Register(cli, app)

	if err := cli.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
