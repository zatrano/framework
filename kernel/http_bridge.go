package kernel

import (
	stdhttp "net/http"

	"github.com/zatrano/framework/contracts"
	"github.com/zatrano/framework/kernel/http"
	"github.com/zatrano/framework/kernel/routing"
)

// SetHTTPBridge registers view/session HTTP finalize behavior (nil clears).
func (app *Application) SetHTTPBridge(bridge contracts.HTTPBridge) {
	app.httpBridge = bridge
}

// HTTPBridge returns the installed HTTP bridge, if any.
func (app *Application) HTTPBridge() contracts.HTTPBridge {
	return app.httpBridge
}

func applyHTTPBridgeMiddleware(app *Application) {
	bridge := app.HTTPBridge()
	if bridge == nil {
		return
	}
	for _, mw := range bridge.Middleware() {
		if fn, ok := mw.(routing.MiddlewareFunc); ok {
			app.router.Use(fn)
		}
	}
}

func finalizeHTTPBridge(app *Application, w stdhttp.ResponseWriter, req *http.Request, resp *http.Response) *http.Response {
	bridge := app.HTTPBridge()
	if bridge == nil {
		return resp
	}
	out := bridge.Finalize(w, req, resp)
	if next, ok := out.(*http.Response); ok && next != nil {
		return next
	}
	return resp
}
