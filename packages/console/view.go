package console

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zatrano/framework/core"
)

func registerViewCommands(console *Application, app *core.Application) {
	console.Register(&MakeViewCommand{app: app})
}

type MakeViewCommand struct {
	app *core.Application
}

func (c *MakeViewCommand) Name() string        { return "make:view" }
func (c *MakeViewCommand) Description() string { return "Create a view template scaffold" }
func (c *MakeViewCommand) Handle(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("view name required")
	}
	name := strings.ReplaceAll(args[0], "\\", "/")
	name = strings.Trim(name, "/")
	name = strings.TrimSuffix(name, ".html")
	parts := strings.Split(name, ".")
	for i, part := range parts {
		parts[i] = toSnake(toExported(part))
		parts[i] = strings.ReplaceAll(parts[i], "_", "-")
		if parts[i] == "" {
			parts[i] = strings.ToLower(part)
		}
	}
	rel := strings.Join(parts, string(os.PathSeparator))
	dir := c.app.BasePath("views", filepath.Dir(rel))
	if filepath.Dir(rel) == "." {
		dir = c.app.BasePath("views")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	base := filepath.Base(rel)
	path := filepath.Join(dir, base+".html")
	content := fmt.Sprintf(`@extends('layouts.app')

@section('title', '%s')

@section('content')
  <h1>%s</h1>
  <form method="POST" action="/">
    @csrf
  </form>
@endsection
`, base, toExported(strings.ReplaceAll(base, "-", " ")))
	if strings.HasPrefix(strings.ToLower(strings.ReplaceAll(name, "\\", "/")), "auth/") ||
		strings.HasPrefix(strings.ToLower(name), "auth.") ||
		strings.EqualFold(parts[0], "auth") {
		content = fmt.Sprintf(`@extends('layouts.auth')

@section('title', '%s')

@section('content')
  <h1>%s</h1>
@endsection
`, base, toExported(strings.ReplaceAll(base, "-", " ")))
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	fmt.Printf("View created: %s\n", path)
	return nil
}
