package bench

import (
	"net"
	"os"
	"sync"

	"github.com/zatrano/framework/v3/core/contracts"
	"github.com/zatrano/framework/v3/core/kernel"
	khttp "github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/framework/v3/core/kernel/routing"
	"github.com/zatrano/rawhttp"
)

func init() {
	// Bootstrap refuses a missing or wrong-length APP_KEY. Production matches
	// the shipped stack: the public-file miss path does not Stat, and CORS
	// does not emit a wildcard origin.
	_ = os.Setenv("APP_KEY", "zatrano-ci-test-key-not-prod!!!!")
	_ = os.Setenv("APP_ENV", "production")
	_ = os.Setenv("APP_DEBUG", "false")
	_ = os.Setenv("LOG_LEVEL", "error")
}

func canonicalHook(ctx *rawhttp.Ctx) rawhttp.RequestConfig {
	limitApp.once.Do(func() {
		limitApp.app = kernel.NewApplication("")
	})
	return limitApp.app.HeaderBodyConfig(ctx)
}

var limitApp struct {
	once sync.Once
	app  *kernel.Application
}

func serveZatranoTier0(conn net.Conn) error {
	return serveZatranoTier0With(conn, serverOld)
}

// serveZatranoTier0Canon is the published tier-0 row: a frozen router, the
// same shape Bootstrap builds, and the Run-shaped server.
func serveZatranoTier0Canon(conn net.Conn) error {
	r, err := tier0CanonRouter()
	if err != nil {
		return err
	}
	srv := serverRunHead(func(ctx *rawhttp.Ctx) {
		req := khttp.NewRequest(ctx)
		resp := r.Dispatch(req)
		if resp == nil {
			ctx.SetStatusCode(404)
			ctx.SetBodyString("Not Found")
			return
		}
		_ = resp.Commit(ctx)
	})
	return srv.ServeConn(conn)
}

// serveZatranoTier0FrozenOld is the frozen router on the unlimited-timeout
// server. It is the baseline for the header-hook gate.
func serveZatranoTier0FrozenOld(conn net.Conn) error {
	r, err := tier0CanonRouter()
	if err != nil {
		return err
	}
	srv := serverOld(func(ctx *rawhttp.Ctx) {
		req := khttp.NewRequest(ctx)
		resp := r.Dispatch(req)
		if resp == nil {
			ctx.SetStatusCode(404)
			ctx.SetBodyString("Not Found")
			return
		}
		_ = resp.Commit(ctx)
	})
	return srv.ServeConn(conn)
}

func serveZatranoTier0Run(conn net.Conn) error {
	return serveZatranoTier0With(conn, serverRunHead)
}

func serveZatranoTier0RunV301(conn net.Conn) error {
	return serveZatranoTier0With(conn, serverRunV301)
}

func serveZatranoTier0With(conn net.Conn, newServer func(rawhttp.Handler) *rawhttp.Server) error {
	r := tier0Router()
	srv := newServer(func(ctx *rawhttp.Ctx) {
		req := khttp.NewRequest(ctx)
		resp := r.Dispatch(req)
		if resp == nil {
			ctx.SetStatusCode(404)
			ctx.SetBodyString("Not Found")
			return
		}
		_ = resp.Commit(ctx)
	})
	return srv.ServeConn(conn)
}

var tier0Once struct {
	once sync.Once
	r    *routing.Router
}

func tier0Router() *routing.Router {
	tier0Once.once.Do(func() {
		r := routing.New()
		r.Get("/plaintext", func(*khttp.Request) *khttp.Response {
			return khttp.Text(hello)
		})
		tier0Once.r = r
	})
	return tier0Once.r
}

var tier0Canon struct {
	once sync.Once
	r    *routing.Router
	err  error
}

func tier0CanonRouter() (*routing.Router, error) {
	tier0Canon.once.Do(func() {
		r := routing.New()
		r.Get("/plaintext", func(*khttp.Request) *khttp.Response {
			return khttp.Text(hello)
		})
		tier0Canon.err = r.Freeze()
		tier0Canon.r = r
	})
	return tier0Canon.r, tier0Canon.err
}

type routeProvider struct{}

func (routeProvider) Register(app contracts.App) error {
	app.Router().Get("/plaintext", func(*khttp.Request) *khttp.Response {
		return khttp.Text(hello)
	})
	return nil
}

func (routeProvider) Boot(contracts.App) error { return nil }

var tier1App struct {
	once sync.Once
	app  *kernel.Application
	err  error
}

func bootedApp() (*kernel.Application, error) {
	tier1App.once.Do(func() {
		dir, err := os.MkdirTemp("", "zat-bench")
		if err != nil {
			tier1App.err = err
			return
		}
		app := kernel.NewApplication(dir)
		app.RegisterProviders(routeProvider{})
		tier1App.err = app.Bootstrap()
		tier1App.app = app
	})
	return tier1App.app, tier1App.err
}

func serveZatranoTier1(conn net.Conn) error {
	return serveZatranoTier1With(conn, serverOld)
}

func serveZatranoTier1Run(conn net.Conn) error {
	return serveZatranoTier1With(conn, serverRunHead)
}

func serveZatranoTier1With(conn net.Conn, newServer func(rawhttp.Handler) *rawhttp.Server) error {
	app, err := bootedApp()
	if err != nil {
		return err
	}
	srv := newServer(func(ctx *rawhttp.Ctx) {
		app.Handle(ctx)
	})
	if srv.HeaderReceived != nil {
		srv.HeaderReceived = app.HeaderBodyConfig
	}
	return srv.ServeConn(conn)
}
