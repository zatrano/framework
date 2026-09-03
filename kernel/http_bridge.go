package kernel

import (
	stdhttp "net/http"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

// HTTPBridge is installed by foundation (session/view/locale) without those
// packages being imported by the kernel boot path.
type HTTPBridge interface {
	Middleware() []routing.MiddlewareFunc
	Finalize(w stdhttp.ResponseWriter, req *http.Request, resp *http.Response) *http.Response
}

// SetHTTPBridge registers foundation HTTP behavior (nil clears).
func (app *Application) SetHTTPBridge(bridge HTTPBridge) {
	app.httpBridge = bridge
}

// HTTPBridge returns the foundation HTTP bridge, if any.
func (app *Application) HTTPBridge() HTTPBridge {
	return app.httpBridge
}
