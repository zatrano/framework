package main

import (
	"fmt"
	htmlpkg "html"
	stdhttp "net/http"
	"os"
	"time"

	"github.com/zatrano/framework/packages/hashing"
	zhttp "github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/middleware"
	"github.com/zatrano/framework/packages/middleware/csrf"
	"github.com/zatrano/framework/packages/routing"
	"github.com/zatrano/framework/packages/session"
)

// Minimal local app for OWASP ZAP / manual security scans.
// Usage: go run ./tests/securitydemo
func main() {
	addr := ":18080"
	if v := os.Getenv("SECURITY_DEMO_ADDR"); v != "" {
		addr = v
	}
	dir, err := os.MkdirTemp("", "zatrano-sec-demo-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	sessions := session.NewManager(dir, 60)
	hasher := hashing.New()
	demoHash, _ := hasher.Make("secret")
	router := routing.New()

	stack := []routing.MiddlewareFunc{
		middleware.SecurityHeadersWith(middleware.SecurityHeaderConfig{}),
		func(next routing.HandlerFunc) routing.HandlerFunc {
			return func(req *zhttp.Request) *zhttp.Response {
				bag, _ := sessions.Start(req.Cookie(sessions.CookieName()))
				req.SetSession(bag)
				resp := next(req)
				_ = sessions.Save(bag)
				if resp == nil {
					resp = zhttp.Text("")
				}
				return resp.WithCookie(&stdhttp.Cookie{
					Name:     sessions.CookieName(),
					Value:    bag.ID(),
					Path:     "/",
					HttpOnly: true,
					SameSite: stdhttp.SameSiteLaxMode,
				})
			}
		},
		csrf.Except(),
	}

	router.Get("/", func(req *zhttp.Request) *zhttp.Response {
		token := ""
		if sess := req.Session(); sess != nil {
			if t, ok := sess.Get("_csrf_token").(string); ok {
				token = t
			}
		}
		return zhttp.HTML(`<!doctype html><html><body>
<h1>ZATRANO security demo</h1>
<form method="POST" action="/login">
<input type="hidden" name="_token" value="` + htmlpkg.EscapeString(token) + `">
<label>email <input name="email" value="ada@example.com"></label>
<label>password <input name="password" type="password" value="secret"></label>
<button>Login</button>
</form>
<p><a href="/me">/me</a></p>
</body></html>`)
	})

	router.Post("/login", func(req *zhttp.Request) *zhttp.Response {
		if !hasher.Check(req.Input("password"), demoHash) {
			return zhttp.Text("invalid").Status(401)
		}
		req.Session().Put("user", req.Input("email"))
		return zhttp.Redirect("/me", 302)
	})

	router.Get("/me", func(req *zhttp.Request) *zhttp.Response {
		u := fmt.Sprint(req.Session().Get("user"))
		if u == "" || u == "<nil>" {
			return zhttp.Text("unauthenticated").Status(401)
		}
		return zhttp.HTML("<p>hello " + htmlpkg.EscapeString(u) + "</p>")
	})

	handler := stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		req := zhttp.NewRequest(r)
		h := router.Dispatch
		for i := len(stack) - 1; i >= 0; i-- {
			h = stack[i](h)
		}
		resp := h(req)
		if resp != nil {
			_ = resp.WriteTo(w)
		}
	})

	srv := &stdhttp.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	fmt.Println("security demo listening on", addr)
	if err := srv.ListenAndServe(); err != nil {
		panic(err)
	}
}
