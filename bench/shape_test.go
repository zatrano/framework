package bench

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/zatrano/rawhttp"
)

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

func loadShapeGolden(t *testing.T) shapeGolden {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(file), "serverconfig.golden"))
	if err != nil {
		t.Fatal(err)
	}
	var g shapeGolden
	if err := json.Unmarshal(raw, &g); err != nil {
		t.Fatal(err)
	}
	return g
}

func TestRunServerMatchesGolden(t *testing.T) {
	g := loadShapeGolden(t)
	srv := serverRunHead(func(*rawhttp.Ctx) {})
	if srv.ReadHeaderTimeout != time.Duration(g.ReadHeaderTimeoutNs) ||
		srv.ReadTimeout != time.Duration(g.ReadTimeoutNs) ||
		srv.WriteTimeout != time.Duration(g.WriteTimeoutNs) ||
		srv.IdleTimeout != time.Duration(g.IdleTimeoutNs) {
		t.Fatalf("timeouts header=%s read=%s write=%s idle=%s", srv.ReadHeaderTimeout, srv.ReadTimeout, srv.WriteTimeout, srv.IdleTimeout)
	}
	if srv.MaxHeaderBytes != g.MaxHeaderBytes || srv.MaxRequestBodySize != g.MaxRequestBodySize {
		t.Fatalf("limits header=%d body=%d", srv.MaxHeaderBytes, srv.MaxRequestBodySize)
	}
	if srv.KeepHijackedConns != g.KeepHijackedConns || srv.AllowUpgrade != g.AllowUpgrade {
		t.Fatalf("hijack=%v upgrade=%v", srv.KeepHijackedConns, srv.AllowUpgrade)
	}
	if srv.Concurrency != g.Concurrency || srv.ReadBufferSize != g.ReadBufferSize || srv.WriteBufferSize != g.WriteBufferSize {
		t.Fatalf("concurrency=%d readBuf=%d writeBuf=%d", srv.Concurrency, srv.ReadBufferSize, srv.WriteBufferSize)
	}
	if (srv.HeaderReceived != nil) != g.HeaderReceived {
		t.Fatalf("headerReceived set=%v", srv.HeaderReceived != nil)
	}
	if (srv.ConnState != nil) != g.ConnStateSet {
		t.Fatalf("connState set=%v", srv.ConnState != nil)
	}
	if g.TrustedProxiesNil && srv.TrustedProxies != nil {
		t.Fatalf("trusted proxies %v", srv.TrustedProxies)
	}
}

func TestHeadFastPathMatchesGolden(t *testing.T) {
	g := loadShapeGolden(t)
	limits := map[string]int64{}
	for _, c := range g.Ceilings {
		if c.RouteBodyLimit > 0 {
			limits[strings.ToUpper(c.Method)+" "+c.Path] = c.RouteBodyLimit
		}
	}
	for _, c := range g.Ceilings {
		got := ceiling(c.Method, c.Path, c.ContentType, c.ContentLength, limits)
		if got.MaxRequestBodySize != c.MaxRequestBodySize {
			t.Fatalf("%s ceiling=%d want %d", c.Name, got.MaxRequestBodySize, c.MaxRequestBodySize)
		}
	}
	ctx := &rawhttp.Ctx{Method: []byte("GET"), Path: []byte("/plaintext")}
	if headFastPath(ctx).MaxRequestBodySize != 32<<20 {
		t.Fatalf("GET hook=%d", headFastPath(ctx).MaxRequestBodySize)
	}
	var captured rawhttp.RequestConfig
	srv := &rawhttp.Server{
		Handler: func(ctx *rawhttp.Ctx) {
			captured = ceilingFromCtx(ctx, limits)
			ctx.SetStatusCode(204)
		},
		HeaderReceived: func(ctx *rawhttp.Ctx) rawhttp.RequestConfig {
			captured = ceilingFromCtx(ctx, limits)
			return captured
		},
		ReadTimeout:  -1,
		WriteTimeout: -1,
		IdleTimeout:  -1,
	}
	body := "POST /hook HTTP/1.1\r\nHost: bench\r\nContent-Type: application/json\r\nContent-Length: 2\r\n\r\n{}"
	buf := &bytes.Buffer{}
	if err := srv.ServeConn(&memConn{r: bytes.NewReader([]byte(body)), w: buf}); err != nil && buf.Len() == 0 {
		t.Fatal(err)
	}
	if captured.MaxRequestBodySize != 4096 {
		t.Fatalf("POST /hook hook=%d", captured.MaxRequestBodySize)
	}
}
