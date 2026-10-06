package kernel

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zatrano/framework/v3/core/bootstrap/addons"
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
	// /hook's 8 MiB cap is tighter than the 32 MiB multipart default, so the
	// lookup runs even though this body is under that default.
	if status != 200 || body != "ok" || hit.Load() != 2 || lookups.Load() != 1 {
		t.Fatalf("5 MiB multipart status=%d body=%q hit=%d lookups=%d", status, body, hit.Load(), lookups.Load())
	}

	status, body = postRaw(t, addr, "application/octet-stream", bytes.Repeat([]byte("o"), 20<<20))
	if status != 200 || body != "ok" || hit.Load() != 3 || lookups.Load() != 2 {
		t.Fatalf("20 MiB octet-stream status=%d body=%q hit=%d lookups=%d", status, body, hit.Load(), lookups.Load())
	}

	status, body = postRaw(t, addr, "image/png", bytes.Repeat([]byte("p"), 20<<20))
	if status != 200 || body != "ok" || hit.Load() != 4 || lookups.Load() != 3 {
		t.Fatalf("20 MiB png status=%d body=%q hit=%d lookups=%d", status, body, hit.Load(), lookups.Load())
	}

	status, body = postRawPath(t, addr, "/upload", "", bytes.Repeat([]byte("m"), 3<<20))
	if status != 200 || body != "ok" || hit.Load() != 5 || lookups.Load() != 4 {
		t.Fatalf("3 MiB missing content-type status=%d body=%q hit=%d lookups=%d", status, body, hit.Load(), lookups.Load())
	}

	// Oversized Content-Length is rejected before the body is read.
	before := hit.Load()
	status = postDeclared(t, addr, "application/json", 3<<20)
	if status != 413 || hit.Load() != before || lookups.Load() != 5 {
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

func TestHTTPServerUpgradeStaysClosed(t *testing.T) {
	app := NewApplication(t.TempDir())
	srv, err := app.httpServer(ListenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if srv.AllowUpgrade {
		t.Fatal("AllowUpgrade must stay off by default")
	}
	called := false
	srv.Handler = func(ctx *rawhttp.Ctx) {
		called = true
		ctx.SetStatusCode(200)
		ctx.SetBodyString("nope")
	}
	ln := serveTestServer(t, srv)
	raw := wsHandshake("/ws", wsClientKey(), "")
	status, _ := exchangeRaw(t, ln.Addr().String(), []byte(raw))
	if status != 400 || called {
		t.Fatalf("status=%d called=%v, want 400 before the handler", status, called)
	}
}

func TestHTTPServerAllowUpgradeSources(t *testing.T) {
	t.Run("listen option", func(t *testing.T) {
		on := true
		app := NewApplication(t.TempDir())
		srv, err := app.httpServer(ListenOptions{AllowUpgrade: &on})
		if err != nil {
			t.Fatal(err)
		}
		if !srv.AllowUpgrade {
			t.Fatal("ListenOptions.AllowUpgrade must force admission on")
		}
	})
	t.Run("env", func(t *testing.T) {
		t.Setenv("HTTP_ALLOW_UPGRADE", "true")
		app := NewApplication(t.TempDir())
		srv, err := app.httpServer(ListenOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if !srv.AllowUpgrade {
			t.Fatal("HTTP_ALLOW_UPGRADE=true must admit upgrades")
		}
	})
	t.Run("env false wins over linked package", func(t *testing.T) {
		t.Setenv("HTTP_ALLOW_UPGRADE", "false")
		addons.Register(addons.Meta{Name: "websocket", Description: "test"})
		t.Cleanup(addons.ClearRegistry)
		app := NewApplication(t.TempDir())
		srv, err := app.httpServer(ListenOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if srv.AllowUpgrade {
			t.Fatal("HTTP_ALLOW_UPGRADE=false must keep admission off")
		}
	})
	t.Run("enabled addon", func(t *testing.T) {
		app := NewApplication(t.TempDir())
		app.SetEnabledAddons([]string{"websocket"})
		srv, err := app.httpServer(ListenOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if !srv.AllowUpgrade {
			t.Fatal("EnabledAddons websocket must admit upgrades")
		}
	})
	t.Run("env false overrides listen option", func(t *testing.T) {
		t.Setenv("HTTP_ALLOW_UPGRADE", "false")
		on := true
		app := NewApplication(t.TempDir())
		srv, err := app.httpServer(ListenOptions{AllowUpgrade: &on})
		if err != nil {
			t.Fatal(err)
		}
		if srv.AllowUpgrade {
			t.Fatal("HTTP_ALLOW_UPGRADE=false must override ListenOptions")
		}
	})
	t.Run("option forces off", func(t *testing.T) {
		off := false
		app := NewApplication(t.TempDir())
		app.SetEnabledAddons([]string{"websocket"})
		srv, err := app.httpServer(ListenOptions{AllowUpgrade: &off})
		if err != nil {
			t.Fatal(err)
		}
		if srv.AllowUpgrade {
			t.Fatal("explicit AllowUpgrade false must override the addon")
		}
	})
}

func TestHTTPServerTimeoutsFromEnv(t *testing.T) {
	t.Setenv("HTTP_READ_TIMEOUT", "-1s")
	t.Setenv("HTTP_WRITE_TIMEOUT", "0")
	t.Setenv("HTTP_IDLE_TIMEOUT", "2m")
	t.Setenv("HTTP_READ_HEADER_TIMEOUT", "5s")
	app := NewApplication(t.TempDir())
	srv, err := app.httpServer(ListenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if srv.ReadTimeout != -1 || srv.WriteTimeout != -1 || srv.IdleTimeout != 2*time.Minute || srv.ReadHeaderTimeout != 5*time.Second {
		t.Fatalf("timeouts read=%v write=%v idle=%v header=%v", srv.ReadTimeout, srv.WriteTimeout, srv.IdleTimeout, srv.ReadHeaderTimeout)
	}

	t.Setenv("HTTP_READ_TIMEOUT", "nope")
	if _, err := app.httpServer(ListenOptions{}); err == nil || !strings.Contains(err.Error(), "HTTP_READ_TIMEOUT") {
		t.Fatalf("invalid duration err=%v", err)
	}
}

func TestUpgradeH2CRejectedWhileOpen(t *testing.T) {
	on := true
	app := NewApplication(t.TempDir())
	srv, err := app.httpServer(ListenOptions{AllowUpgrade: &on})
	if err != nil {
		t.Fatal(err)
	}
	called := false
	srv.Handler = func(ctx *rawhttp.Ctx) {
		called = true
		ctx.SetStatusCode(200)
	}
	ln := serveTestServer(t, srv)
	raw := "GET / HTTP/1.1\r\nHost: localhost\r\nConnection: Upgrade\r\nUpgrade: h2c\r\n\r\n"
	status, _ := exchangeRaw(t, ln.Addr().String(), []byte(raw))
	if status != 400 || called {
		t.Fatalf("h2c status=%d called=%v, want 400 before the handler", status, called)
	}
}

func TestUpgradeOnPlainRouteCloses(t *testing.T) {
	on := true
	app := NewApplication(t.TempDir())
	srv, err := app.httpServer(ListenOptions{AllowUpgrade: &on})
	if err != nil {
		t.Fatal(err)
	}
	srv.Handler = func(ctx *rawhttp.Ctx) {
		ctx.SetStatusCode(200)
		ctx.SetBodyString("plain")
	}
	ln := serveTestServer(t, srv)
	c, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(3 * time.Second))
	if _, err := io.WriteString(c, wsHandshake("/", wsClientKey(), "")); err != nil {
		t.Fatal(err)
	}
	buf, err := io.ReadAll(c)
	if err != nil {
		t.Fatal(err)
	}
	text := string(buf)
	if !strings.Contains(text, "200") || !strings.Contains(text, "plain") {
		t.Fatalf("response %q", text)
	}
}

func TestWriteTimeoutZeroAllowsLongWrite(t *testing.T) {
	t.Setenv("HTTP_WRITE_TIMEOUT", "0")
	t.Setenv("HTTP_READ_TIMEOUT", "-1")
	t.Setenv("HTTP_IDLE_TIMEOUT", "-1")
	t.Setenv("HTTP_READ_HEADER_TIMEOUT", "10s")
	app := NewApplication(t.TempDir())
	srv, err := app.httpServer(ListenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if srv.WriteTimeout >= 0 {
		t.Fatalf("write timeout %v, want unlimited", srv.WriteTimeout)
	}
	srv.Handler = func(ctx *rawhttp.Ctx) {
		time.Sleep(70 * time.Second)
		ctx.SetStatusCode(200)
		ctx.SetBodyString("late-ok")
	}
	ln := serveTestServer(t, srv)
	c, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(90 * time.Second))
	if _, err := io.WriteString(c, "GET / HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"); err != nil {
		t.Fatal(err)
	}
	buf, err := io.ReadAll(c)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(buf), "late-ok") {
		t.Fatalf("70s write was cut: %q", buf)
	}
}

func TestHTTPServerUpgradeEchoOriginAndShutdown(t *testing.T) {
	dir := t.TempDir()
	app := NewApplication(dir)
	t.Cleanup(func() {
		if app.logger != nil {
			_ = app.logger.Close()
		}
	})
	app.router.Get("/ws", func(req *http.Request) *http.Response {
		origin := strings.TrimSpace(req.Header("Origin"))
		if origin != "" {
			if !strings.EqualFold(originHost(origin), req.Host()) {
				return http.Abort(403, "origin not allowed")
			}
		}
		return http.Hijack(func(conn net.Conn, leftover []byte) error {
			key := req.Header("Sec-WebSocket-Key")
			_, err := io.WriteString(conn, "HTTP/1.1 101 Switching Protocols\r\n"+
				"Upgrade: websocket\r\n"+
				"Connection: Upgrade\r\n"+
				"Sec-WebSocket-Accept: "+wsAccept(key)+"\r\n\r\n")
			if err != nil {
				return err
			}
			buf := append([]byte(nil), leftover...)
			tmp := make([]byte, 64)
			for len(buf) < 6+4 { // 2 header + 4 mask + "ping"
				_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
				n, rerr := conn.Read(tmp)
				if n > 0 {
					buf = append(buf, tmp[:n]...)
				}
				if rerr != nil {
					return rerr
				}
			}
			payload := make([]byte, 4)
			mask := buf[2:6]
			for i := range payload {
				payload[i] = buf[6+i] ^ mask[i%4]
			}
			echo := []byte{0x81, byte(len(payload))}
			echo = append(echo, payload...)
			_, err = conn.Write(echo)
			return err
		})
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	on := true
	srv, err := app.httpServer(ListenOptions{AllowUpgrade: &on})
	if err != nil {
		t.Fatal(err)
	}
	ln := serveTestServer(t, srv)
	addr := ln.Addr().String()
	key := wsClientKey()

	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if _, err := io.WriteString(c, wsHandshake("/ws", key, "")); err != nil {
		t.Fatal(err)
	}
	_ = c.SetReadDeadline(time.Now().Add(3 * time.Second))
	head, err := readHeaders(c)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(head, "101") || !strings.Contains(head, "Sec-WebSocket-Accept: "+wsAccept(key)) {
		t.Fatalf("handshake: %q", head)
	}
	frame := maskFrame([]byte("ping"), [4]byte{1, 2, 3, 4})
	if _, err := c.Write(frame); err != nil {
		t.Fatal(err)
	}
	echo := make([]byte, 16)
	n, err := io.ReadFull(c, echo[:6])
	if err != nil || n < 6 || echo[0] != 0x81 || string(echo[2:6]) != "ping" {
		t.Fatalf("echo n=%d err=%v bytes=%x", n, err, echo[:n])
	}
	_ = c.Close()

	bad, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer bad.Close()
	if _, err := io.WriteString(bad, wsHandshake("/ws", key, "https://evil.example")); err != nil {
		t.Fatal(err)
	}
	_ = bad.SetReadDeadline(time.Now().Add(3 * time.Second))
	badHead, err := readHeaders(bad)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(badHead, "403") {
		t.Fatalf("bad origin: %q", badHead)
	}

	block, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer block.Close()
	if _, err := io.WriteString(block, wsHandshake("/ws", key, "")); err != nil {
		t.Fatal(err)
	}
	_ = block.SetReadDeadline(time.Now().Add(3 * time.Second))
	if h, err := readHeaders(block); err != nil || !strings.Contains(h, "101") {
		t.Fatalf("block handshake %q err=%v", h, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()
	if err := srv.Shutdown(ctx); err == nil {
		t.Fatal("Shutdown should return when the hijacked handler is still blocked")
	}
	_ = block.SetReadDeadline(time.Now().Add(2 * time.Second))
	tmp := make([]byte, 8)
	if _, err := block.Read(tmp); err == nil {
		t.Fatal("hijacked connection stayed open after Shutdown")
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

func wsClientKey() string {
	raw := make([]byte, 16)
	for i := range raw {
		raw[i] = byte(i + 1)
	}
	return base64.StdEncoding.EncodeToString(raw)
}

func wsAccept(key string) string {
	sum := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(sum[:])
}

func wsHandshake(path, key, origin string) string {
	b := "GET " + path + " HTTP/1.1\r\n" +
		"Host: localhost\r\n" +
		"Connection: keep-alive, Upgrade\r\n" +
		"Upgrade: websocket\r\n" +
		"Sec-WebSocket-Key: " + key + "\r\n" +
		"Sec-WebSocket-Version: 13\r\n"
	if origin != "" {
		b += "Origin: " + origin + "\r\n"
	}
	return b + "\r\n"
}

func maskFrame(payload []byte, mask [4]byte) []byte {
	out := make([]byte, 2+4+len(payload))
	out[0] = 0x81
	out[1] = 0x80 | byte(len(payload))
	copy(out[2:6], mask[:])
	for i, b := range payload {
		out[6+i] = b ^ mask[i%4]
	}
	return out
}

func originHost(origin string) string {
	origin = strings.TrimPrefix(origin, "https://")
	origin = strings.TrimPrefix(origin, "http://")
	if i := strings.IndexByte(origin, '/'); i >= 0 {
		origin = origin[:i]
	}
	return origin
}

func readHeaders(c net.Conn) (string, error) {
	var buf []byte
	tmp := make([]byte, 256)
	for !bytes.Contains(buf, []byte("\r\n\r\n")) {
		n, err := c.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			return string(buf), err
		}
		if len(buf) > 1<<20 {
			return string(buf), io.ErrShortBuffer
		}
	}
	return string(buf), nil
}
