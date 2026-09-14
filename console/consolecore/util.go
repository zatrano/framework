package consolecore

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/zatrano/framework/v2/kernel"
)

// HasFlag reports whether args contain any of the given flags.
func HasFlag(args []string, flags ...string) bool {
	set := map[string]bool{}
	for _, f := range flags {
		set[f] = true
	}
	for _, a := range args {
		if set[a] {
			return true
		}
	}
	return false
}

// FormatFromArgs reads --format / --json from CLI args.
func FormatFromArgs(args []string) (string, error) {
	format := "text"
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--format" && i+1 < len(args):
			format = args[i+1]
			i++
		case strings.HasPrefix(a, "--format="):
			format = strings.TrimPrefix(a, "--format=")
		case a == "--json":
			format = "json"
		}
	}
	format = strings.ToLower(strings.TrimSpace(format))
	switch format {
	case "", "text", "pretty":
		return "text", nil
	case "json":
		return "json", nil
	default:
		return "", fmt.Errorf("unknown --format %q (want json or text)", format)
	}
}

// ToSnake converts ExportedName to exported_name.
func ToSnake(name string) string {
	var b strings.Builder
	for i, r := range name {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte('_')
		}
		b.WriteRune(r)
	}
	return strings.ToLower(b.String())
}

// ToExported uppercases the first letter.
func ToExported(name string) string {
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

// ParseEnabledAddons reads var EnabledAddons = []string{...} from source.
func ParseEnabledAddons(src string) []string {
	re := regexp.MustCompile(`(?s)var EnabledAddons = \[\]string\{(.*?)\}`)
	m := re.FindStringSubmatch(src)
	if len(m) < 2 {
		return nil
	}
	out := []string{}
	for _, line := range strings.Split(m[1], "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimSuffix(line, ",")
		line = strings.Trim(line, `"`)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

// ModulePath reads the module path from root/go.mod.
func ModulePath(root string) (string, error) {
	b, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	return "", fmt.Errorf("console: module path not found in go.mod")
}

// FrameworkModuleRoot walks parents until it finds this framework module.
func FrameworkModuleRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("console: cannot resolve caller path")
	}
	dir := filepath.Dir(file)
	for i := 0; i < 12; i++ {
		b, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err == nil && bytes.Contains(b, []byte("module github.com/zatrano/framework/v2")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("console: framework module root not found")
}

// ConsumerModule is the application module path, or "your/module".
func ConsumerModule(app *kernel.Application) string {
	if app == nil {
		return "your/module"
	}
	mod, err := ModulePath(app.BasePath())
	if err != nil || strings.TrimSpace(mod) == "" || mod == "github.com/zatrano/framework/v2" {
		return "your/module"
	}
	return mod
}

// SeedEnvFromExample copies .env.example to .env when .env is missing.
func SeedEnvFromExample(root string) error {
	envPath := filepath.Join(root, ".env")
	if _, err := os.Stat(envPath); !os.IsNotExist(err) {
		return err
	}
	examplePath := filepath.Join(root, ".env.example")
	raw, err := os.ReadFile(examplePath)
	if err != nil {
		return err
	}
	return os.WriteFile(envPath, raw, 0o644)
}
