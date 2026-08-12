package console

import (
	"bufio"
	"fmt"
	"github.com/zatrano/framework/packages/cache"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/zatrano/framework/core"
)

func registerUtilityCommands(console *Application, app *core.Application) {
	console.Register(
		&InspireCommand{},
		&TinkerCommand{app: app},
	)
}

var quotes = []string{
	"Simplicity is the ultimate sophistication.",
	"Code is like humor. When you have to explain it, it’s bad.",
	"First, solve the problem. Then, write the code.",
	"Make it work, make it right, make it fast.",
	"Programs must be written for people to read.",
	"The only way to go fast is to go well.",
	"Talk is cheap. Show me the code.",
	"ZATRANO: ship expressive Go apps with joy.",
}

type InspireCommand struct{}

func (c *InspireCommand) Name() string        { return "inspire" }
func (c *InspireCommand) Description() string { return "Display an inspiring quote" }
func (c *InspireCommand) Handle(args []string) error {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	fmt.Println()
	fmt.Printf("  \"%s\"\n", quotes[r.Intn(len(quotes))])
	fmt.Println()
	return nil
}

type TinkerCommand struct {
	app *core.Application
}

func (c *TinkerCommand) Name() string        { return "tinker" }
func (c *TinkerCommand) Description() string { return "Interact with the application" }
func (c *TinkerCommand) Handle(args []string) error {
	if err := c.app.Bootstrap(); err != nil {
		return err
	}

	fmt.Println("ZATRANO tinker (type help, exit to quit)")
	scanner := bufio.NewScanner(os.Stdin)

	runLine := func(line string) bool {
		line = strings.TrimSpace(line)
		if line == "" {
			return true
		}
		parts := strings.Fields(line)
		cmd := strings.ToLower(parts[0])
		rest := ""
		if len(parts) > 1 {
			rest = strings.Join(parts[1:], " ")
		}

		switch cmd {
		case "exit", "quit", "q":
			return false
		case "help", "?":
			fmt.Println("  help")
			fmt.Println("  app")
			fmt.Println("  config <key>")
			fmt.Println("  env <key>")
			fmt.Println("  hash <value>")
			fmt.Println("  encrypt <value>")
			fmt.Println("  decrypt <payload>")
			fmt.Println("  url <path>")
			fmt.Println("  route <name>")
			fmt.Println("  cache:get <key>")
			fmt.Println("  cache:put <key> <value>")
			fmt.Println("  metrics")
			fmt.Println("  exit")
		case "app":
			fmt.Printf("name=%s env=%s url=%s\n",
				c.app.Config().GetString("app.name", "ZATRANO"),
				c.app.Environment(),
				c.app.Config().GetString("app.url", "http://localhost:8080"),
			)
		case "config":
			if rest == "" {
				fmt.Println("usage: config <key>")
				return true
			}
			fmt.Println(c.app.Config().Get(rest))
		case "env":
			if rest == "" {
				fmt.Println("usage: env <key>")
				return true
			}
			fmt.Println(os.Getenv(rest))
		case "hash":
			if rest == "" {
				fmt.Println("usage: hash <value>")
				return true
			}
			out, err := c.app.Hash().Make(rest)
			if err != nil {
				fmt.Println("error:", err)
				return true
			}
			fmt.Println(out)
		case "encrypt":
			if rest == "" {
				fmt.Println("usage: encrypt <value>")
				return true
			}
			out, err := c.app.Encrypter().Encrypt(rest)
			if err != nil {
				fmt.Println("error:", err)
				return true
			}
			fmt.Println(out)
		case "decrypt":
			if rest == "" {
				fmt.Println("usage: decrypt <payload>")
				return true
			}
			out, err := c.app.Encrypter().Decrypt(rest)
			if err != nil {
				fmt.Println("error:", err)
				return true
			}
			fmt.Println(out)
		case "url":
			if rest == "" {
				rest = "/"
			}
			fmt.Println(c.app.URL().To(rest))
		case "route":
			if rest == "" {
				fmt.Println("usage: route <name>")
				return true
			}
			u, err := c.app.URL().Route(rest)
			if err != nil {
				fmt.Println("error:", err)
				return true
			}
			fmt.Println(u)
		case "cache:get":
			if rest == "" {
				fmt.Println("usage: cache:get <key>")
				return true
			}
			value, ok := cache.From(c.app).Get(rest)
			if !ok {
				fmt.Println("nil")
				return true
			}
			fmt.Println(value)
		case "cache:put":
			fields := strings.SplitN(rest, " ", 2)
			if len(fields) < 2 {
				fmt.Println("usage: cache:put <key> <value>")
				return true
			}
			_ = cache.From(c.app).Put(fields[0], fields[1], 5*time.Minute)
			fmt.Println("ok")
		case "metrics":
			if c.app.Metrics() == nil {
				fmt.Println("metrics unavailable")
				return true
			}
			fmt.Printf("%#v\n", c.app.Metrics().Snapshot())
		default:
			fmt.Printf("unknown command %q (try help)\n", cmd)
		}
		return true
	}

	if len(args) > 0 {
		_ = runLine(strings.Join(args, " "))
		return nil
	}

	for {
		fmt.Print(">>> ")
		if !scanner.Scan() {
			fmt.Println()
			break
		}
		if !runLine(scanner.Text()) {
			break
		}
	}
	return scanner.Err()
}
