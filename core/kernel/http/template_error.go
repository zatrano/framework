package http

import (
	"fmt"
	"html"
)

// DebugTemplateErrorHTML is the fail-loud debug page when Template finalize cannot render.
// Callers must not return Template(...) from Template-error paths (Finalize recursion).
func DebugTemplateErrorHTML(msg string) *Response {
	page := fmt.Sprintf(`<!doctype html><html><head><meta charset="utf-8"><title>Template Error</title></head>
<body><h1>Template Error</h1><pre>%s</pre></body></html>`, html.EscapeString(msg))
	return HTML(page).Status(500)
}
