package kernel

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"

	"github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/rawhttp"
)

// ablateMem is the free-deadline connection used by the bench ablation.
// Deadlines return without arming a timer, matching bench memConn.
type ablateMem struct {
	r *bytes.Reader
	w *bytes.Buffer
}

func (c *ablateMem) Read(p []byte) (int, error) { return c.r.Read(p) }
func (c *ablateMem) Write(p []byte) (int, error) {
	if c.w == nil {
		return len(p), nil
	}
	return c.w.Write(p)
}

func (c *ablateMem) Close() error { return nil }
func (c *ablateMem) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 9}
}
func (c *ablateMem) RemoteAddr() net.Addr             { return c.LocalAddr() }
func (c *ablateMem) SetDeadline(time.Time) error      { return nil }
func (c *ablateMem) SetReadDeadline(time.Time) error  { return nil }
func (c *ablateMem) SetWriteDeadline(time.Time) error { return nil }

// BenchmarkAblateRealHook is the production HeaderReceived hook
// (decideBodyLimit) on the frozen plaintext route. Timeouts stay off so the
// row isolates the hook. The bench module cannot call this method.
func BenchmarkAblateRealHook(b *testing.B) {
	app := NewApplication(b.TempDir())
	app.router.Get("/plaintext", func(*http.Request) *http.Response {
		return http.Text("Hello, World!")
	})
	if err := app.router.Freeze(); err != nil {
		b.Fatal(err)
	}
	req := []byte("GET /plaintext HTTP/1.1\r\nHost: bench\r\nX-Request-ID: bench-fixed-id\r\nConnection: keep-alive\r\n\r\n")
	srv := &rawhttp.Server{
		Handler: func(ctx *rawhttp.Ctx) {
			r := http.NewRequest(ctx)
			resp := app.router.Dispatch(r)
			if resp == nil {
				ctx.SetStatusCode(404)
				ctx.SetBodyString("Not Found")
				return
			}
			_ = resp.Commit(ctx)
		},
		ReadTimeout:    -1,
		WriteTimeout:   -1,
		IdleTimeout:    -1,
		HeaderReceived: app.headerBodyConfig,
	}
	var warm bytes.Buffer
	if err := srv.ServeConn(&ablateMem{r: bytes.NewReader(req), w: &warm}); err != nil && err != io.EOF {
		b.Fatalf("warmup: %v", err)
	}
	if warm.Len() == 0 {
		b.Fatal("warmup wrote nothing")
	}
	payload := bytes.Repeat(req, b.N)
	b.ReportAllocs()
	b.ResetTimer()
	_ = srv.ServeConn(&ablateMem{r: bytes.NewReader(payload)})
}
