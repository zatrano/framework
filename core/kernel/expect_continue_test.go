package kernel

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"testing"
	"time"

	"github.com/zatrano/framework/v3/core/kernel/http"
)

func TestExpectContinueUnderLimit(t *testing.T) {
	app := NewApplication(t.TempDir())
	app.router.Post("/upload", func(*http.Request) *http.Response {
		return http.Text("ok")
	})
	srv := mustServer(t, app)
	ln := serveTestServer(t, srv)
	c, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err := io.WriteString(c, "POST /upload HTTP/1.1\r\nHost: localhost\r\nContent-Type: application/json\r\nContent-Length: 2\r\nExpect: 100-continue\r\n\r\n"); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 256)
	n, err := c.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf[:n], []byte("100 Continue")) {
		t.Fatalf("missing 100 Continue: %q", buf[:n])
	}
	if _, err := io.WriteString(c, "{}"); err != nil {
		t.Fatal(err)
	}
	rest := make([]byte, 512)
	n, err = io.ReadAtLeast(c, rest, 12)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(rest[:n], []byte("200")) {
		t.Fatalf("final %q", rest[:n])
	}
	if got := waitReserved(app, 0); got != 0 {
		t.Fatalf("reserved=%d", got)
	}
}

func TestExpectContinueOverLimitHasNo100(t *testing.T) {
	app := NewApplication(t.TempDir())
	app.router.Post("/upload", func(*http.Request) *http.Response {
		return http.Text("ok")
	})
	srv := mustServer(t, app)
	ln := serveTestServer(t, srv)
	raw := readExpect(t, ln.Addr().String(), "POST /upload HTTP/1.1\r\nHost: localhost\r\nContent-Type: application/json\r\nContent-Length: 3000000\r\nExpect: 100-continue\r\nConnection: close\r\n\r\n")
	if bytes.Contains(raw, []byte("100 Continue")) {
		t.Fatalf("wrote 100 Continue: %q", raw)
	}
	if !bytes.Contains(raw, []byte("413")) {
		t.Fatalf("status %q", raw)
	}
}

func TestExpectContinueBudgetHasNo100(t *testing.T) {
	// 40 MiB holds one 30 MiB reservation and rejects the next. The per-client
	// share, once it exists, must not be tighter than that reservation.
	t.Setenv("HTTP_MAX_INFLIGHT_BODY_BYTES", fmt.Sprint(int64(40<<20)))
	t.Setenv("HTTP_MAX_INFLIGHT_BODY_BYTES_PER_CLIENT", fmt.Sprint(int64(40<<20)))
	app := NewApplication(t.TempDir())
	app.router.Post("/upload", func(*http.Request) *http.Response {
		return http.Text("ok")
	})
	srv := mustServer(t, app)
	ln := serveTestServer(t, srv)
	addr := ln.Addr().String()

	const want = int64(30 << 20)
	hold, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer hold.Close()
	_ = hold.SetDeadline(time.Now().Add(15 * time.Second))
	if _, err := fmt.Fprintf(hold, "POST /upload HTTP/1.1\r\nHost: localhost\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n", want); err != nil {
		t.Fatal(err)
	}
	if _, err := hold.Write([]byte{0xab}); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for app.BodyReserved() < want && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if got := app.BodyReserved(); got < want {
		t.Fatalf("reserved=%d want >= %d", got, want)
	}
	raw := readExpect(t, addr, "POST /upload HTTP/1.1\r\nHost: localhost\r\nContent-Type: application/octet-stream\r\nContent-Length: 31457280\r\nExpect: 100-continue\r\nConnection: close\r\n\r\n")
	if bytes.Contains(raw, []byte("100 Continue")) {
		t.Fatalf("wrote 100 Continue: %q", raw)
	}
	if !bytes.Contains(raw, []byte("503")) {
		t.Fatalf("status %q", raw)
	}
	if !bytes.Contains(raw, []byte("Retry-After: 1")) {
		t.Fatalf("headers %q", raw)
	}
	_ = hold.Close()
	deadline = time.Now().Add(10 * time.Second)
	for app.BodyReserved() != 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if got := app.BodyReserved(); got != 0 {
		t.Fatalf("reserved=%d after the holding connection closed", got)
	}
}

func readExpect(t *testing.T, addr, head string) []byte {
	t.Helper()
	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err := io.WriteString(c, head); err != nil {
		t.Fatal(err)
	}
	raw, err := io.ReadAll(c)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) == 0 {
		t.Fatal("client read no response")
	}
	return raw
}
