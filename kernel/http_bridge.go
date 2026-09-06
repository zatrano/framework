package kernel

import (
	"fmt"
	stdhttp "net/http"

	"github.com/zatrano/framework/contracts"
	"github.com/zatrano/framework/kernel/http"
)

// SetHTTPBridge registers view/session HTTP finalize behavior (nil clears).
func (app *Application) SetHTTPBridge(bridge contracts.HTTPBridge) {
	app.httpBridge = bridge
}

// HTTPBridge returns the installed HTTP bridge, if any.
func (app *Application) HTTPBridge() contracts.HTTPBridge {
	return app.httpBridge
}

func applyHTTPBridgeMiddleware(app *Application) error {
	bridge := app.HTTPBridge()
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
