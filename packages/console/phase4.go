package console

import (
	"github.com/zatrano/framework/packages/schedule"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/zatrano/framework/core"
)

func registerPhase4Commands(console *Application, app *core.Application) {
	console.Register(
		&ScheduleRunCommand{app: app},
		&ScheduleListCommand{app: app},
		&MakeNotificationCommand{app: app},
		&LangPublishCommand{app: app},
		&MakeLangCommand{app: app},
	)
}

type ScheduleRunCommand struct {
	app *core.Application
}

func (c *ScheduleRunCommand) Name() string        { return "schedule:run" }
func (c *ScheduleRunCommand) Description() string { return "Run the scheduled commands due now" }
func (c *ScheduleRunCommand) Handle(args []string) error {
	if err := c.app.Bootstrap(); err != nil {
		return err
	}
	errs := schedule.From(c.app).RunDue(time.Now())
	if len(errs) == 0 {
		fmt.Println("No scheduled commands are ready to run, or all completed successfully.")
		return nil
	}
	for _, err := range errs {
		fmt.Printf("Scheduled event failed: %v\n", err)
	}
	return errs[0]
}

type ScheduleListCommand struct {
	app *core.Application
}

func (c *ScheduleListCommand) Name() string        { return "schedule:list" }
func (c *ScheduleListCommand) Description() string { return "List all scheduled events" }
func (c *ScheduleListCommand) Handle(args []string) error {
	if err := c.app.Bootstrap(); err != nil {
		return err
	}
	events := schedule.From(c.app).Events()
	if len(events) == 0 {
		fmt.Println("No scheduled events defined.")
		return nil
	}
	for _, event := range events {
		fmt.Printf("%s\t%s\n", event.DisplayName(), event.Expression())
	}
	return nil
}

type MakeNotificationCommand struct {
	app *core.Application
}

func (c *MakeNotificationCommand) Name() string        { return "make:notification" }
func (c *MakeNotificationCommand) Description() string { return "Create a new notification class" }
func (c *MakeNotificationCommand) Handle(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("notification name required")
	}
	name := args[0]
	dir := c.app.BasePath("app", "notifications")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, toSnake(name)+".go")
	content := fmt.Sprintf(`package notifications

import (
	"github.com/zatrano/framework/packages/mail"
	"github.com/zatrano/framework/packages/notification"
)

type %s struct {
	notification.Base
	Message string
}

func (n *%s) Via() []string {
	return []string{"mail", "database"}
}

func (n *%s) ToMail(notifiable notification.Notifiable) *mail.Message {
	return &mail.Message{
		Subject: "%s",
		HTML:    "<p>" + n.Message + "</p>",
		Text:    n.Message,
	}
}

func (n *%s) ToDatabase(notifiable notification.Notifiable) map[string]any {
	return map[string]any{
		"message": n.Message,
	}
}

func (n *%s) ToPush(notifiable notification.Notifiable) map[string]any {
	return map[string]any{
		"title":   "%s",
		"message": n.Message,
	}
}
`, name, name, name, name, name, name, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("Notification created: %s\n", path)
	return nil
}
