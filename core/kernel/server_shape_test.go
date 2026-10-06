package kernel

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/zatrano/framework/v3/core/bootstrap/addons"
	"github.com/zatrano/framework/v3/core/kernel/http"
)

// shapeGolden is bench/serverconfig.golden. Run's rawhttp.Server and the
// header-time body cap must stay on this file; the bench module reads it too.
type shapeGolden struct {
	ReadHeaderTimeoutNs int64         `json:"readHeaderTimeoutNs"`
	ReadTimeoutNs       int64         `json:"readTimeoutNs"`
	WriteTimeoutNs      int64         `json:"writeTimeoutNs"`
	IdleTimeoutNs       int64         `json:"idleTimeoutNs"`
	MaxHeaderBytes      int           `json:"maxHeaderBytes"`
	MaxRequestBodySize  int           `json:"maxRequestBodySize"`
	KeepHijackedConns   bool          `json:"keepHijackedConns"`
	AllowUpgrade        bool          `json:"allowUpgrade"`
	Concurrency         int           `json:"concurrency"`
	ReadBufferSize      int           `json:"readBufferSize"`
	WriteBufferSize     int           `json:"writeBufferSize"`
	HeaderReceived      bool          `json:"headerReceived"`
	ConnStateSet        bool          `json:"connStateSet"`
	TrustedProxiesNil   bool          `json:"trustedProxiesNil"`
	Ceilings            []ceilingCase `json:"ceilings"`
}

type ceilingCase struct {
	Name               string `json:"name"`
	Method             string `json:"method"`
	Path               string `json:"path"`
	ContentType        string `json:"contentType"`
	ContentLength      int    `json:"contentLength"`
	RouteBodyLimit     int64  `json:"routeBodyLimit"`
	MaxRequestBodySize int    `json:"maxRequestBodySize"`
}

func TestRunServerMatchesBenchGolden(t *testing.T) {
	for _, key := range []string{
		"HTTP_READ_TIMEOUT",
		"HTTP_WRITE_TIMEOUT",
		"HTTP_IDLE_TIMEOUT",
		"HTTP_READ_HEADER_TIMEOUT",
		"HTTP_ALLOW_UPGRADE",
		"HTTP_MAX_INFLIGHT_BODY_BYTES",
		"HTTP_MAX_HEADER_BYTES",
		"TRUSTED_PROXIES",
		"MAX_BODY_BYTES",
		"MAX_UPLOAD_BYTES",
	} {
		t.Setenv(key, "")
	}
	g := loadBenchGolden(t)
	app := NewApplication(t.TempDir())
	plain := func(*http.Request) *http.Response { return http.Text("ok") }
	app.router.Get("/plaintext", plain)
	app.router.Post("/plaintext", plain)
	app.router.Add("HEAD", "/plaintext", plain)
	app.router.Add("OPTIONS", "/plaintext", plain)
	hook := app.router.Post("/hook", plain)
	for _, c := range g.Ceilings {
		if c.Path == "/hook" && c.RouteBodyLimit > 0 {
			hook.BodyLimit(c.RouteBodyLimit)
		}
	}
	if err := app.router.Freeze(); err != nil {
		t.Fatal(err)
	}
	srv, err := app.httpServer(ListenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if srv.ReadHeaderTimeout != time.Duration(g.ReadHeaderTimeoutNs) ||
		srv.ReadTimeout != time.Duration(g.ReadTimeoutNs) ||
		srv.WriteTimeout != time.Duration(g.WriteTimeoutNs) ||
		srv.IdleTimeout != time.Duration(g.IdleTimeoutNs) {
		t.Fatalf("timeouts header=%s read=%s write=%s idle=%s", srv.ReadHeaderTimeout, srv.ReadTimeout, srv.WriteTimeout, srv.IdleTimeout)
	}
	if srv.MaxHeaderBytes != g.MaxHeaderBytes || srv.MaxRequestBodySize != g.MaxRequestBodySize {
		t.Fatalf("limits header=%d body=%d", srv.MaxHeaderBytes, srv.MaxRequestBodySize)
	}
	if srv.KeepHijackedConns != g.KeepHijackedConns {
		t.Fatalf("hijack=%v", srv.KeepHijackedConns)
	}
	if _, linked := addons.Lookup("websocket"); !linked && srv.AllowUpgrade != g.AllowUpgrade {
		t.Fatalf("upgrade=%v golden=%v", srv.AllowUpgrade, g.AllowUpgrade)
	}
	if srv.AllowUpgrade != resolveAllowUpgrade(app, ListenOptions{}) {
		t.Fatalf("upgrade=%v resolver=%v", srv.AllowUpgrade, resolveAllowUpgrade(app, ListenOptions{}))
	}
	if srv.Concurrency != g.Concurrency || srv.ReadBufferSize != g.ReadBufferSize || srv.WriteBufferSize != g.WriteBufferSize {
		t.Fatalf("concurrency=%d readBuf=%d writeBuf=%d", srv.Concurrency, srv.ReadBufferSize, srv.WriteBufferSize)
	}
	if (srv.HeaderReceived != nil) != g.HeaderReceived || (srv.ConnState != nil) != g.ConnStateSet {
		t.Fatalf("headerReceived=%v connState=%v", srv.HeaderReceived != nil, srv.ConnState != nil)
	}
	if g.TrustedProxiesNil && len(srv.TrustedProxies) != 0 {
		t.Fatalf("trusted proxies %v", srv.TrustedProxies)
	}
	for _, c := range g.Ceilings {
		got := app.decideBodyLimit(c.Method, c.Path, c.ContentType, "", c.ContentLength)
		if got.MaxRequestBodySize != c.MaxRequestBodySize {
			t.Fatalf("%s ceiling=%d want %d", c.Name, got.MaxRequestBodySize, c.MaxRequestBodySize)
		}
	}
}

func loadBenchGolden(t *testing.T) shapeGolden {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	path := filepath.Join(filepath.Dir(file), "..", "..", "bench", "serverconfig.golden")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var g shapeGolden
	if err := json.Unmarshal(raw, &g); err != nil {
		t.Fatal(err)
	}
	if len(g.Ceilings) == 0 {
		t.Fatal("golden has no ceilings")
	}
	return g
}

func TestGoldenCoversBodyLimitMethods(t *testing.T) {
	g := loadBenchGolden(t)
	seen := map[string]bool{}
	var limited bool
	for _, c := range g.Ceilings {
		seen[strings.ToUpper(c.Method)] = true
		if c.RouteBodyLimit > 0 {
			limited = true
		}
	}
	for _, method := range []string{"GET", "HEAD", "POST", "OPTIONS"} {
		if !seen[method] {
			t.Fatalf("golden missing %s", method)
		}
	}
	if !limited {
		t.Fatal("golden missing a BodyLimit route")
	}
}
