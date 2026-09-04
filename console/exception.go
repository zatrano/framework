package console

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zatrano/framework/kernel"
)

func registerExceptionCommands(console *Application, app *kernel.Application) {
	console.Register(&MakeExceptionCommand{app: app})
}

type MakeExceptionCommand struct {
	app *kernel.Application
}

func (c *MakeExceptionCommand) Name() string { return "make:exception" }
func (c *MakeExceptionCommand) Description() string {
	return "Create a custom HTTP exception renderer scaffold"
}
func (c *MakeExceptionCommand) Handle(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("exception name required")
	}
	name := args[0]
	structName := toExported(name)
	if !strings.HasSuffix(strings.ToLower(structName), "exception") {
		structName += "Exception"
	}
	status := 422
	dir := c.app.BasePath("app", "exceptions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, toSnake(structName)+".go")
	content := fmt.Sprintf(`package exceptions

import (
	"github.com/zatrano/framework/kernel/exceptions"
	. "github.com/zatrano/framework/kernel/http"
)

// %s is a custom exception helper.
type %s struct {
	Message string
}

func (e *%s) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "%s"
}

// HTTPError converts to a framework HTTP error.
func (e *%s) HTTPError() *exceptions.HTTPError {
	return &exceptions.HTTPError{Status: %d, Message: e.Error()}
}

// Register%s registers a renderer for status %d.
func Register%s(h *exceptions.Handler) {
	h.RenderUsing(%d, func(req *Request, err error) *Response {
		msg := err.Error()
		if req.WantsJSON() {
			return JSON(map[string]any{"message": msg}).Status(%d)
		}
		return Abort(%d, msg)
	})
}
`, structName, structName, structName, structName, structName, status, structName, status, structName, status, status, status)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("Exception created: %s\n", path)
	fmt.Printf("Call exceptions.Register%s(app.Exceptions()) during boot.\n", structName)
	return nil
}
