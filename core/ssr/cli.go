package ssr

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zatrano/framework/v3/core/bootstrap/addons"
	"github.com/zatrano/framework/v3/core/console/consolecore"
	"github.com/zatrano/framework/v3/core/contracts"
	"github.com/zatrano/framework/v3/core/kernel/dirs"
)

func Commands(app contracts.App) []addons.CLICommand {
	return []addons.CLICommand{
		{Name: "make:template", Description: "Create a Canvas template scaffold", Handle: (&MakeTemplateCommand{app: app}).Handle},
		{Name: "make:component", Description: "Create a Canvas component scaffold", Handle: (&MakeComponentCommand{app: app}).Handle},
	}
}

type MakeTemplateCommand struct {
	app contracts.App
}

func (c *MakeTemplateCommand) Handle(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("template name required")
	}
	layout := "app"
	layoutSet := false
	var nameArg string
	for _, arg := range args {
		if strings.HasPrefix(arg, "--layout=") {
			layout = strings.TrimSpace(strings.TrimPrefix(arg, "--layout="))
			layoutSet = true
			continue
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}
		if nameArg == "" {
			nameArg = arg
		}
	}
	if nameArg == "" {
		return fmt.Errorf("template name required")
	}
	if layout == "" {
		layout = "app"
	}
	name := strings.ReplaceAll(nameArg, "\\", "/")
	name = strings.Trim(name, "/")
	name = strings.TrimSuffix(name, ".html")
	parts := strings.Split(name, ".")
	for i, part := range parts {
		parts[i] = consolecore.ToSnake(consolecore.ToExported(part))
		parts[i] = strings.ReplaceAll(parts[i], "_", "-")
		if parts[i] == "" {
			parts[i] = strings.ToLower(part)
		}
	}
	isAuth := strings.EqualFold(parts[0], "auth")
	layoutName := "layouts.app"
	if isAuth {
		if !layoutSet {
			layout = "auth"
		}
		layoutName = "layouts." + layout
	} else if layoutSet {
		layoutName = "layouts." + layout
	}
	relParts := parts
	if isAuth {
		if parts[0] != "auth" {
			relParts = append([]string{"auth"}, parts...)
		}
	} else if parts[0] != "web" {
		relParts = append([]string{"web"}, parts...)
	}
	rel := strings.Join(relParts, string(os.PathSeparator))
	dir := filepath.Join(dirs.TemplatesDirForCreate(c.app), filepath.Dir(rel))
	if filepath.Dir(rel) == "." {
		dir = dirs.TemplatesDirForCreate(c.app)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	base := filepath.Base(rel)
	path := filepath.Join(dir, base+".html")
	title := consolecore.ToExported(strings.ReplaceAll(base, "-", " "))
	var content string
	if isAuth {
		content = fmt.Sprintf(`@extends('%s')

@section('title', '%s')

@section('content')
  <h1>%s</h1>
@endsection
`, layoutName, base, title)
	} else {
		content = fmt.Sprintf(`@extends('%s')

@section('title', '%s')

@section('content')
  <h1>%s</h1>
  <form method="POST" action="/">
    @csrf
  </form>
@endsection
`, layoutName, base, title)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("Canvas template created: %s\n", path)
	return nil
}

type MakeComponentCommand struct {
	app contracts.App
}

func (c *MakeComponentCommand) Handle(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("component name required")
	}
	name := args[0]
	slug := strings.TrimSuffix(consolecore.ToSnake(consolecore.ToExported(name)), "_component")
	slug = strings.ReplaceAll(slug, "_", "-")
	if slug == "" {
		slug = strings.ToLower(name)
	}
	dir := filepath.Join(dirs.TemplatesDirForCreate(c.app), "components")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, slug+".html")
	content := fmt.Sprintf(`<div class="component-%s">
  {{ index . "slot" }}
</div>
`, slug)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("Component created: %s\n", path)
	fmt.Printf("Render with: canvas.New(templatesDir).Component(%q, map[string]any{\"slot\": \"...\"})\n", slug)
	return nil
}
