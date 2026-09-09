package console

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/zatrano/framework/v2/console/generator"
	"github.com/zatrano/framework/v2/kernel"
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
		&VersionCommand{},
	)
	registerCacheCommands(console, app)
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
	registerAddCommands(console, app)
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
	switch name {
	case "--help", "-h", "help":
		return c.commands["list"].Handle(nil)
	case "--version", "-v":
		return c.commands["version"].Handle(nil)
	}
	command, ok := c.commands[name]
	if !ok {
		return unknownCommandError(name)
	}
	return command.Handle(args[1:])
}

// packageOwnedCLI maps kernel-absent commands to catalog package names.
// The CLI process does not import those packages; this is a catalog hint only.
var packageOwnedCLI = map[string]string{
	"make:request": "validation",
	"make:rule":    "validation",
}

func unknownCommandError(name string) error {
	if pkg, ok := packageOwnedCLI[name]; ok {
		if _, exists := catalogLookup(pkg); exists {
			return fmt.Errorf("%s requires the %s package, which is not imported in this process.\nKernel CLI does not import packages. From an application that imports %s:\n  go run ./cmd/app %s\nEnable with: go run ./cmd/app package:enable %s", name, pkg, pkg, name, pkg)
		}
	}
	return fmt.Errorf("command [%s] not defined\nNext: run zatrano --help (or list) for available commands", name)
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
	names := make([]string, 0, len(c.console.commands))
	for name := range c.console.commands {
		names = append(names, name)
	}
	sort.Strings(names)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, name := range names {
		command := c.console.commands[name]
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
	path := filepath.Join(c.app.BasePath("app", "http", "controllers", pkg), toSnake(name)+".go")
	content := fmt.Sprintf(`package %s

import . "github.com/zatrano/framework/v2/kernel/http"

type %s struct{}

func (c *%s) Index(req *Request) *Response {
	return JSON(map[string]any{
		"message": "%s",
	})
}
`, pkg, name, name, name)
	if err := generator.WriteExclusive(path, content); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("controller already exists: %s", path)
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
	. "github.com/zatrano/framework/v2/kernel/http"
	. "github.com/zatrano/framework/v2/kernel/routing"
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
