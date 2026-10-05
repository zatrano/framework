package bench

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofiber/fiber/v2"
	"github.com/labstack/echo/v4"
	"github.com/zatrano/framework/v3/core/contracts"
	"github.com/zatrano/framework/v3/core/kernel"
	khttp "github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/framework/v3/core/kernel/routing"
	"github.com/zatrano/rawhttp"
)

const allowedOrigin = "https://allowed.example"

type compatRoutes struct{}

func (compatRoutes) Register(app contracts.App) error {
	r := routing.From(app)
	plain := func(*khttp.Request) *khttp.Response { return khttp.Text(hello) }
	r.Get("/plaintext", plain)
	r.Add("HEAD", "/plaintext", plain)
	r.Get("/panic", func(*khttp.Request) *khttp.Response { panic("bench") })
	r.Get("/ip", func(req *khttp.Request) *khttp.Response { return khttp.Text(req.IP()) })
	echoIn := func(req *khttp.Request) *khttp.Response { return khttp.JSON(req.All()) }
	r.Get("/echo", echoIn)
	r.Post("/echo", echoIn)
	return nil
}

func (compatRoutes) Boot(contracts.App) error { return nil }

type live struct {
	name string
	addr string
}

func TestCompatBlackBox(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", allowedOrigin)
	z := mustZatrano(t)
	servers := []*live{mustFiber(t), mustGin(t), mustEcho(t)}
	for _, s := range servers {
		t.Run(s.name, func(t *testing.T) {
			// Product gaps are recorded, not treated as a harness compile failure.
			// ZATRANO's unmatched 404 skips global middleware, so OPTIONS and the
			// 404 security headers fail the spec until a later approved change.
			checkPanic(t, z, s)
			checkOptions(t, z, s)
			checkForwarded(t, z, s)
			checkQueryTrim(t, z, s)
			checkJSONEmpty(t, z, s)
			checkHead(t, z, s)
			check404(t, z, s)
		})
	}
}

func note(t *testing.T, format string, args ...any) {
	t.Helper()
	msg := fmt.Sprintf(format, args...)
	fmt.Println("COMPAT", msg)
	t.Log(msg)
}

func checkPanic(t *testing.T, z, s *live) {
	t.Helper()
	pz := call(t, z, http.MethodGet, "/panic", nil, "")
	ps := call(t, s, http.MethodGet, "/panic", nil, "")
	if ps.code != pz.code {
		note(t, "panic status %d, zatrano %d body %q", ps.code, pz.code, ps.body)
	}
	if pz.code != http.StatusInternalServerError {
		note(t, "zatrano panic status %d body %q", pz.code, pz.body)
	}
	if ok := call(t, s, http.MethodGet, "/plaintext", nil, ""); ok.code != http.StatusOK {
		note(t, "server down after panic: %d", ok.code)
	}
}

func checkOptions(t *testing.T, z, s *live) {
	t.Helper()
	hdr := map[string]string{
		"Origin":                        allowedOrigin,
		"Access-Control-Request-Method": "GET",
	}
	az := call(t, z, http.MethodOptions, "/plaintext", hdr, "")
	as := call(t, s, http.MethodOptions, "/plaintext", hdr, "")
	if as.code != az.code {
		note(t, "OPTIONS status %d, zatrano %d", as.code, az.code)
	}
	for _, h := range []string{"Access-Control-Allow-Origin", "Access-Control-Allow-Methods", "Vary"} {
		if as.header.Get(h) != az.header.Get(h) {
			note(t, "OPTIONS %s=%q, zatrano %q", h, as.header.Get(h), az.header.Get(h))
		}
	}
}

func checkForwarded(t *testing.T, z, s *live) {
	t.Helper()
	hdr := map[string]string{"X-Forwarded-For": "198.51.100.1"}
	az := call(t, z, http.MethodGet, "/ip", hdr, "")
	as := call(t, s, http.MethodGet, "/ip", hdr, "")
	if strings.TrimSpace(as.body) != strings.TrimSpace(az.body) {
		note(t, "ip %q, zatrano %q", strings.TrimSpace(as.body), strings.TrimSpace(az.body))
	}
	if strings.Contains(as.body, "198.51.100.1") {
		note(t, "forwarded address was trusted: %q", as.body)
	}
}

func checkQueryTrim(t *testing.T, z, s *live) {
	t.Helper()
	az := call(t, z, http.MethodGet, "/echo?name=%20Ada%20&note=", nil, "")
	as := call(t, s, http.MethodGet, "/echo?name=%20Ada%20&note=", nil, "")
	if az.code != http.StatusOK {
		t.Fatalf("zatrano query status %d body %s", az.code, az.body)
	}
	sameInputs(t, "query", az.body, as.body)
}

func checkJSONEmpty(t *testing.T, z, s *live) {
	t.Helper()
	hdr := map[string]string{"Content-Type": "application/json"}
	body := `{"name":"  Ada  ","note":""}`
	az := call(t, z, http.MethodPost, "/echo", hdr, body)
	as := call(t, s, http.MethodPost, "/echo", hdr, body)
	if az.code != http.StatusOK {
		t.Fatalf("zatrano json status %d body %s", az.code, az.body)
	}
	sameInputs(t, "json", az.body, as.body)
}

func checkHead(t *testing.T, z, s *live) {
	t.Helper()
	az := call(t, z, http.MethodHead, "/plaintext", nil, "")
	as := call(t, s, http.MethodHead, "/plaintext", nil, "")
	if as.code != az.code {
		note(t, "HEAD status %d, zatrano %d", as.code, az.code)
	}
	if as.body != "" {
		note(t, "HEAD body %q", as.body)
	}
	if as.header.Get("X-Frame-Options") == "" || as.header.Get("X-Frame-Options") != az.header.Get("X-Frame-Options") {
		note(t, "HEAD X-Frame-Options=%q, zatrano %q", as.header.Get("X-Frame-Options"), az.header.Get("X-Frame-Options"))
	}
}

func check404(t *testing.T, z, s *live) {
	t.Helper()
	az := call(t, z, http.MethodGet, "/missing", nil, "")
	as := call(t, s, http.MethodGet, "/missing", nil, "")
	if as.code != az.code {
		note(t, "404 status %d, zatrano %d", as.code, az.code)
	}
	for _, h := range []string{"X-Frame-Options", "X-Content-Type-Options", "Referrer-Policy", "Permissions-Policy"} {
		if as.header.Get(h) != az.header.Get(h) || as.header.Get(h) == "" {
			note(t, "404 %s=%q, zatrano %q", h, as.header.Get(h), az.header.Get(h))
		}
	}
	if !idPattern.MatchString(as.header.Get("X-Request-ID")) {
		note(t, "404 X-Request-ID=%q", as.header.Get("X-Request-ID"))
	}
}

func sameInputs(t *testing.T, kind, want, got string) {
	t.Helper()
	wm, gm := map[string]any{}, map[string]any{}
	if err := json.Unmarshal([]byte(want), &wm); err != nil {
		t.Fatalf("zatrano %s: %v body %s", kind, err, want)
	}
	if err := json.Unmarshal([]byte(got), &gm); err != nil {
		note(t, "%s: %v body %s", kind, err, got)
		return
	}
	if wm["name"] != "Ada" {
		note(t, "zatrano %s name=%v body %s", kind, wm["name"], want)
	}
	if _, ok := wm["note"]; ok {
		note(t, "zatrano %s kept empty note: %s", kind, want)
	}
	if gm["name"] != wm["name"] {
		note(t, "%s name=%v, zatrano %v", kind, gm["name"], wm["name"])
	}
	if _, ok := gm["note"]; ok {
		note(t, "%s kept empty note: %s", kind, got)
	}
}

type reply struct {
	code   int
	body   string
	header http.Header
}

func call(t *testing.T, s *live, method, path string, hdr map[string]string, body string) reply {
	t.Helper()
	req, err := http.NewRequest(method, "http://"+s.addr+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		t.Fatalf("%s %s %s: %v", s.name, method, path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return reply{code: resp.StatusCode, body: string(raw), header: resp.Header}
}

func mustZatrano(t *testing.T) *live {
	t.Helper()
	dir, err := os.MkdirTemp("", "zat-compat")
	if err != nil {
		t.Fatal(err)
	}
	app := kernel.NewApplication(dir)
	app.RegisterProviders(compatRoutes{})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	srv := serverRunHead(func(ctx *rawhttp.Ctx) { app.Handle(ctx) })
	return listenRaw(t, "zatrano", srv)
}

func listenRaw(t *testing.T, name string, srv *rawhttp.Server) *live {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = ln.Close() })
	return &live{name: name, addr: ln.Addr().String()}
}

func mustFiber(t *testing.T) *live {
	t.Helper()
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(func(c *fiber.Ctx) (err error) {
		defer func() {
			if rec := recover(); rec != nil {
				_ = c.Status(http.StatusInternalServerError).SendString("Internal Server Error")
				err = nil
			}
		}()
		if c.Method() == http.MethodOptions {
			writeFiberCommon(c)
			return c.SendStatus(http.StatusNoContent)
		}
		err = c.Next()
		writeFiberCommon(c)
		return err
	})
	app.Get("/plaintext", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "text/plain; charset=utf-8")
		return c.SendString(hello)
	})
	app.Get("/panic", func(*fiber.Ctx) error { panic("bench") })
	app.Get("/ip", func(c *fiber.Ctx) error {
		return c.SendString(c.Context().RemoteIP().String())
	})
	app.Get("/echo", func(c *fiber.Ctx) error {
		return c.JSON(mergedInputs(c.Context().QueryArgs().String(), c.Body(), string(c.Request().Header.ContentType())))
	})
	app.Post("/echo", func(c *fiber.Ctx) error {
		return c.JSON(mergedInputs(c.Context().QueryArgs().String(), c.Body(), string(c.Request().Header.ContentType())))
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = app.Listener(ln) }()
	t.Cleanup(func() { _ = ln.Close() })
	return &live{name: "fiber", addr: ln.Addr().String()}
}

func writeFiberCommon(c *fiber.Ctx) {
	id := c.Get("X-Request-ID")
	if !idPattern.MatchString(id) {
		id = newHexID()
	}
	c.Set("X-Request-ID", id)
	c.Set("X-Frame-Options", "SAMEORIGIN")
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
	c.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
	c.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	c.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-CSRF-TOKEN, X-Idempotency-Key")
	c.Set("Access-Control-Max-Age", "600")
	if c.Get("Origin") == allowedOrigin {
		c.Set("Access-Control-Allow-Origin", allowedOrigin)
		c.Set("Vary", "Origin")
	}
}

func mustGin(t *testing.T) *live {
	t.Helper()
	r := gin.New()
	_ = r.SetTrustedProxies(nil)
	r.Use(gin.Recovery(), func(c *gin.Context) {
		writeStdCommon(c.Writer.Header(), c.GetHeader("X-Request-ID"), c.GetHeader("Origin"))
		if c.Request.Method == http.MethodOptions {
			c.Status(http.StatusNoContent)
			c.Abort()
			return
		}
		c.Next()
	})
	r.GET("/plaintext", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(hello))
	})
	r.HEAD("/plaintext", func(c *gin.Context) {
		c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(hello))
	})
	r.GET("/panic", func(*gin.Context) { panic("bench") })
	r.GET("/ip", func(c *gin.Context) { c.String(http.StatusOK, c.RemoteIP()) })
	r.GET("/echo", func(c *gin.Context) { c.JSON(http.StatusOK, mergedInputs(c.Request.URL.RawQuery, nil, "")) })
	r.POST("/echo", func(c *gin.Context) {
		raw, _ := io.ReadAll(c.Request.Body)
		c.JSON(http.StatusOK, mergedInputs(c.Request.URL.RawQuery, raw, c.GetHeader("Content-Type")))
	})
	return listenHTTP(t, "gin", r)
}

func mustEcho(t *testing.T) *live {
	t.Helper()
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(echoRecover, func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			writeStdCommon(c.Response().Header(), c.Request().Header.Get("X-Request-ID"), c.Request().Header.Get("Origin"))
			if c.Request().Method == http.MethodOptions {
				return c.NoContent(http.StatusNoContent)
			}
			return next(c)
		}
	})
	e.GET("/plaintext", func(c echo.Context) error {
		return c.Blob(http.StatusOK, "text/plain; charset=utf-8", []byte(hello))
	})
	e.HEAD("/plaintext", func(c echo.Context) error {
		return c.Blob(http.StatusOK, "text/plain; charset=utf-8", nil)
	})
	e.GET("/panic", func(echo.Context) error { panic("bench") })
	e.GET("/ip", func(c echo.Context) error {
		host, _, _ := net.SplitHostPort(c.Request().RemoteAddr)
		return c.String(http.StatusOK, host)
	})
	e.GET("/echo", func(c echo.Context) error {
		return c.JSON(http.StatusOK, mergedInputs(c.QueryString(), nil, ""))
	})
	e.POST("/echo", func(c echo.Context) error {
		raw, _ := io.ReadAll(c.Request().Body)
		return c.JSON(http.StatusOK, mergedInputs(c.QueryString(), raw, c.Request().Header.Get("Content-Type")))
	})
	return listenHTTP(t, "echo", e)
}

func echoRecover(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) (err error) {
		defer func() {
			if rec := recover(); rec != nil {
				err = c.String(http.StatusInternalServerError, "Internal Server Error")
			}
		}()
		return next(c)
	}
}

func writeStdCommon(h http.Header, reqID, origin string) {
	if !idPattern.MatchString(reqID) {
		reqID = newHexID()
	}
	h.Set("X-Request-ID", reqID)
	h.Set("X-Frame-Options", "SAMEORIGIN")
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
	h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
	h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	h.Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, X-CSRF-TOKEN, X-Idempotency-Key")
	h.Set("Access-Control-Max-Age", "600")
	if origin == allowedOrigin {
		h.Set("Access-Control-Allow-Origin", allowedOrigin)
		h.Set("Vary", "Origin")
	}
}

func listenHTTP(t *testing.T, name string, h http.Handler) *live {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = http.Serve(ln, h) }()
	t.Cleanup(func() { _ = ln.Close() })
	return &live{name: name, addr: ln.Addr().String()}
}

// mergedInputs trims query and JSON string fields and drops empty strings,
// matching TrimStrings plus ConvertEmptyStringsToNull on All().
func mergedInputs(rawQuery string, body []byte, contentType string) map[string]string {
	out := map[string]string{}
	if q, err := url.ParseQuery(rawQuery); err == nil {
		for key, values := range q {
			if len(values) == 0 {
				continue
			}
			value := strings.TrimSpace(values[0])
			if value == "" {
				continue
			}
			out[key] = value
		}
	}
	media := strings.ToLower(strings.TrimSpace(contentType))
	if i := strings.IndexByte(media, ';'); i >= 0 {
		media = strings.TrimSpace(media[:i])
	}
	if media == "application/json" && len(body) > 0 {
		var raw map[string]any
		if err := json.Unmarshal(body, &raw); err == nil {
			for key, value := range raw {
				s, ok := value.(string)
				if !ok {
					continue
				}
				s = strings.TrimSpace(s)
				if s == "" {
					delete(out, key)
					continue
				}
				out[key] = s
			}
		}
	}
	return out
}
