package ssr

import (
	"fmt"

	"github.com/zatrano/framework/v3/core/contracts"
	"github.com/zatrano/framework/v3/core/kernel/http"
)

type httpBridge struct {
	app contracts.App
}

func installHTTPBridge(app contracts.App) {
	app.SetHTTPBridge(&httpBridge{app: app})
}

func (b *httpBridge) Middleware() []any { return nil }

func (b *httpBridge) Finalize(reqAny any, respAny any) any {
	resp, _ := respAny.(*http.Response)
	return RenderTemplate(b.app, resp)
}

// RenderTemplate executes a template response via the bound Canvas engine.
// Missing engine with a TemplateName is fail-loud (no silent empty HTML).
func RenderTemplate(app contracts.App, resp *http.Response) *http.Response {
	if resp == nil || resp.TemplateName() == "" {
		return resp
	}
	engine := From(app)
	if engine == nil {
		msg := "Canvas engine not bound (enable template / import framework/v3/core/ssr)"
		if app != nil && app.IsDebug() {
			return DebugTemplateErrorHTML(msg)
		}
		return http.Abort(500, "Template rendering failed")
	}
	data := resp.TemplateData()
	if data == nil {
		data = map[string]any{}
	}
	html, err := engine.Render(resp.TemplateName(), data)
	if err != nil {
		if app != nil && app.IsDebug() {
			fallback := DebugTemplateErrorHTML(fmt.Sprint(err))
			if page := TryRender(app, "errors.template", map[string]any{"error": fmt.Sprint(err)}, 500, string(fallback.Content())); page != nil {
				return page
			}
			return fallback
		}
		return http.Abort(500, "Template rendering failed")
	}
	resp.SetContent([]byte(html), "text/html; charset=utf-8")
	return resp
}
