package main

import (
	"os"

	"github.com/zatrano/framework/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
