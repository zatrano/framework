package kernel

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/rawhttp"
)

func TestBodyBudgetSuccessAndRelease(t *testing.T) {
	app := NewApplication(t.TempDir())
	app.router.Post("/upload", func(*http.Request) *http.Response {
		return http.Text("ok")
	})
	srv := mustServer(t, app)
	ln := serveTestServer(t, srv)
	status, body := postRawPath(t, ln.Addr().String(), "/upload", "application/json", []byte(`{"a":1}`))
	if status != 200 || body != "ok" {
		t.Fatalf("status=%d body=%q", status, body)
	}
	if got := waitReserved(app, 0); got != 0 {
		t.Fatalf("reserved=%d", got)
	}
}

func TestBodyBudgetSingleRequestOverBudgetIs413(t *testing.T) {
	t.Setenv("HTTP_MAX_INFLIGHT_BODY_BYTES", fmt.Sprint(1<<20))
	app := NewApplication(t.TempDir())
	app.router.Post("/upload", func(*http.Request) *http.Response {
		return http.Text("ok")
	})
	srv := mustServer(t, app)
	ln := serveTestServer(t, srv)
	status := postDeclaredPath(t, ln.Addr().String(), "/upload", "application/json", 64)
	if status != 413 {
		t.Fatalf("status=%d", status)
	}
	if got := waitReserved(app, 0); got != 0 {
		t.Fatalf("reserved=%d", got)
	}
}

func TestBodyBudgetRejectsWith503AndRetryAfter(t *testing.T) {
	t.Setenv("HTTP_MAX_INFLIGHT_BODY_BYTES", fmt.Sprint(3<<20))
	app := NewApplication(t.TempDir())
	hold := make(chan struct{})
	app.router.Post("/upload", func(*http.Request) *http.Response {
		<-hold
		return http.Text("ok")
	})
	srv := mustServer(t, app)
	ln := serveTestServer(t, srv)
	addr := ln.Addr().String()
	body := bytes.Repeat([]byte("a"), 2<<20)

	first := make(chan int, 1)
	go func() {
		status, _ := postRawPath(t, addr, "/upload", "application/json", body)
		first <- status
	}()
	deadline := time.Now().Add(5 * time.Second)
	for app.BodyReserved() < 2<<20 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if app.BodyReserved() < 2<<20 {
		t.Fatalf("first request did not reserve, reserved=%d", app.BodyReserved())
	}
	st, hdr := postRawHeaders(t, addr, "/upload", "application/json", body, len(body))
	if st != 503 {
		t.Fatalf("status=%d", st)
	}
	if hdr["Retry-After"] != "1" {
		t.Fatalf("Retry-After=%q", hdr["Retry-After"])
	}
	close(hold)
	if got := <-first; got != 200 {
		t.Fatalf("first status=%d", got)
	}
	if got := waitReserved(app, 0); got != 0 {
		t.Fatalf("reserved=%d", got)
	}
}

func TestBodyBudgetPanicReleases(t *testing.T) {
	app := NewApplication(t.TempDir())
	app.router.Post("/boom", func(*http.Request) *http.Response {
		panic("budget")
	})
	srv := mustServer(t, app)
	ln := serveTestServer(t, srv)
	status, _ := postRawPath(t, ln.Addr().String(), "/boom", "application/json", []byte(`{"a":1}`))
	if status != 500 {
		t.Fatalf("status=%d", status)
	}
	if got := waitReserved(app, 0); got != 0 {
		t.Fatalf("reserved=%d", got)
	}
}

func TestBodyBudgetClientAbortReleases(t *testing.T) {
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
	fmt.Fprintf(c, "POST /upload HTTP/1.1\r\nHost: localhost\r\nContent-Type: application/json\r\nContent-Length: %d\r\n\r\n", 1<<20)
	_, _ = c.Write([]byte("partial"))
	deadline := time.Now().Add(2 * time.Second)
	for app.BodyReserved() == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if app.BodyReserved() == 0 {
		t.Fatal("abort did not reserve")
	}
	_ = c.Close()
	if got := waitReserved(app, 0); got != 0 {
		t.Fatalf("reserved=%d", got)
	}
}

func TestBodyBudgetReadTimeoutReleases(t *testing.T) {
	t.Setenv("HTTP_READ_TIMEOUT", "1")
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
	fmt.Fprintf(c, "POST /upload HTTP/1.1\r\nHost: localhost\r\nContent-Type: application/json\r\nContent-Length: %d\r\n\r\npartial", 1<<20)
	if got := waitReserved(app, 0); got != 0 {
		t.Fatalf("reserved=%d", got)
	}
}

func TestBodyBudgetKeepAliveSequential(t *testing.T) {
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
	req := "POST /upload HTTP/1.1\r\nHost: localhost\r\nContent-Type: application/json\r\nContent-Length: 2\r\n\r\n{}"
	if _, err := io.WriteString(c, req+req); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 4096)
	n, err := io.ReadAtLeast(c, buf, 32)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Count(buf[:n], []byte("200")) < 1 {
		t.Fatalf("response %q", buf[:n])
	}
	// The second response may still be in flight; read the rest.
	_ = c.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	rest := make([]byte, 4096)
	n2, _ := c.Read(rest)
	all := append(buf[:n], rest[:n2]...)
	if bytes.Count(all, []byte("HTTP/1.1 200")) < 2 {
		t.Fatalf("responses %q", all)
	}
	if got := waitReserved(app, 0); got != 0 {
		t.Fatalf("reserved=%d", got)
	}
}

func TestBodyBudget413Releases(t *testing.T) {
	app := NewApplication(t.TempDir())
	app.router.Post("/upload", func(*http.Request) *http.Response {
		return http.Text("ok")
	})
	srv := mustServer(t, app)
	ln := serveTestServer(t, srv)
	status := postChunked(t, ln.Addr().String(), "/upload", "application/json", 3<<20)
	if status != 413 {
		t.Fatalf("status=%d", status)
	}
	if got := waitReserved(app, 0); got != 0 {
		t.Fatalf("reserved=%d", got)
	}
}

func TestBodyBudgetLeakDrain(t *testing.T) {
	t.Run("tcp-rst-1000", func(t *testing.T) {
		drainTCP(t, 1000, true)
	})
	t.Run("tcp-fin-1000", func(t *testing.T) {
		drainTCP(t, 1000, false)
	})
	t.Run("memory-10000", func(t *testing.T) {
		drainMemory(t, 10000)
	})
}

func drainTCP(t *testing.T, n int, rst bool) {
	t.Helper()
	app := NewApplication(t.TempDir())
	app.router.Post("/upload", func(*http.Request) *http.Response {
		return http.Text("ok")
	})
	srv := mustServer(t, app)
	ln := serveTestServer(t, srv)
	addr := ln.Addr().String()
	if err := runDrain(n, func(k int) error { return oneDrainTCP(addr, k, rst) }); err != nil {
		t.Fatalf("tcp rst=%v: %v", rst, err)
	}
	waitBudgetIdle(t, app)
}

// drainMemory drives the same mix through Server.Serve on net.Pipe.
// Serve reports ConnState(StateClosed) from the accept-loop defer, the same
// hook a TCP connection uses. The pipe itself is not a TCP FIN or RST.
func drainMemory(t *testing.T, n int) {
	t.Helper()
	app := NewApplication(t.TempDir())
	app.router.Post("/upload", func(*http.Request) *http.Response {
		return http.Text("ok")
	})
	srv := mustServer(t, app)
	var closed atomic.Int64
	prev := srv.ConnState
	srv.ConnState = func(c net.Conn, st rawhttp.ConnState) {
		if prev != nil {
			prev(c, st)
		}
		if st == rawhttp.StateClosed {
			closed.Add(1)
		}
	}
	ln := newMemListener()
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })
	if err := runDrain(n, func(k int) error { return oneDrainPipe(ln, k) }); err != nil {
		t.Fatalf("memory: %v", err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for closed.Load() < int64(n) && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if got := closed.Load(); got != int64(n) {
		t.Fatalf("ConnState StateClosed=%d want %d", got, n)
	}
	waitBudgetIdle(t, app)
}

func runDrain(n int, one func(k int) error) error {
	errCh := make(chan error, n)
	var wg sync.WaitGroup
	var seq atomic.Int64
	const workers = 32
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for {
				k := int(seq.Add(1))
				if k > n {
					return
				}
				if err := one(k); err != nil {
					errCh <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			return err
		}
	}
	return nil
}

func oneDrainTCP(addr string, k int, rst bool) error {
	c, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}
	defer func() {
		if rst {
			rstClose(c)
			return
		}
		_ = c.Close()
	}()
	return writeDrain(c, k)
}

func oneDrainPipe(ln *memListener, k int) error {
	server, client := net.Pipe()
	select {
	case ln.conns <- server:
	case <-time.After(5 * time.Second):
		_ = client.Close()
		_ = server.Close()
		return fmt.Errorf("enqueue: accept did not take the pipe")
	}
	defer client.Close()
	return writeDrain(client, k)
}

func writeDrain(c net.Conn, k int) error {
	_ = c.SetDeadline(time.Now().Add(5 * time.Second))
	switch k % 5 {
	case 0:
		if _, err := io.WriteString(c, "POST /upload HTTP/1.1\r\nHost: localhost\r\nContent-Type: application/json\r\nContent-Length: 100000\r\n\r\nnope"); err != nil {
			return fmt.Errorf("write abort: %w", err)
		}
	case 1:
		if _, err := fmt.Fprintf(c, "POST /upload HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\nContent-Type: application/json\r\nContent-Length: %d\r\n\r\n", 3<<20); err != nil {
			return fmt.Errorf("write declared: %w", err)
		}
	default:
		if _, err := io.WriteString(c, "POST /upload HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\nContent-Type: application/json\r\nContent-Length: 2\r\n\r\n{}"); err != nil {
			return fmt.Errorf("write: %w", err)
		}
		buf := make([]byte, 256)
		n, err := c.Read(buf)
		if n == 0 {
			if err == nil {
				err = io.ErrUnexpectedEOF
			}
			return fmt.Errorf("read: %w", err)
		}
	}
	return nil
}

func waitBudgetIdle(t *testing.T, app *Application) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if app.BodyReserved() == 0 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("reserved=%d after 10s", app.BodyReserved())
}

type memListener struct {
	conns chan net.Conn
	stop  chan struct{}
	once  sync.Once
}

func newMemListener() *memListener {
	return &memListener{conns: make(chan net.Conn, 128), stop: make(chan struct{})}
}

func (l *memListener) Accept() (net.Conn, error) {
	select {
	case c, ok := <-l.conns:
		if !ok {
			return nil, net.ErrClosed
		}
		return c, nil
	case <-l.stop:
		return nil, net.ErrClosed
	}
}

func (l *memListener) Close() error {
	l.once.Do(func() { close(l.stop) })
	return nil
}

func (l *memListener) Addr() net.Addr { return memAddr{} }

type memAddr struct{}

func (memAddr) Network() string { return "pipe" }
func (memAddr) String() string  { return "pipe" }

func TestBodyBudgetConcurrentMultipart(t *testing.T) {
	app := NewApplication(t.TempDir())
	release := make(chan struct{})
	var entered atomic.Int32
	app.router.Post("/upload", func(*http.Request) *http.Response {
		entered.Add(1)
		<-release
		return http.Text("ok")
	})
	srv := mustServer(t, app)
	ln := serveTestServer(t, srv)
	addr := ln.Addr().String()
	payload := bytes.Repeat([]byte("m"), 30<<20)

	var peak atomic.Int64
	var heap atomic.Int64
	stop := make(chan struct{})
	go func() {
		var mem runtime.MemStats
		for {
			select {
			case <-stop:
				return
			default:
			}
			if r := app.BodyReserved(); r > peak.Load() {
				peak.Store(r)
			}
			runtime.ReadMemStats(&mem)
			if int64(mem.HeapAlloc) > heap.Load() {
				heap.Store(int64(mem.HeapAlloc))
			}
			time.Sleep(20 * time.Millisecond)
		}
	}()

	errCh := make(chan error, 20)
	var okN, rej, other atomic.Int32
	var wg sync.WaitGroup
	wg.Add(20)
	for i := 0; i < 20; i++ {
		go func() {
			defer wg.Done()
			st, err := postStream(addr, "/upload", "multipart/form-data; boundary=b", payload)
			if err != nil {
				errCh <- err
				return
			}
			switch st {
			case 200:
				okN.Add(1)
			case 503:
				rej.Add(1)
			default:
				other.Add(1)
			}
		}()
	}
	deadline := time.Now().Add(45 * time.Second)
	for entered.Load()+rej.Load()+other.Load() < 20 && len(errCh) == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	close(release)
	wg.Wait()
	close(stop)
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("postStream: %v", err)
		}
	}
	if peak.Load() > 256<<20 {
		t.Fatalf("peak reserved %d exceeds 256 MiB", peak.Load())
	}
	if rej.Load() == 0 || okN.Load() == 0 {
		t.Fatalf("ok=%d rejected=%d peak=%d", okN.Load(), rej.Load(), peak.Load())
	}
	if other.Load() != 0 {
		t.Fatalf("unexpected statuses=%d ok=%d rejected=%d", other.Load(), okN.Load(), rej.Load())
	}
	t.Logf("multipart peak reserved=%d heap=%d ok=%d rejected=%d", peak.Load(), heap.Load(), okN.Load(), rej.Load())
	if got := waitReserved(app, 0); got != 0 {
		t.Fatalf("reserved=%d", got)
	}
}

func TestBodyLimitOverBudgetBoot(t *testing.T) {
	app := NewApplication(t.TempDir())
	app.router.Post("/big", func(*http.Request) *http.Response {
		return http.Text("ok")
	}).BodyLimit(300 << 20)
	srv, err := app.httpServer(ListenOptions{})
	if err != nil || srv == nil {
		t.Fatal(err)
	}
	app.environment = "production"
	if _, err := app.httpServer(ListenOptions{}); err == nil {
		t.Fatal("production boot accepted a route cap above the budget")
	}
	app.environment = "local"
	t.Setenv("HTTP_STRICT_LIMITS", "true")
	if _, err := app.httpServer(ListenOptions{}); err == nil {
		t.Fatal("HTTP_STRICT_LIMITS accepted a route cap above the budget")
	}
}

func mustServer(t *testing.T, app *Application) *rawhttp.Server {
	t.Helper()
	app.life = lifeBooted
	srv, err := app.httpServer(ListenOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return srv
}

func waitReserved(app *Application, want int64) int64 {
	deadline := time.Now().Add(3 * time.Second)
	var got int64
	for time.Now().Before(deadline) {
		got = app.BodyReserved()
		if got == want {
			return got
		}
		time.Sleep(10 * time.Millisecond)
	}
	return app.BodyReserved()
}

// rstClose aborts the socket so the local port does not sit in TIME_WAIT.
func rstClose(c net.Conn) {
	if tc, ok := c.(*net.TCPConn); ok {
		_ = tc.SetLinger(0)
	}
	_ = c.Close()
}

func postStream(addr, path, contentType string, body []byte) (int, error) {
	c, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return -1, fmt.Errorf("dial: %w", err)
	}
	defer rstClose(c)
	_ = c.SetDeadline(time.Now().Add(60 * time.Second))
	if _, err := fmt.Fprintf(c, "POST %s HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\nContent-Type: %s\r\nContent-Length: %d\r\n\r\n", path, contentType, len(body)); err != nil {
		return -1, fmt.Errorf("write headers: %w", err)
	}
	stCh := make(chan int, 1)
	errCh := make(chan error, 1)
	go func() {
		buf := make([]byte, 256)
		n, rerr := c.Read(buf)
		if n == 0 {
			if rerr == nil {
				rerr = io.ErrUnexpectedEOF
			}
			errCh <- fmt.Errorf("read: %w", rerr)
			return
		}
		status, _ := parseStatusHeaders(buf[:n])
		if status == 0 {
			errCh <- fmt.Errorf("read: no status in %q", buf[:n])
			return
		}
		stCh <- status
	}()
	if _, err := c.Write(body); err != nil {
		select {
		case st := <-stCh:
			return st, nil
		case rerr := <-errCh:
			return -1, fmt.Errorf("write body: %w (read: %v)", err, rerr)
		default:
			return -1, fmt.Errorf("write body: %w", err)
		}
	}
	select {
	case st := <-stCh:
		return st, nil
	case err := <-errCh:
		return -1, err
	case <-time.After(60 * time.Second):
		return -1, fmt.Errorf("read: timeout")
	}
}

func postRawHeaders(t *testing.T, addr, path, contentType string, body []byte, n int) (int, map[string]string) {
	t.Helper()
	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(10 * time.Second))
	fmt.Fprintf(c, "POST %s HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\nContent-Type: %s\r\nContent-Length: %d\r\n\r\n", path, contentType, n)
	if len(body) > 0 {
		if _, err := c.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := io.ReadAll(c)
	if err != nil {
		t.Fatal(err)
	}
	return parseStatusHeaders(raw)
}

func parseStatusHeaders(raw []byte) (int, map[string]string) {
	line, rest, _ := bytes.Cut(raw, []byte("\r\n"))
	fields := bytes.Fields(line)
	status := 0
	if len(fields) >= 2 {
		fmt.Sscanf(string(fields[1]), "%d", &status)
	}
	hdr := map[string]string{}
	for _, row := range bytes.Split(rest, []byte("\r\n")) {
		if len(row) == 0 {
			break
		}
		k, v, ok := bytes.Cut(row, []byte(":"))
		if ok {
			hdr[string(k)] = string(bytes.TrimSpace(v))
		}
	}
	return status, hdr
}
