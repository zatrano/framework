package ssr

import (
	"fmt"
	"html"

	"github.com/zatrano/framework/v3/core/kernel/http"
)

// TryRender eagerly renders a Canvas template when an engine is bound.
// On missing engine or render error it returns fallbackHTML (never http.Template —
// that would re-enter Finalize and risk recursion).
func TryRender(app App, name string, data map[string]any, status int, fallbackHTML string) *http.Response {
	eng := From(app)
	if eng == nil {
		return http.HTML(fallbackHTML).Status(status)
	}
	if data == nil {
		data = map[string]any{}
	}
	out, err := eng.Render(name, data)
	if err != nil {
		return http.HTML(fallbackHTML).Status(status)
	}
	return http.HTML(out).Status(status)
}

// DebugTemplateErrorHTML delegates to kernel/http (shared fail-loud page).
func DebugTemplateErrorHTML(msg string) *http.Response {
	return http.DebugTemplateErrorHTML(msg)
}

// FallbackHTTPErrorPage is the kernel-style inline error page (escaped).
func FallbackHTTPErrorPage(status int, title, body string) string {
	return fmt.Sprintf(`<!doctype html><html><head><meta charset="utf-8"><title>%d %s</title>
<style>body{font-family:ui-sans-serif,system-ui;background:#0b1220;color:#e8eef8;padding:2rem;max-width:880px;margin:0 auto}
h1{color:#3dd6c6}pre{white-space:pre-wrap;background:#121a2b;padding:1rem;border-radius:8px;overflow:auto}</style></head>
<body><h1>%d %s</h1><pre>%s</pre></body></html>`,
		status, html.EscapeString(title), status, html.EscapeString(title), html.EscapeString(body))
}
