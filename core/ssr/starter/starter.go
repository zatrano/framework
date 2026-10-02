package starter

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/zatrano/framework/v3/core/contracts"
	"github.com/zatrano/framework/v3/core/kernel/dirs"
)

//go:embed all:stubs
var stubFiles embed.FS

// Write writes layouts/app.html and web/welcome.html when missing, then switches
// a starter HomeHandler from http.HTML to http.Template("web.welcome").
func Write(app contracts.App) error {
	if app == nil {
		return nil
	}
	root := dirs.TemplatesDirForCreate(app)
	pairs := []struct {
		stub string
		dest []string
	}{
		{"stubs/layouts/app.html", []string{"layouts", "app.html"}},
		{"stubs/web/welcome.html", []string{"web", "welcome.html"}},
	}
	for _, pair := range pairs {
		body, err := stubFiles.ReadFile(pair.stub)
		if err != nil {
			return fmt.Errorf("Canvas stub %s: %w", pair.stub, err)
		}
		dst := filepath.Join(append([]string{root}, pair.dest...)...)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if _, err := os.Stat(dst); err == nil {
			continue
		}
		if err := os.WriteFile(dst, body, 0o644); err != nil {
			return err
		}
	}
	return switchStarterHomeToTemplate(app)
}

var starterHTMLReturn = regexp.MustCompile(`return\s+http\.HTML\("[^"]*"\)`)

func switchStarterHomeToTemplate(app contracts.App) error {
	if app == nil {
		return nil
	}
	path := filepath.Join(app.BasePath(), "app", "http", "handlers", "web", "home_handler.go")
	body, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	text := strings.ReplaceAll(string(body), "\r\n", "\n")
	if !strings.Contains(text, "type HomeHandler") {
		return nil
	}
	if strings.Contains(text, `http.Template("web.welcome")`) {
		return nil
	}
	if !starterHTMLReturn.MatchString(text) {
		return nil
	}
	text = starterHTMLReturn.ReplaceAllString(text, `return http.Template("web.welcome")`)
	text = strings.ReplaceAll(text, "Index returns a kernel HTML welcome page (the view package is opt-in).", "Index renders the web welcome Canvas template.")
	text = strings.ReplaceAll(text, "Index returns a kernel HTML welcome page (the template package is opt-in).", "Index renders the web welcome Canvas template.")
	return os.WriteFile(path, []byte(text), 0o644)
}
