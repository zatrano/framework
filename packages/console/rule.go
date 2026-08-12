package console

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/zatrano/framework/core"
)

func registerRuleCommands(console *Application, app *core.Application) {
	console.Register(&MakeRuleCommand{app: app})
}

type MakeRuleCommand struct {
	app *core.Application
}

func (c *MakeRuleCommand) Name() string        { return "make:rule" }
func (c *MakeRuleCommand) Description() string { return "Create a custom validation rule scaffold" }
func (c *MakeRuleCommand) Handle(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("rule name required")
	}
	name := args[0]
	ruleName := toRuleName(name)
	structName := toExported(name)
	if !strings.HasSuffix(strings.ToLower(structName), "rule") {
		structName += "Rule"
	}
	dir := c.app.BasePath("app", "rules")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, toSnake(structName)+".go")
	content := fmt.Sprintf(`package rules

import "github.com/zatrano/framework/packages/validation"

// Register%s registers the "%s" validation rule.
func Register%s() {
	validation.Extend(%q, func(v *validation.Validator, field, value, param string) bool {
		// TODO: implement "%s" rule.
		_ = param
		return value != ""
	})
}
`, structName, ruleName, structName, ruleName, ruleName)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("Rule created: %s\n", path)
	fmt.Printf("Call rules.Register%s() during boot (e.g. AppServiceProvider).\n", structName)
	return nil
}

func toRuleName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, "-", "_")
	var b strings.Builder
	for i, r := range name {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
			continue
		}
		if r == ' ' {
			b.WriteByte('_')
			continue
		}
		b.WriteRune(r)
	}
	out := strings.ToLower(b.String())
	out = strings.TrimSuffix(out, "_rule")
	return out
}
