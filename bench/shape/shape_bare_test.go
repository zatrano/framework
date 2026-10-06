// Single-factor ablation of server configuration on a BARE rawhttp handler,
// driven through ServeConn with an in-memory connection. Copy into any module
// that requires github.com/zatrano/rawhttp (e.g. bench/). Run on the machine
// whose numbers you intend to publish:
//
//	go test -run '^$' -bench Shape -benchtime=2s -count=10 -benchmem
//
// Reading the result: Shape_Run - Shape_RunNoTimeouts = cost of the per-request
// time.Now()/deadline bookkeeping on THIS machine (the in-memory conn makes the
// SetDeadline calls themselves free; see deadline_cost_test.go for the real
// net.Conn cost). Shape_RunSmallHdr isolates the 1 MiB read buffer effect.
package shape

import (
	"bytes"
	"net"
	"testing"
	"time"

	"github.com/zatrano/rawhttp"
)

type addr struct{}

func (addr) Network() string { return "tcp" }
func (addr) String() string  { return "127.0.0.1:1" }

type memConn struct{ r *bytes.Reader }

func (c *memConn) Read(b []byte) (int, error)       { return c.r.Read(b) }
func (c *memConn) Write(b []byte) (int, error)      { return len(b), nil }
func (c *memConn) Close() error                     { return nil }
func (c *memConn) LocalAddr() net.Addr              { return addr{} }
func (c *memConn) RemoteAddr() net.Addr             { return addr{} }
func (c *memConn) SetDeadline(time.Time) error      { return nil }
func (c *memConn) SetReadDeadline(time.Time) error  { return nil }
func (c *memConn) SetWriteDeadline(time.Time) error { return nil }

const req = "GET /plaintext HTTP/1.1\r\nHost: localhost\r\nUser-Agent: bench\r\nAccept: */*\r\n\r\n"

func run(b *testing.B, s *rawhttp.Server) {
	s.Handler = func(ctx *rawhttp.Ctx) {
		ctx.SetContentType("text/plain; charset=utf-8")
		ctx.SetBodyString("Hello, World!")
	}
	c := &memConn{r: bytes.NewReader(bytes.Repeat([]byte(req), b.N))}
	b.ReportAllocs()
	b.ResetTimer()
	_ = s.ServeConn(c)
}

func BenchmarkShape_Old(b *testing.B) {
	run(b, &rawhttp.Server{ReadTimeout: -1, WriteTimeout: -1, IdleTimeout: -1, ReadHeaderTimeout: -1})
}

func BenchmarkShape_Run(b *testing.B) {
	run(b, &rawhttp.Server{ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 60 * time.Second,
		WriteTimeout: 60 * time.Second, IdleTimeout: 120 * time.Second, MaxHeaderBytes: 16 << 10, KeepHijackedConns: true})
}

func BenchmarkShape_RunNoTimeouts(b *testing.B) {
	run(b, &rawhttp.Server{ReadTimeout: -1, WriteTimeout: -1, IdleTimeout: -1, ReadHeaderTimeout: -1,
		MaxHeaderBytes: 1 << 20, KeepHijackedConns: true})
}

func BenchmarkShape_RunSmallHdr(b *testing.B) {
	run(b, &rawhttp.Server{ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 60 * time.Second,
		WriteTimeout: 60 * time.Second, IdleTimeout: 120 * time.Second, KeepHijackedConns: true})
}

func BenchmarkShape_OnlyTimeouts(b *testing.B) {
	run(b, &rawhttp.Server{ReadHeaderTimeout: 10 * time.Second, ReadTimeout: 60 * time.Second,
		WriteTimeout: 60 * time.Second, IdleTimeout: 120 * time.Second})
}
