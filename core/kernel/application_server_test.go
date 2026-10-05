package kernel

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/rawhttp"
)

func TestHTTPServerBodyLimits(t *testing.T) {
	app := NewApplication(t.TempDir())
	app.router.Post("/hook", func(req *http.Request) *http.Response {
		return http.Text("ok")
	}).BodyLimit(8 << 20)
	srv, err := app.httpServer(ListenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if srv.MaxRequestBodySize != http.MaxRequestBodySize() {
		t.Fatalf("MaxRequestBodySize=%d want %d", srv.MaxRequestBodySize, http.MaxRequestBodySize())
	}
	if srv.MaxRequestBodySize < 32<<20 {
		t.Fatalf("server ceiling %d is below the 32 MiB upload default", srv.MaxRequestBodySize)
	}
	if srv.HeaderReceived == nil {
		t.Fatal("HeaderReceived must apply the body cap")
	}
	// v0.2.2 keeps request headers pinned in the connection buffer, so a chunked
	// body larger than that buffer becomes 431 before the size cap is reached.
	// The cap itself is what this test checks; the reader limit is an engine fix.
	srv.ReadBufferSize = 40 << 20

	var hit atomic.Int32
	var lookups atomic.Int32
	bodyLimitLookupHook = func() { lookups.Add(1) }
	t.Cleanup(func() { bodyLimitLookupHook = nil })
	srv.Handler = func(ctx *rawhttp.Ctx) {
		hit.Add(1)
		ctx.SetStatusCode(200)
		ctx.SetBodyString("ok")
	}
	ln := serveTestServer(t, srv)
	addr := ln.Addr().String()

	status, body := postRaw(t, addr, "application/json", bytes.Repeat([]byte("x"), 1<<20))
	if status != 200 || body != "ok" || hit.Load() != 1 || lookups.Load() != 0 {
		t.Fatalf("1 MiB JSON status=%d body=%q hit=%d lookups=%d", status, body, hit.Load(), lookups.Load())
	}

	mpCT, mpBody := multipartPayload(t, 5<<20)
	status, body = postRaw(t, addr, mpCT, mpBody)
	if status != 200 || body != "ok" || hit.Load() != 2 || lookups.Load() != 0 {
		t.Fatalf("5 MiB multipart status=%d body=%q hit=%d lookups=%d", status, body, hit.Load(), lookups.Load())
	}

	status, body = postRaw(t, addr, "application/octet-stream", bytes.Repeat([]byte("o"), 20<<20))
	if status != 200 || body != "ok" || hit.Load() != 3 || lookups.Load() != 0 {
		t.Fatalf("20 MiB octet-stream status=%d body=%q hit=%d lookups=%d", status, body, hit.Load(), lookups.Load())
	}

	status, body = postRaw(t, addr, "image/png", bytes.Repeat([]byte("p"), 20<<20))
	if status != 200 || body != "ok" || hit.Load() != 4 || lookups.Load() != 0 {
		t.Fatalf("20 MiB png status=%d body=%q hit=%d lookups=%d", status, body, hit.Load(), lookups.Load())
	}

	status, body = postRawPath(t, addr, "/upload", "", bytes.Repeat([]byte("m"), 3<<20))
	if status != 200 || body != "ok" || hit.Load() != 5 || lookups.Load() != 0 {
		t.Fatalf("3 MiB missing content-type status=%d body=%q hit=%d lookups=%d", status, body, hit.Load(), lookups.Load())
	}

	// Oversized Content-Length is rejected before the body is read.
	before := hit.Load()
	status = postDeclared(t, addr, "application/json", 3<<20)
	if status != 413 || hit.Load() != before || lookups.Load() != 1 {
		t.Fatalf("3 MiB JSON status=%d hit=%d lookups=%d", status, hit.Load(), lookups.Load())
	}
	status = postDeclared(t, addr, "application/json; charset=utf-8", 3<<20)
	if status != 413 || hit.Load() != before {
		t.Fatalf("JSON charset status=%d hit=%d", status, hit.Load())
	}
	status = postDeclared(t, addr, "application/vnd.api+json", 3<<20)
	if status != 413 || hit.Load() != before {
		t.Fatalf("+json status=%d hit=%d", status, hit.Load())
	}
	status = postDeclared(t, addr, "application/x-www-form-urlencoded", 3<<20)
	if status != 413 || hit.Load() != before {
		t.Fatalf("urlencoded status=%d hit=%d", status, hit.Load())
	}
	status = postDeclared(t, addr, "text/plain", 3<<20)
	if status != 413 || hit.Load() != before {
		t.Fatalf("text status=%d hit=%d", status, hit.Load())
	}
	status = postDeclared(t, addr, "Multipart/Form-Data; boundary=X", 33<<20)
	if status != 413 || hit.Load() != before {
		t.Fatalf("33 MiB multipart status=%d hit=%d", status, hit.Load())
	}
	status = postDeclared(t, addr, "multipart/form-data;boundary=x", 33<<20)
	if status != 413 || hit.Load() != before {
		t.Fatalf("multipart boundary status=%d hit=%d", status, hit.Load())
	}

	status, body = postRawPath(t, addr, "/hook", "application/json", bytes.Repeat([]byte("j"), 3<<20))
	if status != 200 || body != "ok" || hit.Load() != before+1 {
		t.Fatalf("webhook 3 MiB JSON status=%d body=%q hit=%d", status, body, hit.Load())
	}
	status = postDeclaredPath(t, addr, "/plain", "application/json", 3<<20)
	if status != 413 || hit.Load() != before+1 {
		t.Fatalf("default 3 MiB JSON status=%d hit=%d", status, hit.Load())
	}

	status = postChunked(t, addr, "/upload", "application/json", 3<<20)
	if status != 413 || hit.Load() != before+1 {
		t.Fatalf("chunked 3 MiB JSON status=%d hit=%d", status, hit.Load())
	}
	status = postChunked(t, addr, "/upload", "multipart/form-data; boundary=chunk", 33<<20)
	if status != 413 || hit.Load() != before+1 {
		t.Fatalf("chunked 33 MiB multipart status=%d hit=%d", status, hit.Load())
	}
}

func serveTestServer(t *testing.T, srv *rawhttp.Server) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = srv.Close() })
	go func() { _ = srv.Serve(ln) }()
	return ln
}

func multipartPayload(t *testing.T, n int) (string, []byte) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", "a.bin")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(bytes.Repeat([]byte("a"), n)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return w.FormDataContentType(), buf.Bytes()
}

func postDeclared(t *testing.T, addr, contentType string, n int) int {
	t.Helper()
	return postDeclaredPath(t, addr, "/upload", contentType, n)
}

func postDeclaredPath(t *testing.T, addr, path, contentType string, n int) int {
	t.Helper()
	var head bytes.Buffer
	head.WriteString("POST " + path + " HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n")
	if contentType != "" {
		head.WriteString("Content-Type: " + contentType + "\r\n")
	}
	head.WriteString("Content-Length: ")
	head.WriteString(itoa(n))
	head.WriteString("\r\n\r\n")
	status, _ := exchangeRaw(t, addr, head.Bytes())
	return status
}

func postRaw(t *testing.T, addr, contentType string, body []byte) (int, string) {
	t.Helper()
	return postRawPath(t, addr, "/upload", contentType, body)
}

func postRawPath(t *testing.T, addr, path, contentType string, body []byte) (int, string) {
	t.Helper()
	var head bytes.Buffer
	head.WriteString("POST " + path + " HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n")
	if contentType != "" {
		head.WriteString("Content-Type: " + contentType + "\r\n")
	}
	head.WriteString("Content-Length: ")
	head.WriteString(itoa(len(body)))
	head.WriteString("\r\n\r\n")
	raw := append(head.Bytes(), body...)
	return exchangeRaw(t, addr, raw)
}

func postChunked(t *testing.T, addr, path, contentType string, n int) int {
	t.Helper()
	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(45 * time.Second))
	var head bytes.Buffer
	fmt.Fprintf(&head, "POST %s HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n", path)
	if contentType != "" {
		fmt.Fprintf(&head, "Content-Type: %s\r\n", contentType)
	}
	head.WriteString("Transfer-Encoding: chunked\r\n\r\n")
	if _, err := c.Write(head.Bytes()); err != nil {
		t.Fatal(err)
	}
	got := make(chan []byte, 1)
	go func() {
		buf, _ := io.ReadAll(c)
		got <- buf
	}()
	chunk := bytes.Repeat([]byte("x"), 32<<10)
	left := n
	for left > 0 {
		nwrite := len(chunk)
		if nwrite > left {
			nwrite = left
		}
		if _, err := fmt.Fprintf(c, "%x\r\n", nwrite); err != nil {
			break
		}
		if _, err := c.Write(chunk[:nwrite]); err != nil {
			break
		}
		if _, err := io.WriteString(c, "\r\n"); err != nil {
			break
		}
		left -= nwrite
	}
	_, _ = io.WriteString(c, "0\r\n\r\n")
	if tc, ok := c.(*net.TCPConn); ok {
		_ = tc.CloseWrite()
	}
	var buf []byte
	select {
	case buf = <-got:
	case <-time.After(30 * time.Second):
		t.Fatal("timed out waiting for chunked response")
	}
	text := string(buf)
	if len(text) < 12 || !strings.HasPrefix(text, "HTTP/1.") {
		t.Fatalf("bad chunked response %q", text[:min(len(text), 80)])
	}
	return statusCode(text)
}

func exchangeRaw(t *testing.T, addr string, raw []byte) (int, string) {
	t.Helper()
	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(30 * time.Second))
	_, werr := c.Write(raw)
	if tc, ok := c.(*net.TCPConn); ok {
		_ = tc.CloseWrite()
	}
	buf, rerr := io.ReadAll(c)
	if len(buf) == 0 {
		if werr != nil {
			t.Fatal(werr)
		}
		if rerr != nil {
			t.Fatal(rerr)
		}
	}
	text := string(buf)
	if len(text) < 12 || !strings.HasPrefix(text, "HTTP/1.") {
		t.Fatalf("bad response %q", text[:min(len(text), 80)])
	}
	code := statusCode(text)
	body := ""
	if i := strings.Index(text, "\r\n\r\n"); i >= 0 {
		body = text[i+4:]
	}
	return code, body
}

func statusCode(text string) int {
	sp := strings.IndexByte(text[9:], ' ')
	if sp < 0 {
		return 0
	}
	code := 0
	for _, ch := range text[9 : 9+sp] {
		if ch < '0' || ch > '9' {
			break
		}
		code = code*10 + int(ch-'0')
	}
	return code
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
