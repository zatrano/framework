package kernel_test

import (
	"bufio"
	"fmt"
	"io"
	"net"
	stdhttp "net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zatrano/framework/v3/core/contracts"
	"github.com/zatrano/framework/v3/core/kernel"
	zhttp "github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/framework/v3/core/kernel/routing"
	"github.com/zatrano/rawhttp"
)

// serveApp runs app.Handle on a loopback rawhttp server and returns its address.
func serveApp(t *testing.T, app *kernel.Application) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &rawhttp.Server{Handler: func(ctx *rawhttp.Ctx) { app.Handle(ctx) }}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })
	return ln.Addr().String()
}

type keepAliveClient struct {
	conn net.Conn
	r    *bufio.Reader
}

func dialKeepAlive(t *testing.T, addr string) *keepAliveClient {
	t.Helper()
	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	_ = c.SetDeadline(time.Now().Add(60 * time.Second))
	return &keepAliveClient{conn: c, r: bufio.NewReader(c)}
}

func (k *keepAliveClient) do(raw string) (*stdhttp.Response, string, error) {
	if _, err := io.WriteString(k.conn, raw); err != nil {
		return nil, "", err
	}
	resp, err := stdhttp.ReadResponse(k.r, nil)
	if err != nil {
		return nil, "", err
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	return resp, string(body), err
}

type retained struct {
	req                                   *zhttp.Request
	method, path, id, q, token, sid, body string
	headers                               map[string]string
}

func lifetimeApp(t *testing.T, keep func(retained)) *kernel.Application {
	t.Helper()
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	app.Router().Post("/echo/{id}", func(req *zhttp.Request) *zhttp.Response {
		keep(retained{
			req:     req,
			method:  req.Method(),
			path:    req.Path(),
			id:      req.Route("id"),
			q:       req.Query("q"),
			token:   req.Header("X-Token"),
			sid:     req.Cookie("sid"),
			body:    bodyOf(req),
			headers: req.HeadersMap(),
		})
		return zhttp.Text("ok")
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	return app
}

func echoRequest(i int, pad string) string {
	body := fmt.Sprintf("body-%d-%s", i, pad)
	return fmt.Sprintf("POST /echo/%d?q=q%d HTTP/1.1\r\nHost: localhost\r\nX-Token: tok-%d\r\nCookie: sid=s%d\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s",
		i, i, i, i, len(body), body)
}

func checkRetained(t *testing.T, i int, got retained) {
	t.Helper()
	want := retained{
		method: "POST",
		path:   "/echo/" + strconv.Itoa(i),
		id:     strconv.Itoa(i),
		q:      "q" + strconv.Itoa(i),
		token:  "tok-" + strconv.Itoa(i),
		sid:    "s" + strconv.Itoa(i),
	}
	if got.method != want.method || got.path != want.path || got.id != want.id || got.q != want.q || got.token != want.token || got.sid != want.sid {
		t.Fatalf("request %d: captured strings changed: %+v", i, got)
	}
	if !strings.HasPrefix(got.body, fmt.Sprintf("body-%d-", i)) {
		t.Fatalf("request %d: captured body changed: %q", i, got.body)
	}
	if got.headers["X-Token"] != want.token {
		t.Fatalf("request %d: captured header map changed: %v", i, got.headers)
	}
	r := got.req
	if r.Method() != want.method || r.Path() != want.path || r.Route("id") != want.id || r.Query("q") != want.q ||
		r.Header("X-Token") != want.token || r.Cookie("sid") != want.sid || bodyOf(r) != got.body {
		t.Fatalf("request %d: retained *Request reads another request: %s %s id=%s q=%s tok=%s sid=%s body=%q",
			i, r.Method(), r.Path(), r.Route("id"), r.Query("q"), r.Header("X-Token"), r.Cookie("sid"), bodyOf(r))
	}
	if r.Ctx() != nil {
		t.Fatalf("request %d: Ctx still reachable after the handler returned", i)
	}
}

// I1 and I2: a *Request and every value it returned stay those of their own
// request after rawhttp reuses the connection buffer for later requests.
func TestRetainedRequestKeepsItsOwnData(t *testing.T) {
	var mu sync.Mutex
	var kept []retained
	app := lifetimeApp(t, func(r retained) {
		mu.Lock()
		kept = append(kept, r)
		mu.Unlock()
	})
	addr := serveApp(t, app)
	client := dialKeepAlive(t, addr)

	stop := make(chan struct{})
	readerDone := make(chan struct{})
	// Reads retained requests while later requests are parsed into the same
	// buffer; the race detector reports any shared memory.
	go func() {
		defer close(readerDone)
		for {
			select {
			case <-stop:
				return
			default:
			}
			mu.Lock()
			snapshot := append([]retained(nil), kept...)
			mu.Unlock()
			for _, r := range snapshot {
				_ = r.req.Path() + r.req.Header("X-Token") + r.req.Cookie("sid") + bodyOf(r.req)
			}
		}
	}()

	const n = 300
	for i := 0; i < n; i++ {
		// Varying padding moves every field to a different buffer offset.
		resp, body, err := client.do(echoRequest(i, strings.Repeat("x", i%37)))
		if err != nil {
			t.Fatalf("request %d: %v", i, err)
		}
		if resp.StatusCode != 200 || body != "ok" {
			t.Fatalf("request %d: status=%d body=%q", i, resp.StatusCode, body)
		}
	}
	close(stop)
	<-readerDone
	if len(kept) != n {
		t.Fatalf("kept %d requests, want %d", len(kept), n)
	}
	for i, r := range kept {
		checkRetained(t, i, r)
	}
}

// A handler that hands its request to a goroutine sees the same data after
// the response is written.
func TestRequestUsedAfterResponseByGoroutine(t *testing.T) {
	results := make(chan error, 64)
	var wg sync.WaitGroup
	app := lifetimeApp(t, func(r retained) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(5 * time.Millisecond)
			if r.req.Path() != r.path || r.req.Header("X-Token") != r.token || bodyOf(r.req) != r.body {
				results <- fmt.Errorf("goroutine saw %s %s %q, handler saw %s %s %q",
					r.req.Path(), r.req.Header("X-Token"), bodyOf(r.req), r.path, r.token, r.body)
			}
		}()
	})
	addr := serveApp(t, app)
	client := dialKeepAlive(t, addr)
	for i := 0; i < 64; i++ {
		if _, _, err := client.do(echoRequest(i, strings.Repeat("y", 64-i))); err != nil {
			t.Fatal(err)
		}
	}
	wg.Wait()
	close(results)
	for err := range results {
		t.Fatal(err)
	}
}

// The engine never writes to a *Response the handler returned. One cached
// response served to many concurrent requests keeps its original headers,
// cookies and body, and no request sees another request's cookie.
func TestSharedCachedResponseIsNeverMutated(t *testing.T) {
	cached := zhttp.JSON(map[string]string{"v": "shared"})
	cached.Header("X-Cache", "hit")
	wantBody := string(cached.Content())

	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	app.SetHTTPBridge(perUserBridge{})
	app.Router().Use(func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *zhttp.Request) *zhttp.Response {
			return next(req).Header("X-User", req.Query("u")).Cookie("XSRF-TOKEN", "csrf-"+req.Query("u"))
		}
	})
	app.Router().Get("/cached", func(req *zhttp.Request) *zhttp.Response { return cached })
	// A response cache inside the global middleware keeps the first response
	// its handler produced; the outer middleware must not write into it.
	var memoMu sync.Mutex
	var memo *zhttp.Response
	app.Router().Group("/memo", func(g contracts.Router) {
		g.Get("/page", func(req *zhttp.Request) *zhttp.Response { return zhttp.JSON(map[string]string{"v": "shared"}) })
	}, func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *zhttp.Request) *zhttp.Response {
			memoMu.Lock()
			defer memoMu.Unlock()
			if memo == nil {
				memo = next(req)
			}
			return memo
		}
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	addr := serveApp(t, app)

	const conns, perConn = 8, 50
	errs := make(chan error, conns*perConn)
	var wg sync.WaitGroup
	for c := 0; c < conns; c++ {
		client := dialKeepAlive(t, addr)
		wg.Add(1)
		go func(c int) {
			defer wg.Done()
			for i := 0; i < perConn; i++ {
				u := fmt.Sprintf("u%d-%d", c, i)
				path := "/cached"
				if i%2 == 1 {
					path = "/memo/page"
				}
				resp, body, err := client.do("GET " + path + "?u=" + u + " HTTP/1.1\r\nHost: localhost\r\n\r\n")
				if err != nil {
					errs <- err
					return
				}
				if body != wantBody {
					errs <- fmt.Errorf("%s: body %q", u, body)
				}
				if got := resp.Header.Values("X-User"); len(got) != 1 || got[0] != u {
					errs <- fmt.Errorf("%s: X-User %v", u, got)
				}
				if got := resp.Header.Values("X-Bridge-User"); len(got) != 1 || got[0] != u {
					errs <- fmt.Errorf("%s: X-Bridge-User %v", u, got)
				}
				cookies := resp.Header.Values("Set-Cookie")
				if len(cookies) != 1 || !strings.HasPrefix(cookies[0], "XSRF-TOKEN=csrf-"+u+";") {
					errs <- fmt.Errorf("%s: Set-Cookie %v", u, cookies)
				}
			}
		}(c)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	if string(cached.Content()) != wantBody || cached.StatusCode() != 200 {
		t.Fatalf("cached response changed: %d %s", cached.StatusCode(), cached.Content())
	}
	if cached.GetHeader("X-Cache") != "hit" || cached.HasHeader("X-User") || cached.HasHeader("X-Bridge-User") || cached.HasHeader("X-Request-Id") {
		t.Fatalf("cached response headers changed: %v", cached.Headers())
	}
	if len(cached.Cookies()) != 0 {
		t.Fatalf("cached response gained cookies: %v", cached.Cookies())
	}
	if memo == nil || len(memo.Cookies()) != 0 || memo.HasHeader("X-User") || memo.HasHeader("X-Bridge-User") {
		t.Fatalf("memoized response was written by an outer layer: cookies=%v", memo.Cookies())
	}
}

// A trusted proxy that adds its own X-Forwarded-For line after the client's
// decides the client IP; the client's line cannot.
func TestForwardedForLinesThroughEngine(t *testing.T) {
	t.Setenv("APP_ENV", "local")
	t.Setenv("TRUSTED_PROXIES", "127.0.0.1")
	app := kernel.NewApplication(t.TempDir())
	t.Cleanup(func() { closeAppLog(t, app) })
	app.Router().Get("/ip", func(req *zhttp.Request) *zhttp.Response { return zhttp.Text(req.IP()) })
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	client := dialKeepAlive(t, serveApp(t, app))
	_, body, err := client.do("GET /ip HTTP/1.1\r\nHost: localhost\r\nX-Forwarded-For: 10.9.9.9\r\nX-Forwarded-For: 203.0.113.99\r\n\r\n")
	if err != nil {
		t.Fatal(err)
	}
	if body != "203.0.113.99" {
		t.Fatalf("IP()=%q, want the hop the proxy appended", body)
	}
}

type perUserBridge struct{}

func (perUserBridge) Middleware() []any { return nil }

func (perUserBridge) Finalize(req any, resp any) any {
	r, _ := req.(*zhttp.Request)
	out, _ := resp.(*zhttp.Response)
	if r == nil || out == nil {
		return resp
	}
	return out.Header("X-Bridge-User", r.Query("u"))
}

func bodyOf(r *zhttp.Request) string {
	b, err := r.Body()
	if err != nil {
		return "body error: " + err.Error()
	}
	return string(b)
}
