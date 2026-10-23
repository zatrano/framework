// Package safety holds the canary stress gate for request isolation.
//
// The gate drives the full stack (rawhttp server, kernel, router, middleware,
// bridge) over loopback keep-alive connections with pipelined requests. Every
// request carries a unique id in its path, query, header, cookie and body.
// The handler echoes what it read, a sample of *Request values is retained and
// read again after later requests reused the connection buffers, and every
// tenth request is served from one shared cached *Response that an outer
// middleware decorates per request. Any value that belongs to another
// request is a mismatch; the gate passes only with zero mismatches.
//
// Default `go test` runs a short smoke. The release gate:
//
//	ZATRANO_CANARY=1 go test -count=1 -timeout 0 -run TestCanary ./tests/safety
//	ZATRANO_CANARY=1 go test -race -count=1 -timeout 0 -run TestCanary ./tests/safety
//
// ZATRANO_CANARY_REQUESTS overrides the request count (default 10,000,000).
package safety

import (
	"bufio"
	"fmt"
	"io"
	"net"
	stdhttp "net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zatrano/framework/v3/core/kernel"
	zhttp "github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/framework/v3/core/kernel/routing"
	"github.com/zatrano/rawhttp"
)

const (
	defaultGateRequests  = 10_000_000
	defaultSmokeRequests = 20_000
	connections          = 16
	pipelineDepth        = 32
	retainEvery          = 997
	cachedEvery          = 10
)

func canaryRequests(t *testing.T) int {
	if os.Getenv("ZATRANO_CANARY") != "1" {
		if testing.Short() {
			return 2_000
		}
		return defaultSmokeRequests
	}
	if s := os.Getenv("ZATRANO_CANARY_REQUESTS"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n <= 0 {
			t.Fatalf("ZATRANO_CANARY_REQUESTS=%q", s)
		}
		return n
	}
	return defaultGateRequests
}

type kept struct {
	req *zhttp.Request
	id  string
}

type canary struct {
	mismatches atomic.Int64
	firstErr   atomic.Value
	keptMu     sync.Mutex
	kept       []kept
	cached     *zhttp.Response
	cachedBody string
}

func (c *canary) fail(format string, a ...any) {
	if c.mismatches.Add(1) == 1 {
		c.firstErr.Store(fmt.Sprintf(format, a...))
	}
}

// echo is the handler body: every field the request returned, in order.
func echo(req *zhttp.Request) string {
	body, err := req.Body()
	if err != nil {
		return "body-error"
	}
	return req.Route("id") + "|" + req.Path() + "|" + req.Query("q") + "|" +
		req.Header("X-Id") + "|" + req.Cookie("sid") + "|" + string(body)
}

func want(id string) string {
	return id + "|/echo/" + id + "|q" + id + "|h" + id + "|s" + id + "|b" + id
}

func (c *canary) app(t *testing.T) *kernel.Application {
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() {
		if cl, ok := app.Logger().(interface{ Close() error }); ok {
			_ = cl.Close()
		}
	})
	c.cached = zhttp.JSON(map[string]string{"v": "shared"})
	c.cachedBody = string(c.cached.Content())
	app.Router().Use(func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *zhttp.Request) *zhttp.Response {
			return next(req).Header("X-Echo-Id", req.Header("X-Id")).Cookie("ctok", "c"+req.Header("X-Id"))
		}
	})
	app.Router().Post("/echo/{id}", func(req *zhttp.Request) *zhttp.Response {
		id := req.Route("id")
		if n, _ := strconv.Atoi(id); n%retainEvery == 0 {
			c.keptMu.Lock()
			c.kept = append(c.kept, kept{req: req, id: id})
			c.keptMu.Unlock()
		}
		return zhttp.Text(echo(req))
	})
	app.Router().Get("/cached", func(*zhttp.Request) *zhttp.Response { return c.cached })
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	return app
}

func appendRequest(b []byte, n int) []byte {
	id := strconv.Itoa(n)
	if n%cachedEvery == cachedEvery-1 {
		b = append(b, "GET /cached HTTP/1.1\r\nHost: canary\r\nX-Id: h"...)
		b = append(b, id...)
		return append(b, "\r\n\r\n"...)
	}
	body := "b" + id
	b = append(b, "POST /echo/"...)
	b = append(b, id...)
	b = append(b, "?q=q"...)
	b = append(b, id...)
	b = append(b, " HTTP/1.1\r\nHost: canary\r\nX-Id: h"...)
	b = append(b, id...)
	b = append(b, "\r\nCookie: sid=s"...)
	b = append(b, id...)
	b = append(b, "\r\nContent-Type: text/plain\r\nContent-Length: "...)
	b = strconv.AppendInt(b, int64(len(body)), 10)
	b = append(b, "\r\n\r\n"...)
	return append(b, body...)
}

func (c *canary) check(n int, resp *stdhttp.Response, body []byte) {
	id := strconv.Itoa(n)
	if resp.StatusCode != 200 {
		c.fail("request %d: status %d", n, resp.StatusCode)
		return
	}
	if got := resp.Header.Values("X-Echo-Id"); len(got) != 1 || got[0] != "h"+id {
		c.fail("request %d: X-Echo-Id %v", n, got)
	}
	if got := resp.Header.Values("Set-Cookie"); len(got) != 1 || !strings.HasPrefix(got[0], "ctok=ch"+id+";") {
		c.fail("request %d: Set-Cookie %v", n, got)
	}
	if n%cachedEvery == cachedEvery-1 {
		if string(body) != c.cachedBody {
			c.fail("request %d: cached body %q", n, body)
		}
		return
	}
	if string(body) != want(id) {
		c.fail("request %d: echo %q, want %q", n, body, want(id))
	}
}

func (c *canary) checkKept() {
	c.keptMu.Lock()
	defer c.keptMu.Unlock()
	for _, k := range c.kept {
		if got := echo(k.req); got != want(k.id) {
			c.fail("retained request %s reads %q", k.id, got)
		}
		if k.req.Ctx() != nil {
			c.fail("retained request %s still reaches the connection context", k.id)
		}
	}
}

func TestCanaryRequestIsolation(t *testing.T) {
	total := canaryRequests(t)
	c := &canary{}
	app := c.app(t)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &rawhttp.Server{Handler: func(ctx *rawhttp.Ctx) { app.Handle(ctx) }}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })
	addr := ln.Addr().String()

	stop := make(chan struct{})
	readerDone := make(chan struct{})
	// Reads retained requests while later requests reuse the buffers; under
	// -race any shared memory is reported.
	go func() {
		defer close(readerDone)
		for {
			select {
			case <-stop:
				return
			case <-time.After(time.Millisecond):
			}
			c.keptMu.Lock()
			recent := c.kept
			if len(recent) > 64 {
				recent = recent[len(recent)-64:]
			}
			snapshot := append([]kept(nil), recent...)
			c.keptMu.Unlock()
			for _, k := range snapshot {
				if got := echo(k.req); got != want(k.id) {
					c.fail("retained request %s reads %q during the run", k.id, got)
				}
			}
		}
	}()

	var next atomic.Int64
	var served atomic.Int64
	var wg sync.WaitGroup
	start := time.Now()
	for w := 0; w < connections; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				c.fail("dial: %v", err)
				return
			}
			defer conn.Close()
			br := bufio.NewReaderSize(conn, 64<<10)
			buf := make([]byte, 0, 64<<10)
			ids := make([]int, 0, pipelineDepth)
			for {
				ids = ids[:0]
				buf = buf[:0]
				for len(ids) < pipelineDepth {
					n := int(next.Add(1)) - 1
					if n >= total {
						break
					}
					ids = append(ids, n)
					buf = appendRequest(buf, n)
				}
				if len(ids) == 0 {
					return
				}
				_ = conn.SetDeadline(time.Now().Add(2 * time.Minute))
				if _, err := conn.Write(buf); err != nil {
					c.fail("write: %v", err)
					return
				}
				for _, n := range ids {
					resp, err := stdhttp.ReadResponse(br, nil)
					if err != nil {
						c.fail("request %d: read: %v", n, err)
						return
					}
					body, err := io.ReadAll(resp.Body)
					_ = resp.Body.Close()
					if err != nil {
						c.fail("request %d: body: %v", n, err)
						return
					}
					c.check(n, resp, body)
					served.Add(1)
				}
			}
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)
	close(stop)
	<-readerDone
	c.checkKept()

	if string(c.cached.Content()) != c.cachedBody || c.cached.HasHeader("X-Echo-Id") || len(c.cached.Cookies()) != 0 {
		c.fail("shared cached response was mutated: headers=%v cookies=%v", c.cached.Headers(), c.cached.Cookies())
	}

	t.Logf("CANARY requests=%d served=%d mismatches=%d retained=%d connections=%d pipeline=%d race=%v elapsed=%s rate=%.0f req/s go=%s",
		total, served.Load(), c.mismatches.Load(), len(c.kept), connections, pipelineDepth, raceEnabled,
		elapsed.Round(time.Millisecond), float64(served.Load())/elapsed.Seconds(), runtime.Version())
	if m := c.mismatches.Load(); m != 0 {
		t.Fatalf("%d mismatches; first: %v", m, c.firstErr.Load())
	}
	if served.Load() != int64(total) {
		t.Fatalf("served %d of %d", served.Load(), total)
	}
}
