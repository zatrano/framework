package view

import "strings"

// Component renders a view under components/{name}.
func (e *Engine) Component(name string, data map[string]any) (string, error) {
	name = strings.TrimSpace(name)
	name = strings.TrimPrefix(name, "components.")
	name = strings.TrimPrefix(name, "components/")
	return e.Render("components."+name, data)
}
