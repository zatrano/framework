package console

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/zatrano/framework/v3/core/console/consolecore"
	"github.com/zatrano/framework/v3/core/console/describe"
	"github.com/zatrano/framework/v3/core/console/doctor"
	"github.com/zatrano/framework/v3/core/console/generator"
	"github.com/zatrano/framework/v3/core/console/pkgmanager"
	"github.com/zatrano/framework/v3/core/console/scaffold"
	"github.com/zatrano/framework/v3/core/kernel"
)

// Application is the console kernel.
type Application = consolecore.Application

// Command is a CLI command.
type Command = consolecore.Command

// New creates a console application.
func New(app *kernel.Application) *Application {
	c := consolecore.New(app)
	c.Unknown = unknownCommandError
	c.Register(
		&ServeCommand{app: app},
		&ListCommand{console: c},
		&MakeHandlerCommand{app: app},
		&MakeMiddlewareCommand{app: app},
		&KeyGenerateCommand{app: app},
		&AboutCommand{app: app},
		&VersionCommand{},
	)
	registerCacheCommands(c, app)
	registerStorageCommands(c, app)
	registerServiceCommands(c, app)
	registerExceptionCommands(c, app)
	registerUtilityCommands(c, app)
	registerEnvCommands(c, app)
	registerDeployCommands(c, app)
	scaffold.Register(c, app, WriteAgentsMarkdown, seedEnvFromExample)
	pkgmanager.Register(c, app)
	describe.Register(c, app)
	doctor.Register(c, app)
	registerAgentsCommand(c, app)
	registerAddonCLI(c, app)
	return c
}

// packageOwnedCLI maps kernel-absent commands to catalog package names.
// The CLI process does not import those packages; this is a catalog hint only.
var packageOwnedCLI = map[string]string{
	"make:request": "validation",
	"make:rule":    "validation",
}

func unknownCommandError(name string) error {
	if pkg, ok := packageOwnedCLI[name]; ok {
		if _, exists := describe.Lookup(pkg); exists {
			return fmt.Errorf("%s requires the %s package, which is not imported in this process.\nKernel CLI does not import packages. From an application that imports %s:\n  go run ./cmd/app %s\nEnable with: go run ./cmd/app package:enable %s", name, pkg, pkg, name, pkg)
		}
	}
	return fmt.Errorf("command [%s] not defined\nNext: run zatrano --help (or list) for available commands", name)
}

type ListCommand struct {
	console *Application
}

func (c *ListCommand) Name() string        { return "list" }
func (c *ListCommand) Description() string { return "List all available commands" }
func (c *ListCommand) Handle(args []string) error {
	return c.console.WriteCommandList()
}

type ServeCommand struct {
	app *kernel.Application
}

func (c *ServeCommand) Name() string        { return "serve" }
func (c *ServeCommand) Description() string { return "Serve the application on the HTTP server" }
func (c *ServeCommand) Handle(args []string) error {
	// Leave addr empty unless overridden so Application.Run can load .env
	// first and then resolve APP_PORT (default 8080).
	addr := ""
	for i := 0; i < len(args); i++ {
		if (args[i] == "--port" || args[i] == "-p") && i+1 < len(args) {
			raw := strings.TrimSpace(args[i+1])
			n, err := strconv.Atoi(raw)
			if err != nil || n < 0 || n > 65535 {
				return cliErr(ExitUsage, fmt.Errorf("serve --port expected a TCP port 0-65535, received %q", raw))
			}
			addr = ":" + strconv.Itoa(n)
			i++
		}
		if strings.HasPrefix(args[i], "--host=") {
			host := strings.TrimPrefix(args[i], "--host=")
			addr = host
		}
	}
	return classifyRuntimeError(c.app.Run(addr))
}

type AboutCommand struct {
	app *kernel.Application
}

func (c *AboutCommand) Name() string        { return "about" }
func (c *AboutCommand) Description() string { return "Display basic application information" }
func (c *AboutCommand) Handle(args []string) error {
	if err := c.app.Bootstrap(); err != nil {
		return err
	}
	fmt.Println("ZATRANO")
	fmt.Printf("  Name:\t%s\n", c.app.Config().GetString("app.name"))
	fmt.Printf("  Version:\t%s\n", c.app.Version())
	fmt.Printf("  Author:\tSerhan KARAKOÇ <serhankarakoc@gmail.com>\n")
	fmt.Printf("  Env:\t%s\n", c.app.Environment())
	fmt.Printf("  Debug:\t%v\n", c.app.IsDebug())
	fmt.Printf("  URL:\t%s\n", c.app.Config().GetString("app.url"))
	fmt.Printf("  Base path:\t%s\n", c.app.BasePath())
	return nil
}

type VersionCommand struct{}

func (c *VersionCommand) Name() string        { return "version" }
func (c *VersionCommand) Description() string { return "Print the ZATRANO framework version" }
func (c *VersionCommand) Handle(args []string) error {
	fmt.Println(productVersion())
	return nil
}

type KeyGenerateCommand struct{ app *kernel.Application }

func (c *KeyGenerateCommand) Name() string        { return "key:generate" }
func (c *KeyGenerateCommand) Description() string { return "Set the application key" }
func (c *KeyGenerateCommand) Handle(args []string) error {
	if err := seedEnvFromExample(c.app.BasePath()); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf(".env not found\nNext: copy .env.example to .env, then rerun key:generate")
		}
		return err
	}
	keyFile := c.app.BasePath(".env")
	raw, err := os.ReadFile(keyFile)
	if err != nil {
		return err
	}

	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return err
	}

	key := "base64:" + base64.StdEncoding.EncodeToString(random)
	content := string(raw)
	if strings.Contains(content, "APP_KEY=") {
		lines := strings.Split(content, "\n")
		for i, line := range lines {
			if strings.HasPrefix(line, "APP_KEY=") {
				lines[i] = "APP_KEY=" + key
			}
		}
		content = strings.Join(lines, "\n")
	} else {
		content += "\nAPP_KEY=" + key + "\n"
	}

	if err := os.WriteFile(keyFile, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Println("Application key set successfully.")
	return nil
}

type MakeHandlerCommand struct {
	app *kernel.Application
}

func (c *MakeHandlerCommand) Name() string        { return "make:handler" }
func (c *MakeHandlerCommand) Description() string { return "Create a new HTTP handler" }
func (c *MakeHandlerCommand) Handle(args []string) error {
	pkg := "web"
	var nameArgs []string
	for _, arg := range args {
		switch arg {
		case "--api":
			pkg = "api"
		case "--admin":
			pkg = "admin"
		default:
			if strings.HasPrefix(arg, "-") {
				continue
			}
			nameArgs = append(nameArgs, arg)
		}
	}
	if len(nameArgs) == 0 {
		return fmt.Errorf("handler name required")
	}
	name := strings.TrimSuffix(strings.TrimSuffix(nameArgs[0], "Handler"), "Controller") + "Handler"
	path := filepath.Join(c.app.BasePath("app", "http", "handlers", pkg), toSnake(name)+".go")
	content := handlerStub(pkg, name, handlerPresentation(c.app, pkg))
	if err := generator.WriteExclusive(path, content); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("handler already exists: %s", path)
		}
		return err
	}
	fmt.Printf("Created: %s\n", path)
	return nil
}

type MakeMiddlewareCommand struct {
	app *kernel.Application
}

func (c *MakeMiddlewareCommand) Name() string        { return "make:middleware" }
func (c *MakeMiddlewareCommand) Description() string { return "Create a new middleware class" }
func (c *MakeMiddlewareCommand) Handle(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("middleware name required")
	}
	name := args[0]
	path := filepath.Join(c.app.BasePath("app", "http", "middleware"), toSnake(name)+".go")
	content := fmt.Sprintf(`package middleware

import (
	. "github.com/zatrano/framework/v3/core/kernel/http"
	. "github.com/zatrano/framework/v3/core/kernel/routing"
)

func %s(next HandlerFunc) HandlerFunc {
	return func(req *Request) *Response {
		// ...
		return next(req)
	}
}
`, name)
	if err := generator.WriteExclusive(path, content); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("middleware already exists: %s", path)
		}
		return err
	}
	return nil
}

type handlerPresentationKind int

const (
	handlerJSON handlerPresentationKind = iota
	handlerTemplate
	handlerHTML
)

func handlerPresentation(app *kernel.Application, pkg string) handlerPresentationKind {
	if pkg == "api" {
		return handlerJSON
	}
	if consumerHasEnabledAddon(app, "template") {
		return handlerTemplate
	}
	if consumerScaffoldName(app) == "api" {
		return handlerJSON
	}
	return handlerHTML
}

func consumerHasEnabledAddon(app *kernel.Application, name string) bool {
	if app == nil {
		return false
	}
	body, err := os.ReadFile(app.BasePath("bootstrap", "enabled.go"))
	if err != nil {
		return false
	}
	for _, n := range consolecore.ParseEnabledAddons(string(body)) {
		if n == name {
			return true
		}
	}
	return false
}

func consumerScaffoldName(app *kernel.Application) string {
	if app == nil {
		return ""
	}
	body, err := os.ReadFile(app.BasePath("bootstrap", "scaffold.go"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(body), "\n") {
		trim := strings.TrimSpace(line)
		if !strings.HasPrefix(trim, "ScaffoldName") {
			continue
		}
		_, val, ok := strings.Cut(trim, "=")
		if !ok {
			return ""
		}
		return strings.Trim(strings.TrimSpace(val), `"`)
	}
	return ""
}

func handlerStub(pkg, name string, kind handlerPresentationKind) string {
	switch kind {
	case handlerJSON:
		return fmt.Sprintf(`package %s

import . "github.com/zatrano/framework/v3/core/kernel/http"

type %s struct{}

func (c *%s) Index(req *Request) *Response {
	return JSON(map[string]any{
		"message": %q,
	})
}
`, pkg, name, name, name)
	case handlerTemplate:
		view := toSnake(strings.TrimSuffix(name, "Handler")) + ".index"
		return fmt.Sprintf(`package %s

import . "github.com/zatrano/framework/v3/core/kernel/http"

type %s struct{}

func (c *%s) Index(req *Request) *Response {
	return Template(%q, map[string]any{})
}
`, pkg, name, name, view)
	default:
		return fmt.Sprintf(`package %s

import . "github.com/zatrano/framework/v3/core/kernel/http"

type %s struct{}

func (c *%s) Index(req *Request) *Response {
	return HTML("<h1>%s</h1>")
}
`, pkg, name, name, name)
	}
}

func toSnake(name string) string { return consolecore.ToSnake(name) }

func toExported(name string) string { return consolecore.ToExported(name) }
