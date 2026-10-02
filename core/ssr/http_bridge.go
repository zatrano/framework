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

// RenderTemplate executes a template response when the engine is bound.
func RenderTemplate(app contracts.App, resp *http.Response) *http.Response {
	if resp == nil || resp.TemplateName() == "" {
		return resp
	}
	engine := From(app)
	if engine == nil {
		return resp
	}
	data := resp.TemplateData()
	if data == nil {
		data = map[string]any{}
	}
	html, err := engine.Render(resp.TemplateName(), data)
	if err != nil {
		if app.IsDebug() {
			return http.HTML(fmt.Sprintf("<h1>Template Error</h1><pre>%v</pre>", err)).Status(500)
		}
		return http.Abort(500, "Template rendering failed")
	}
	resp.SetContent([]byte(html), "text/html; charset=utf-8")
	return resp
}
