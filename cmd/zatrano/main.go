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
	// Framework CLI defaults to demo for exploration; production apps usually omit
	// the default (app) or set APP_BOOT=api|web|minimal.
	app := bootstrap.FromEnv("demo")
	cli := console.New(app)
	appconsole.Register(cli, app)

	if err := cli.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
