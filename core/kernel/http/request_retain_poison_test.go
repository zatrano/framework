//go:build rawhttp_poison

package http

import (
	"bufio"
	"io"
	"net"
	stdhttp "net/http"
	"testing"

	"github.com/zatrano/rawhttp"
)

// TestRetainedRequestBytesAfterHandler records what a handler may keep.
// Path and HeaderValue are copies. headerBytes aliases the request buffer
// and is filled with 0xDE after the response is written.
func TestRetainedRequestBytesAfterHandler(t *testing.T) {
	var (
		path     string
		copied   string
		borrowed []byte
	)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &rawhttp.Server{
		Handler: func(ctx *rawhttp.Ctx) {
			req := NewRequest(ctx)
			path = req.Path()
			copied, _ = req.HeaderValue("X-Trace")
			borrowed = req.headerBytes("X-Trace")
			ReleaseRequest(req)
			ctx.SetStatusCode(200)
			ctx.SetBodyString("ok")
		},
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	raw := "GET /kept HTTP/1.1\r\nHost: localhost\r\nX-Trace: abc\r\nConnection: close\r\n\r\n"
	if _, err := io.WriteString(conn, raw); err != nil {
		t.Fatal(err)
	}
	resp, err := stdhttp.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d", resp.StatusCode)
	}
	if path != "/kept" {
		t.Fatalf("path=%q", path)
	}
	if copied != "abc" {
		t.Fatalf("HeaderValue=%q", copied)
	}
	if len(borrowed) == 0 || !allPoison(borrowed) {
		t.Fatalf("headerBytes=%q", borrowed)
	}
}

func allPoison(b []byte) bool {
	for _, c := range b {
		if c != 0xDE {
			return false
		}
	}
	return true
}
