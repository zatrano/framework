package kernel

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/zatrano/framework/v3/core/kernel/log"
	"github.com/zatrano/rawhttp"
)

func TestMaxHeaderBytesDefault(t *testing.T) {
	t.Setenv("HTTP_MAX_HEADER_BYTES", "")
	app := NewApplication(t.TempDir())
	srv, err := app.httpServer(ListenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if srv.MaxHeaderBytes != DefaultMaxHeaderBytes {
		t.Fatalf("MaxHeaderBytes=%d want %d", srv.MaxHeaderBytes, DefaultMaxHeaderBytes)
	}
}

func TestMaxHeaderBytesInvalid(t *testing.T) {
	app := NewApplication(t.TempDir())
	for _, raw := range []string{"abc", "16kb", "1.5", "-1", "0"} {
		t.Setenv("HTTP_MAX_HEADER_BYTES", raw)
		if _, err := app.httpServer(ListenOptions{}); err == nil {
			t.Fatalf("HTTP_MAX_HEADER_BYTES=%q accepted", raw)
		}
	}
}

func TestMaxHeaderBytesWarnsAbove64KiB(t *testing.T) {
	t.Setenv("HTTP_MAX_HEADER_BYTES", "131072")
	app := NewApplication(t.TempDir())
	path := t.TempDir() + "/zatrano.log"
	logger, err := log.New("warning", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = logger.Close() })
	app.logger = logger
	srv, err := app.httpServer(ListenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if srv.MaxHeaderBytes != 131072 {
		t.Fatalf("MaxHeaderBytes=%d", srv.MaxHeaderBytes)
	}
	_ = logger.Close()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "per-connection buffer cost") || !strings.Contains(string(body), "v0.2.4") {
		t.Fatalf("warning log = %q", body)
	}
}

func TestHeaderBlockAround16KiB(t *testing.T) {
	t.Setenv("HTTP_MAX_HEADER_BYTES", "")
	app := NewApplication(t.TempDir())
	srv, err := app.httpServer(ListenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	srv.Handler = func(ctx *rawhttp.Ctx) {
		ctx.SetStatusCode(200)
		ctx.SetBodyString("ok")
	}
	ln := serveTestServer(t, srv)
	addr := ln.Addr().String()

	if status, body := headerSizedRequest(t, addr, 14<<10); status != 200 || body != "ok" {
		t.Fatalf("14 KiB headers status=%d body=%q", status, body)
	}
	if status, _ := headerSizedRequest(t, addr, 17<<10); status != 431 {
		t.Fatalf("17 KiB headers status=%d want 431", status)
	}
}

func headerSizedRequest(t *testing.T, addr string, headerBytes int) (int, string) {
	t.Helper()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	prefix := "GET / HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\nX-Pad: "
	suffix := "\r\n\r\n"
	pad := headerBytes - len(prefix) - len(suffix)
	if pad < 1 {
		t.Fatalf("header target %d smaller than the request line", headerBytes)
	}
	var b strings.Builder
	b.Grow(headerBytes)
	b.WriteString(prefix)
	b.WriteString(strings.Repeat("a", pad))
	b.WriteString(suffix)
	if b.Len() != headerBytes {
		t.Fatalf("header block %d want %d", b.Len(), headerBytes)
	}
	if _, err := io.WriteString(conn, b.String()); err != nil {
		t.Fatal(err)
	}
	br := bufio.NewReader(conn)
	line, err := br.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	var status int
	if _, err := fmt.Sscanf(line, "HTTP/1.1 %d", &status); err != nil {
		t.Fatalf("status line %q: %v", line, err)
	}
	if status != 200 {
		return status, ""
	}
	for {
		h, err := br.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if h == "\r\n" || h == "\n" {
			break
		}
	}
	body, err := io.ReadAll(br)
	if err != nil {
		t.Fatal(err)
	}
	return status, strings.TrimSpace(string(body))
}
