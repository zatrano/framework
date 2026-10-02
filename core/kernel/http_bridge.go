package kernel

import (
	"fmt"

	"github.com/zatrano/framework/v3/core/contracts"
	"github.com/zatrano/framework/v3/core/kernel/http"
)

// SetHTTPBridge registers template/session HTTP finalize behavior (nil clears).
// Illegal after applyHTTPBridgeMiddleware captures the bridge during Bootstrap.
func (app *Application) SetHTTPBridge(bridge contracts.HTTPBridge) {
	if app == nil {
		return
	}
	app.lifeMu.Lock()
	defer app.lifeMu.Unlock()
	if app.httpBridgeFrozen {
		panic("application: cannot change HTTP bridge after bootstrap")
	}
	app.httpBridge = bridge
}

// HTTPBridge returns the installed HTTP bridge, if any.
func (app *Application) HTTPBridge() contracts.HTTPBridge {
	if app == nil {
		return nil
	}
	app.lifeMu.Lock()
	defer app.lifeMu.Unlock()
	if app.httpBridgeFrozen {
		return app.httpBridgeCaptured
	}
	return app.httpBridge
}

func applyHTTPBridgeMiddleware(app *Application) error {
	if app == nil {
		return nil
	}
	app.lifeMu.Lock()
	if app.httpBridgeFrozen {
		app.lifeMu.Unlock()
		return nil
	}
	bridge := app.httpBridge
	app.httpBridgeCaptured = bridge
	app.httpBridgeFrozen = true
	app.lifeMu.Unlock()

	if bridge == nil {
		return nil
	}
	for i, mw := range bridge.Middleware() {
		fn, err := middlewareOf(mw)
		if err != nil {
			return fmt.Errorf("http bridge: middleware[%d]: %w", i, err)
		}
		app.router.Use(fn)
	}
	return nil
}

func finalizeHTTPBridge(app *Application, req *http.Request, resp *http.Response) *http.Response {
	if app == nil {
		return resp
	}
	bridge := app.httpBridgeCaptured
	if bridge == nil {
		// No SSR/session bridge: Template responses must not silently commit empty HTML.
		if resp != nil && resp.TemplateName() != "" {
			msg := "Canvas engine not bound (enable template / import framework/v3/core/ssr)"
			if app.IsDebug() {
				return http.DebugTemplateErrorHTML(msg)
			}
			return http.Abort(500, "Template rendering failed")
		}
		return resp
	}
	out := bridge.Finalize(req, resp)
	if next, ok := out.(*http.Response); ok && next != nil {
		return next
	}
	return resp
}
