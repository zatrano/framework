package console

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/zatrano/framework/v2/console/describe"
	"github.com/zatrano/framework/v2/kernel"
)

func registerAgentsCommand(console *Application, app *kernel.Application) {
	_ = app
	console.Register(&AgentsGenerateCommand{})
}

// AgentsGenerateCommand writes AGENTS.md from the live describe document.
type AgentsGenerateCommand struct {
	out io.Writer
}

func (c *AgentsGenerateCommand) Name() string { return "agents:generate" }
func (c *AgentsGenerateCommand) Description() string {
	return "Generate AGENTS.md from zatrano describe (do not edit the file by hand)"
}

func (c *AgentsGenerateCommand) writer() io.Writer {
	if c.out != nil {
		return c.out
	}
	return os.Stdout
}

func (c *AgentsGenerateCommand) Handle(args []string) error {
	if hasFlag(args, "--help", "-h") {
		fmt.Fprintln(c.writer(), "Usage: zatrano agents:generate [path]")
		return nil
	}
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			continue
		}
		root = a
		break
	}
	path, err := WriteAgentsMarkdown(root)
	if err != nil {
		return err
	}
	fmt.Fprintf(c.writer(), "Wrote %s\n", path)
	return nil
}

// WriteAgentsMarkdown renders describe output into root/AGENTS.md.
func WriteAgentsMarkdown(root string) (string, error) {
	return describe.WriteAgentsMarkdown(root)
}

// RenderAgentsMarkdown turns a describe document into AGENTS.md (deterministic).
func RenderAgentsMarkdown(doc *describe.DescribeDocument) string {
	return describe.RenderAgentsMarkdown(doc)
}
