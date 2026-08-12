package console

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"

	"github.com/zatrano/framework/core"
)

func registerOctaneCommands(console *Application, app *core.Application) {
	console.Register(&OctaneStartCommand{app: app})
}

type OctaneStartCommand struct {
	app *core.Application
}

func (c *OctaneStartCommand) Name() string { return "octane:start" }
func (c *OctaneStartCommand) Description() string {
	return "Start HTTP serving with Octane metrics and optional GOMAXPROCS worker hint"
}
func (c *OctaneStartCommand) Handle(args []string) error {
	addr := ":8080"
	workers := 0
	for i := 0; i < len(args); i++ {
		if (args[i] == "--port" || args[i] == "-p") && i+1 < len(args) {
			addr = ":" + args[i+1]
			i++
		}
		if strings.HasPrefix(args[i], "--host=") {
			addr = strings.TrimPrefix(args[i], "--host=")
		}
		if (args[i] == "--workers" || args[i] == "-w") && i+1 < len(args) {
			if n, err := strconv.Atoi(args[i+1]); err == nil {
				workers = n
			}
			i++
		}
	}
	if err := c.app.Bootstrap(); err != nil {
		return err
	}
	if workers > 0 {
		c.app.Octane().SetWorkers(workers)
		runtime.GOMAXPROCS(workers)
	}
	fmt.Printf("Octane workers=%d gomaxprocs=%d\n", c.app.Octane().Workers(), runtime.GOMAXPROCS(0))
	return c.app.Run(addr)
}
