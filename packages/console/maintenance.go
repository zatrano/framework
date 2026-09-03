package console

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/zatrano/framework/kernel"
	"github.com/zatrano/framework/packages/maintenance"
)

func registerMaintenanceCommands(console *Application, app *kernel.Application) {
	console.Register(
		&DownCommand{app: app},
		&UpCommand{app: app},
	)
}

type DownCommand struct {
	app *kernel.Application
}

func (c *DownCommand) Name() string        { return "down" }
func (c *DownCommand) Description() string { return "Put the application into maintenance mode" }
func (c *DownCommand) Handle(args []string) error {
	if err := c.app.Bootstrap(); err != nil {
		return err
	}
	payload := maintenance.Payload{
		Message:    "Application is in maintenance mode.",
		RetryAfter: 60,
	}
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "--message" && i+1 < len(args):
			payload.Message = args[i+1]
			i++
		case strings.HasPrefix(args[i], "--message="):
			payload.Message = strings.TrimPrefix(args[i], "--message=")
		case args[i] == "--retry" && i+1 < len(args):
			if n, err := strconv.Atoi(args[i+1]); err == nil {
				payload.RetryAfter = n
			}
			i++
		case strings.HasPrefix(args[i], "--retry="):
			if n, err := strconv.Atoi(strings.TrimPrefix(args[i], "--retry=")); err == nil {
				payload.RetryAfter = n
			}
		case args[i] == "--secret" && i+1 < len(args):
			payload.Secret = args[i+1]
			i++
		case strings.HasPrefix(args[i], "--secret="):
			payload.Secret = strings.TrimPrefix(args[i], "--secret=")
		case args[i] == "--allow" && i+1 < len(args):
			payload.AllowedIPs = append(payload.AllowedIPs, strings.Split(args[i+1], ",")...)
			i++
		}
	}
	if err := c.app.Maintenance().Enable(payload); err != nil {
		return err
	}
	fmt.Println("Application is now in maintenance mode.")
	return nil
}

type UpCommand struct {
	app *kernel.Application
}

func (c *UpCommand) Name() string        { return "up" }
func (c *UpCommand) Description() string { return "Bring the application out of maintenance mode" }
func (c *UpCommand) Handle(args []string) error {
	if err := c.app.Bootstrap(); err != nil {
		return err
	}
	if err := c.app.Maintenance().Disable(); err != nil {
		return err
	}
	fmt.Println("Application is now live.")
	return nil
}
