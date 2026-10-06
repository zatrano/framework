package http

import (
	"bufio"
	"fmt"
	"io"
	"net"
	stdhttp "net/http"
	"sync"
	"testing"
	"time"

	"github.com/zatrano/rawhttp"
)

func TestPoolOffKeepsRequestAfterRelease(t *testing.T) {
	ConfigureRequestPool(false)
	t.Cleanup(func() { ConfigureRequestPool(false) })

	req := testRequest("GET", "/kept?q=1", []byte("body"))
	req.SetHeader("X-Trace", "abc")
	release := make(chan struct{})
	got := make(chan string, 1)
	panicked := make(chan any, 1)
	go func() {
		<-release
		defer func() {
			if p := recover(); p != nil {
				panicked <- p
			}
		}()
		value, _ := req.HeaderValue("X-Trace")
		got <- value
	}()
	ReleaseRequest(req)
	close(release)

	select {
	case p := <-panicked:
		if !requestFreedPoison {
			t.Fatalf("pool off panicked: %v", p)
		}
		if p == nil {
			t.Fatal("poison build did not panic")
		}
		if fmt.Sprint(p) != "request used after handler returned" {
			t.Fatalf("panic=%v", p)
		}
	case value := <-got:
		if requestFreedPoison {
			t.Fatalf("poison build returned %q", value)
		}
		if value != "abc" {
			t.Fatalf("value=%q", value)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out")
	}
}

func TestPooledRequestsDoNotLeakAcrossUsers(t *testing.T) {
	ConfigureRequestPool(true)
	t.Cleanup(func() { ConfigureRequestPool(false) })

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &rawhttp.Server{
		Handler: func(ctx *rawhttp.Ctx) {
			req := NewRequest(ctx)
			defer ReleaseRequest(req)
			value, _ := req.HeaderValue("X-Trace")
			ctx.SetStatusCode(200)
			ctx.SetBodyString(value)
		},
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	const n = 1000
	sem := make(chan struct{}, 128)
	var wg sync.WaitGroup
	errs := make(chan string, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()
			id := fmt.Sprintf("trace-%d", i)
			var conn net.Conn
			var err error
			for attempt := 0; attempt < 50; attempt++ {
				conn, err = net.Dial("tcp", ln.Addr().String())
				if err == nil {
					break
				}
				time.Sleep(2 * time.Millisecond)
			}
			if err != nil {
				errs <- err.Error()
				return
			}
			defer conn.Close()
			_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
			raw := "GET / HTTP/1.1\r\nHost: localhost\r\nX-Trace: " + id + "\r\nConnection: close\r\n\r\n"
			if _, err := io.WriteString(conn, raw); err != nil {
				errs <- err.Error()
				return
			}
			resp, err := stdhttp.ReadResponse(bufio.NewReader(conn), nil)
			if err != nil {
				errs <- err.Error()
				return
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				errs <- err.Error()
				return
			}
			if string(body) != id {
				errs <- fmt.Sprintf("want %s got %q", id, body)
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	var leaks []string
	for msg := range errs {
		leaks = append(leaks, msg)
	}
	if len(leaks) != 0 {
		t.Fatalf("cross-talk or errors: %d first=%s", len(leaks), leaks[0])
	}
}
