package console

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/zatrano/framework/kernel"
)

// Application is the console kernel.
type Application struct {
	app      *kernel.Application
	commands map[string]Command
}

// Command is a CLI command.
type Command interface {
	Name() string
	Description() string
	Handle(args []string) error
}

// New creates a console application.
func New(app *kernel.Application) *Application {
	console := &Application{
		app:      app,
		commands: make(map[string]Command),
	}
	console.Register(
		&ServeCommand{app: app},
		&ListCommand{console: console},
		&MakeControllerCommand{app: app},
		&MakeMiddlewareCommand{app: app},
		&KeyGenerateCommand{app: app},
		&AboutCommand{app: app},
	)
	registerCacheCommands(console, app)
	registerRequestCommands(console, app)
	registerRuleCommands(console, app)
	registerStorageCommands(console, app)
	registerMakeProviderCommand(console, app)
	registerServiceCommands(console, app)
	registerExceptionCommands(console, app)
	registerUtilityCommands(console, app)
	registerEnvCommands(console, app)
	registerDeployCommands(console, app)
	registerMakeCommand(console, app)
	registerPackageCommands(console, app)
	registerNewCommand(console, app)
	registerDescribeCommand(console, app)
	registerDoctorCommand(console, app)
	registerAgentsCommand(console, app)
	registerAddonCLI(console, app)
	return console
}

// Register registers commands.
func (c *Application) Register(commands ...Command) {
	for _, command := range commands {
		c.commands[command.Name()] = command
	}
}

// Run executes the console application.
func (c *Application) Run(args []string) error {
	if len(args) == 0 {
		return c.commands["list"].Handle(nil)
	}

	name := args[0]
	command, ok := c.commands[name]
	if !ok {
		return fmt.Errorf("command [%s] not defined", name)
	}
	return command.Handle(args[1:])
}

// Commands returns registered commands.
func (c *Application) Commands() map[string]Command {
	return c.commands
}

type ListCommand struct {
	console *Application
}

func (c *ListCommand) Name() string        { return "list" }
func (c *ListCommand) Description() string { return "List all available commands" }
func (c *ListCommand) Handle(args []string) error {
	fmt.Println("ZATRANO Console")
	fmt.Println()
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for name, command := range c.console.commands {
		fmt.Fprintf(w, "  %s\t%s\n", name, command.Description())
	}
	return w.Flush()
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
			addr = ":" + args[i+1]
			i++
		}
		if strings.HasPrefix(args[i], "--host=") {
			host := strings.TrimPrefix(args[i], "--host=")
			addr = host
		}
	}
	return c.app.Run(addr)
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

type KeyGenerateCommand struct {
	app *kernel.Application
}

func (c *KeyGenerateCommand) Name() string        { return "key:generate" }
func (c *KeyGenerateCommand) Description() string { return "Set the application key" }
func (c *KeyGenerateCommand) Handle(args []string) error {
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

type MakeControllerCommand struct {
	app *kernel.Application
}

func (c *MakeControllerCommand) Name() string        { return "make:controller" }
func (c *MakeControllerCommand) Description() string { return "Create a new controller class" }
func (c *MakeControllerCommand) Handle(args []string) error {
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
		return fmt.Errorf("controller name required")
	}
	name := strings.TrimSuffix(nameArgs[0], "Controller") + "Controller"
	dir := c.app.BasePath("app", "http", "controllers", pkg)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, toSnake(name)+".go")
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("controller already exists: %s", path)
	}
	content := fmt.Sprintf(`package %s

import . "github.com/zatrano/framework/http"

type %s struct{}

func (c *%s) Index(req *Request) *Response {
	return JSON(map[string]any{
		"message": "%s",
	})
}
`, pkg, name, name, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
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
	dir := c.app.BasePath("app", "http", "middleware")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, toSnake(name)+".go")
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("middleware already exists: %s", path)
	}
	content := fmt.Sprintf(`package middleware

import (
	. "github.com/zatrano/framework/http"
	. "github.com/zatrano/framework/routing"
)

func %s(next HandlerFunc) HandlerFunc {
	return func(req *Request) *Response {
		// ...
		return next(req)
	}
}
`, name)
	return os.WriteFile(path, []byte(content), 0o644)
}

func toSnake(name string) string {
	var b strings.Builder
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		b.WriteRune(r)
	}
	return strings.ToLower(b.String())
}

func toExported(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return name
	}
	runes := []rune(name)
	if runes[0] >= 'a' && runes[0] <= 'z' {
		runes[0] = runes[0] - 'a' + 'A'
	}
	return string(runes)
}
