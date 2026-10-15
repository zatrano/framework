package main

import (
	"bytes"
	"net"
	"os"
	"regexp"
	"sort"
	"testing"
	"time"
)

func TestFiberTunedResponseBytesMatchZatrano(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("APP_DEBUG", "false")
	t.Setenv("LOG_LEVEL", "error")
	t.Setenv("CORS_ENABLED", "true")
	t.Setenv("APP_KEY", "zatrano-ci-test-key-not-prod!!!!")
	t.Setenv("HTTP_POOL_REQUESTS", "")

	zAddr := serveOnFreePort(t, serveZatrano)
	fAddr := serveOnFreePort(t, serveFiberTuned)

	fixed := "abc-123"
	cases := []struct {
		name   string
		req    string
		maskID bool
	}{
		{
			name: "plaintext-fixed",
			req:  "GET /plaintext HTTP/1.1\r\nHost: bench\r\nX-Request-ID: " + fixed + "\r\nConnection: keep-alive\r\n\r\n",
		},
		{
			name:   "plaintext-invalid",
			req:    "GET /plaintext HTTP/1.1\r\nHost: bench\r\nX-Request-ID: bad id\r\nConnection: keep-alive\r\n\r\n",
			maskID: true,
		},
		{
			name:   "plaintext-generated",
			req:    "GET /plaintext HTTP/1.1\r\nHost: bench\r\nConnection: keep-alive\r\n\r\n",
			maskID: true,
		},
		{
			name: "correlation",
			req:  "GET /plaintext HTTP/1.1\r\nHost: bench\r\nX-Correlation-ID: corr-1\r\nConnection: keep-alive\r\n\r\n",
		},
		{
			name: "traceparent",
			req:  "GET /plaintext HTTP/1.1\r\nHost: bench\r\nTraceparent: 00-0123456789abcdef0123456789abcdef-0123456789abcdef-01\r\nConnection: keep-alive\r\n\r\n",
		},
		{
			name: "json",
			req:  "GET /json HTTP/1.1\r\nHost: bench\r\nX-Request-ID: " + fixed + "\r\nConnection: keep-alive\r\n\r\n",
		},
		{
			name: "users",
			req:  "GET /users/42 HTTP/1.1\r\nHost: bench\r\nX-Request-ID: " + fixed + "\r\nConnection: keep-alive\r\n\r\n",
		},
		{
			name: "echo",
			req:  "POST /echo HTTP/1.1\r\nHost: bench\r\nX-Request-ID: " + fixed + "\r\nContent-Type: application/json\r\nContent-Length: 27\r\nConnection: keep-alive\r\n\r\n{\"message\":\"Hello, World!\"}",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			z := mustRoundTrip(t, zAddr, tc.req)
			f := mustRoundTrip(t, fAddr, tc.req)
			if tc.maskID {
				if !hex32.Match(headerValue(z, "X-Request-Id")) || !hex32.Match(headerValue(f, "X-Request-Id")) {
					t.Fatalf("generated id z=%q f=%q", headerValue(z, "X-Request-Id"), headerValue(f, "X-Request-Id"))
				}
				z = maskHeader(z, "X-Request-Id", "ID")
				f = maskHeader(f, "X-Request-Id", "ID")
			}
			if !bytes.Equal(canonicalWire(z), canonicalWire(f)) {
				t.Fatalf("canonical bytes differ\nZATRANO:\n%s\nFiber tuned:\n%s", z, f)
			}
		})
	}

	a := headerValue(mustRoundTrip(t, fAddr, "GET /plaintext HTTP/1.1\r\nHost: bench\r\n\r\n"), "X-Request-Id")
	b := headerValue(mustRoundTrip(t, fAddr, "GET /plaintext HTTP/1.1\r\nHost: bench\r\n\r\n"), "X-Request-Id")
	if bytes.Equal(a, b) {
		t.Fatalf("two generated ids matched: %s", a)
	}
}

var hex32 = regexp.MustCompile(`^[0-9a-f]{32}$`)

func serveOnFreePort(t *testing.T, serve func(string) error) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()
	errCh := make(chan error, 1)
	go func() { errCh <- serve(addr) }()
	if err := waitReady(addr, 20*time.Second); err != nil {
		select {
		case serveErr := <-errCh:
			t.Fatalf("serve %s: %v (ready: %v)", addr, serveErr, err)
		default:
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		select {
		case err := <-errCh:
			if err != nil && !os.IsTimeout(err) {
				t.Logf("serve %s exit: %v", addr, err)
			}
		default:
		}
	})
	return addr
}

func mustRoundTrip(t *testing.T, addr, req string) []byte {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err := conn.Write([]byte(req)); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 8192)
	n := 0
	for {
		if n == len(buf) {
			t.Fatal("short buffer")
		}
		r, err := conn.Read(buf[n:])
		n += r
		if err != nil {
			t.Fatalf("read: %v\n%s", err, buf[:n])
		}
		if i := bytes.Index(buf[:n], headerSplit); i >= 0 {
			body := n - (i + 4)
			need := contentLen(buf[:i])
			for body < need {
				if n == len(buf) {
					t.Fatal("short buffer")
				}
				r, err = conn.Read(buf[n:])
				n += r
				body += r
				if err != nil {
					t.Fatalf("body: %v\n%s", err, buf[:n])
				}
			}
			return append([]byte(nil), buf[:i+4+need]...)
		}
	}
}

func headerValue(raw []byte, name string) []byte {
	head, _, _ := bytes.Cut(raw, headerSplit)
	prefix := []byte(name + ": ")
	for _, line := range bytes.Split(head, []byte("\r\n"))[1:] {
		if bytes.HasPrefix(line, prefix) {
			return line[len(prefix):]
		}
	}
	return nil
}

func maskHeader(raw []byte, name, value string) []byte {
	out := append([]byte(nil), raw...)
	head, _, _ := bytes.Cut(out, headerSplit)
	prefix := []byte(name + ": ")
	for _, line := range bytes.Split(head, []byte("\r\n")) {
		if bytes.HasPrefix(line, prefix) {
			old := append([]byte(nil), line...)
			return bytes.Replace(out, old, []byte(name+": "+value), 1)
		}
	}
	return out
}

func dateMasked(raw []byte) []byte {
	return maskHeader(raw, "Date", "DATE")
}

// canonicalWire sorts header lines. fasthttp writes Date, Content-Type, and
// Content-Length before application headers, so the raw frame order differs
// from ZATRANO. The compared bytes are the status line, every header line,
// and the body.
func canonicalWire(raw []byte) []byte {
	raw = dateMasked(raw)
	head, body, _ := bytes.Cut(raw, headerSplit)
	lines := bytes.Split(head, []byte("\r\n"))
	status := lines[0]
	rest := append([][]byte(nil), lines[1:]...)
	sort.Slice(rest, func(i, j int) bool { return bytes.Compare(rest[i], rest[j]) < 0 })
	var buf bytes.Buffer
	buf.Write(status)
	buf.WriteString("\r\n")
	for _, line := range rest {
		if len(line) == 0 {
			continue
		}
		buf.Write(line)
		buf.WriteString("\r\n")
	}
	buf.WriteString("\r\n")
	buf.Write(body)
	return buf.Bytes()
}
