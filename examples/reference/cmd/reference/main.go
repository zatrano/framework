package main

import (
	"fmt"
	"os"

	"github.com/zatrano/framework/v2/bootstrap"
	fwconsole "github.com/zatrano/framework/v2/console"
	"github.com/zatrano/framework/v2/examples/reference"
)

func main() {
	app := bootstrap.App(bootstrap.WithProviders(reference.Providers()...))
	cli := fwconsole.New(app)
	if err := cli.Run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(fwconsole.CodeFromError(err))
	}
}
