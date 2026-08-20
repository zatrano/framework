package console

import (
	"fmt"
	"github.com/zatrano/framework/packages/cache"
	"github.com/zatrano/framework/packages/queue"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zatrano/framework/core"
)

func registerSupportCommands(console *Application, app *core.Application) {
	console.Register(
		&QueueWorkCommand{app: app},
		&CacheClearCommand{app: app},
		&MakeJobCommand{app: app},
		&MakeListenerCommand{app: app},
		&MakeEventCommand{app: app},
	)
}

type QueueWorkCommand struct {
	app *core.Application
}

func (c *QueueWorkCommand) Name() string        { return "queue:work" }
func (c *QueueWorkCommand) Description() string { return "Start processing jobs on the queue" }
func (c *QueueWorkCommand) Handle(args []string) error {
	if err := c.app.Bootstrap(); err != nil {
		return err
	}
	once := false
	queueName := ""
	for i := 0; i < len(args); i++ {
		if args[i] == "--once" {
			once = true
		}
		if (args[i] == "--queue" || args[i] == "-q") && i+1 < len(args) {
			queueName = args[i+1]
			i++
		}
	}

	for {
		err := queue.From(c.app).Work(queueName)
		if err != nil {
			if strings.Contains(err.Error(), "no rows") || err.Error() == "sql: no rows in result set" {
				if once {
					fmt.Println("No jobs available.")
					return nil
				}
				time.Sleep(time.Second)
				continue
			}
			fmt.Printf("Job failed: %v\n", err)
			if once {
				return err
			}
			continue
		}
		fmt.Println("Processed a job.")
		if once {
			return nil
		}
	}
}

type CacheClearCommand struct {
	app *core.Application
}

func (c *CacheClearCommand) Name() string        { return "cache:clear" }
func (c *CacheClearCommand) Description() string { return "Flush the application cache" }
func (c *CacheClearCommand) Handle(args []string) error {
	if err := c.app.Bootstrap(); err != nil {
		return err
	}
	if err := cache.From(c.app).Flush(); err != nil {
		return err
	}
	fmt.Println("Application cache cleared successfully.")
	return nil
}

type MakeJobCommand struct {
	app *core.Application
}

func (c *MakeJobCommand) Name() string        { return "make:job" }
func (c *MakeJobCommand) Description() string { return "Create a new job class" }
func (c *MakeJobCommand) Handle(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("job name required")
	}
	name := args[0]
	dir := c.app.BasePath("app", "jobs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, toSnake(name)+".go")
	content := fmt.Sprintf(`package jobs

import "fmt"

const %sName = "%s"

// Tries is the suggested max attempts for this job.
const %sTries = 3

func Handle%s(payload map[string]any) error {
	fmt.Printf("Handling %s: %%v\n", payload)
	return nil
}
`, name, toSnake(name), name, name, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("Job created: %s\n", path)
	return nil
}

type MakeEventCommand struct {
	app *core.Application
}

func (c *MakeEventCommand) Name() string        { return "make:event" }
func (c *MakeEventCommand) Description() string { return "Create a new event class" }
func (c *MakeEventCommand) Handle(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("event name required")
	}
	name := args[0]
	dir := c.app.BasePath("app", "events")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, toSnake(name)+".go")
	content := fmt.Sprintf(`package events

const %sName = "%s"

type %s struct {
	Payload map[string]any
}
`, name, toSnake(name), name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("Event created: %s\n", path)
	return nil
}

type MakeListenerCommand struct {
	app *core.Application
}

func (c *MakeListenerCommand) Name() string        { return "make:listener" }
func (c *MakeListenerCommand) Description() string { return "Create a new event listener" }
func (c *MakeListenerCommand) Handle(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("listener name required")
	}
	name := args[0]
	dir := c.app.BasePath("app", "listeners")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, toSnake(name)+".go")
	content := fmt.Sprintf(`package listeners

import "fmt"

func %s(event any) error {
	fmt.Printf("%s received: %%v\n", event)
	return nil
}
`, name, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("Listener created: %s\n", path)
	return nil
}
