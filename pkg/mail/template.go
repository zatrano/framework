package mail

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/zatrano/framework/pkg/i18n"
)

// TemplateRenderer renders HTML email templates from a directory of .html files.
// Templates support layouts, partials (components), and i18n.
//
// Directory structure:
//
//	views/mails/
//	  layouts/
//	    default.html     ← base layout with {{ .Content }}
//	  welcome.html       ← mail template
//	  components/
//	    button.html      ← reusable partial
type TemplateRenderer struct {
	dir    string
	i18n   *i18n.Bundle
	funcMap template.FuncMap
}

// NewTemplateRenderer creates a renderer that loads templates from dir.
func NewTemplateRenderer(dir string, bundle *i18n.Bundle) *TemplateRenderer {
	r := &TemplateRenderer{
		dir:  dir,
		i18n: bundle,
		funcMap: template.FuncMap{
			"safe": func(s string) template.HTML { return template.HTML(s) },
		},
	}
	return r
}

// Render renders a mail template by name with the given data.
// If a layout is specified, the template is rendered inside the layout.
//
//	html, err := renderer.Render(ctx, "welcome", "default", map[string]any{"Name": "Alice"})
func (r *TemplateRenderer) Render(_ context.Context, name, layout string, data map[string]any) (string, error) {
	// Parse all partials/components first.
	tmpl := template.New("").Funcs(r.funcMap)

	componentsDir := filepath.Join(r.dir, "components")
	if info, err := os.Stat(componentsDir); err == nil && info.IsDir() {
		_ = filepath.WalkDir(componentsDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".html") {
				return err
			}
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			tplName := strings.TrimSuffix(d.Name(), ".html")
			_, err = tmpl.New("component_" + tplName).Parse(string(content))
			return err
		})
	}

	// Parse the main template.
	mainPath := filepath.Join(r.dir, name+".html")
	mainContent, err := os.ReadFile(mainPath)
	if err != nil {
		return "", fmt.Errorf("mail: template %q not found: %w", name, err)
	}
	mainTmpl, err := tmpl.New(name).Parse(string(mainContent))
	if err != nil {
		return "", fmt.Errorf("mail: parse template %q: %w", name, err)
	}

	// Render main template.
	var mainBuf bytes.Buffer
	if err := mainTmpl.Execute(&mainBuf, data); err != nil {
		return "", fmt.Errorf("mail: execute template %q: %w", name, err)
	}

	// If no layout, return the rendered content.
	if layout == "" {
		return mainBuf.String(), nil
	}

	// Parse and render layout with content injected.
	layoutPath := filepath.Join(r.dir, "layouts", layout+".html")
	layoutContent, err := os.ReadFile(layoutPath)
	if err != nil {
		return "", fmt.Errorf("mail: layout %q not found: %w", layout, err)
	}
	layoutTmpl, err := template.New(layout).Funcs(r.funcMap).Parse(string(layoutContent))
	if err != nil {
		return "", fmt.Errorf("mail: parse layout %q: %w", layout, err)
	}

	layoutData := make(map[string]any)
	for k, v := range data {
		layoutData[k] = v
	}
	layoutData["Content"] = template.HTML(mainBuf.String())

	var layoutBuf bytes.Buffer
	if err := layoutTmpl.Execute(&layoutBuf, layoutData); err != nil {
		return "", fmt.Errorf("mail: execute layout %q: %w", layout, err)
	}

	return layoutBuf.String(), nil
}
