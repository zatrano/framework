package http

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"net"
	stdhttp "net/http"
	"strings"
	"testing"
	"time"

	"github.com/zatrano/rawhttp"
)

func TestInlineHeaderListMatchesMapPath(t *testing.T) {
	list := Text("ok")
	list.SetHeader("X-Frame-Options", "SAMEORIGIN")
	list.AddHeader("Vary", "Origin")
	list.AddHeader("X-Request-ID", "abc")
	if list.HeaderMapBuilt() {
		t.Fatal("list path built a map")
	}
	mapped := Text("ok")
	mapped.Headers()
	mapped.SetHeader("X-Frame-Options", "SAMEORIGIN")
	mapped.AddHeader("Vary", "Origin")
	mapped.AddHeader("X-Request-ID", "abc")
	if !mapped.HeaderMapBuilt() {
		t.Fatal("map path did not build a map")
	}

	left, leftBody := commitResponse(t, list)
	right, rightBody := commitResponse(t, mapped)
	if leftBody != "ok" || rightBody != "ok" {
		t.Fatalf("bodies %q %q", leftBody, rightBody)
	}
	if left.StatusCode != 200 || right.StatusCode != 200 {
		t.Fatalf("status %d %d", left.StatusCode, right.StatusCode)
	}
	for _, key := range []string{"X-Frame-Options", "Vary", "X-Request-ID", "Content-Type"} {
		if left.Header.Get(key) != right.Header.Get(key) {
			t.Fatalf("%s list=%q map=%q", key, left.Header.Get(key), right.Header.Get(key))
		}
	}
	if left.Header.Get("X-Frame-Options") != "SAMEORIGIN" || left.Header.Get("X-Request-ID") != "abc" {
		t.Fatalf("headers=%v", left.Header)
	}
}

func TestInlineHeaderOverflowMatchesMap(t *testing.T) {
	list := Text("ok")
	mapped := Text("ok")
	mapped.Headers()
	for i := 0; i < 12; i++ {
		name := fmt.Sprintf("X-N-%d", i)
		value := fmt.Sprintf("v%d", i)
		list.SetHeader(name, value)
		mapped.SetHeader(name, value)
	}
	if list.HeaderMapBuilt() {
		t.Fatal("overflow built a map")
	}
	left, leftBody := commitResponse(t, list)
	right, _ := commitResponse(t, mapped)
	if leftBody != "ok" || left.StatusCode != 200 {
		t.Fatalf("status=%d body=%q", left.StatusCode, leftBody)
	}
	for i := 0; i < 12; i++ {
		name := fmt.Sprintf("X-N-%d", i)
		if left.Header.Get(name) != right.Header.Get(name) {
			t.Fatalf("%s list=%q map=%q", name, left.Header.Get(name), right.Header.Get(name))
		}
	}
}

func TestHeaderInjectionDoesNotSplitResponse(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 40; i++ {
		name := randHeaderBytes(rng, 12)
		value := randHeaderBytes(rng, 24)
		resp := Text("ok")
		resp.SetHeader("X-Ok", "yes")
		resp.AddHeader(name, value)
		parsed, body := commitResponse(t, resp)
		if body != "ok" {
			t.Fatalf("body=%q name=%q value=%q", body, name, value)
		}
		if parsed.StatusCode != 200 {
			t.Fatalf("status=%d", parsed.StatusCode)
		}
		if parsed.Header.Get("X-Ok") != "yes" {
			t.Fatalf("missing safe header, got %v (name=%q)", parsed.Header, name)
		}
		if len(parsed.Header) > 8 {
			t.Fatalf("too many headers: %v", parsed.Header)
		}
		for key, vals := range parsed.Header {
			for _, v := range vals {
				if strings.ContainsAny(key, "\r\n") || strings.ContainsAny(v, "\r\n") {
					t.Fatalf("split header %q=%q", key, v)
				}
			}
		}
	}
}

func randHeaderBytes(rng *rand.Rand, n int) string {
	buf := make([]byte, n)
	for i := range buf {
		buf[i] = byte(rng.Intn(256))
	}
	return string(buf)
}

func commitResponse(t *testing.T, resp *Response) (*stdhttp.Response, string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &rawhttp.Server{
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
		IdleTimeout:  time.Second,
		Handler: func(ctx *rawhttp.Ctx) {
			_ = resp.Commit(ctx)
		},
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	conn, err := net.DialTimeout("tcp", ln.Addr().String(), time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := io.WriteString(conn, "GET / HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"); err != nil {
		t.Fatal(err)
	}
	parsed, err := stdhttp.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer parsed.Body.Close()
	body, err := io.ReadAll(parsed.Body)
	if err != nil {
		t.Fatal(err)
	}
	return parsed, string(body)
}
