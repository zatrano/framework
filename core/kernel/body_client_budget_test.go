package kernel

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/framework/v3/core/kernel/log"
	"github.com/zatrano/rawhttp"
)

func TestClientBudgetKeyGroupsIPv6(t *testing.T) {
	a := clientBudgetKey("2001:db8:1:2::1")
	b := clientBudgetKey("2001:db8:1:2::ffff")
	c := clientBudgetKey("2001:db8:1:3::1")
	if a != b || a == c {
		t.Fatalf("keys %q %q %q", a, b, c)
	}
	if clientBudgetKey("203.0.113.9") != "203.0.113.9" {
		t.Fatal("ipv4 key")
	}
	if _, ok := clientShareKey("127.0.0.1:9", "127.0.0.1", false); ok {
		t.Fatal("loopback must not take a share")
	}
	if _, ok := clientShareKey("10.1.2.3:9", "10.1.2.3", false); ok {
		t.Fatal("rfc1918 must not take a share")
	}
	if _, ok := clientShareKey("100.64.1.1:9", "100.64.1.1", false); ok {
		t.Fatal("cgnat must not take a share")
	}
	if _, ok := clientShareKey("[fd00::1]:9", "fd00::1", false); ok {
		t.Fatal("ula must not take a share")
	}
	if _, ok := clientShareKey("169.254.1.1:9", "169.254.1.1", false); ok {
		t.Fatal("link-local must not take a share")
	}
	key, ok := clientShareKey("203.0.113.8:9", "203.0.113.8", false)
	if !ok || key != "203.0.113.8" {
		t.Fatalf("public key %q %v", key, ok)
	}
	fwd, ok := clientShareKey("127.0.0.1:9", "203.0.113.8", true)
	if !ok || fwd != "203.0.113.8" {
		t.Fatalf("trusted forward %q %v", fwd, ok)
	}
	if _, ok := clientShareKey("127.0.0.1:9", "127.0.0.1", true); ok {
		t.Fatal("trusted proxy without a distinct client must not take a share")
	}
	left, ok1 := clientShareKey("[2001:db8:1:2::1]:9", "2001:db8:1:2::1", false)
	right, ok2 := clientShareKey("[2001:db8:1:2::ffff]:9", "2001:db8:1:2::ffff", false)
	if !ok1 || !ok2 || left != right {
		t.Fatalf("ipv6 share %q %q %v %v", left, right, ok1, ok2)
	}
}

func TestPerClientShareRejectsFifthUpload(t *testing.T) {
	t.Setenv("HTTP_MAX_INFLIGHT_BODY_BYTES_PER_CLIENT", fmt.Sprint(int64(120<<20)))
	t.Setenv("TRUSTED_PROXIES", "127.0.0.1")
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

	var rej atomic.Int32
	var wg sync.WaitGroup
	errCh := make(chan error, 16)
	send := func(ip string, n int) {
		wg.Add(n)
		for i := 0; i < n; i++ {
			go func() {
				defer wg.Done()
				st, err := postStreamForwarded(addr, "/upload", "multipart/form-data; boundary=b", payload, ip)
				if err != nil {
					errCh <- err
					return
				}
				if st == 503 {
					rej.Add(1)
				}
			}()
		}
	}
	send("203.0.113.8", 9)
	send("203.0.113.9", 1)
	deadline := time.Now().Add(45 * time.Second)
	for entered.Load()+rej.Load() < 10 && len(errCh) == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if entered.Load() != 5 || rej.Load() != 5 {
		reportStreamErr(t, errCh)
		t.Fatalf("entered=%d rejected=%d", entered.Load(), rej.Load())
	}
	close(release)
	wg.Wait()
	reportStreamErr(t, errCh)
	if got := waitReserved(app, 0); got != 0 || app.ClientReserved() != 0 {
		t.Fatalf("reserved=%d client=%d", got, app.ClientReserved())
	}
}

func TestUndistinguishedPeersUseOnlyTheGlobalBudget(t *testing.T) {
	t.Setenv("HTTP_MAX_INFLIGHT_BODY_BYTES_PER_CLIENT", fmt.Sprint(int64(30<<20)))
	for _, peer := range []string{"", "192.168.5.5"} {
		peer := peer
		t.Run(peer, func(t *testing.T) {
			app := NewApplication(t.TempDir())
			release := make(chan struct{})
			var entered atomic.Int32
			app.router.Post("/upload", func(*http.Request) *http.Response {
				entered.Add(1)
				<-release
				return http.Text("ok")
			})
			srv := mustServer(t, app)
			var ln net.Listener
			if peer == "" {
				ln = serveTestServer(t, srv)
			} else {
				ln = serveSpoofedPeer(t, srv, peer)
			}
			payload := bytes.Repeat([]byte("m"), 30<<20)
			errCh := make(chan error, 16)
			var rej atomic.Int32
			var wg sync.WaitGroup
			for i := 0; i < 9; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					st, err := postStreamForwarded(ln.Addr().String(), "/upload", "application/octet-stream", payload, "203.0.113.9")
					if err != nil {
						errCh <- err
						return
					}
					if st == 503 {
						rej.Add(1)
					}
				}()
			}
			deadline := time.Now().Add(45 * time.Second)
			for entered.Load()+rej.Load() < 9 && len(errCh) == 0 && time.Now().Before(deadline) {
				time.Sleep(20 * time.Millisecond)
			}
			if entered.Load() != 8 || rej.Load() != 1 {
				reportStreamErr(t, errCh)
				t.Fatalf("entered=%d rejected=%d", entered.Load(), rej.Load())
			}
			close(release)
			wg.Wait()
			reportStreamErr(t, errCh)
			if got := waitReserved(app, 0); got != 0 || app.ClientReserved() != 0 {
				t.Fatalf("reserved=%d client=%d", got, app.ClientReserved())
			}
		})
	}
}

func TestPublicPeerTakesTheShare(t *testing.T) {
	t.Setenv("HTTP_MAX_INFLIGHT_BODY_BYTES_PER_CLIENT", fmt.Sprint(int64(30<<20)))
	app := NewApplication(t.TempDir())
	release := make(chan struct{})
	var entered atomic.Int32
	app.router.Post("/upload", func(*http.Request) *http.Response {
		entered.Add(1)
		<-release
		return http.Text("ok")
	})
	srv := mustServer(t, app)
	ln := serveSpoofedPeer(t, srv, "203.0.113.8")
	payload := bytes.Repeat([]byte("m"), 30<<20)
	errCh := make(chan error, 16)
	var rej atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 9; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			st, err := postStreamForwarded(ln.Addr().String(), "/upload", "application/octet-stream", payload, "198.51.100.2")
			if err != nil {
				errCh <- err
				return
			}
			if st == 503 {
				rej.Add(1)
			}
		}()
	}
	deadline := time.Now().Add(45 * time.Second)
	for entered.Load()+rej.Load() < 9 && len(errCh) == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if entered.Load() != 1 || rej.Load() != 8 {
		reportStreamErr(t, errCh)
		t.Fatalf("entered=%d rejected=%d", entered.Load(), rej.Load())
	}
	close(release)
	wg.Wait()
	reportStreamErr(t, errCh)
}

func TestFakeForwardedForIsIgnored(t *testing.T) {
	t.Setenv("HTTP_MAX_INFLIGHT_BODY_BYTES_PER_CLIENT", fmt.Sprint(64<<10))
	app := NewApplication(t.TempDir())
	release := make(chan struct{})
	var entered atomic.Int32
	app.router.Post("/upload", func(*http.Request) *http.Response {
		entered.Add(1)
		<-release
		return http.Text("ok")
	})
	srv := mustServer(t, app)
	ln := serveSpoofedPeer(t, srv, "203.0.113.50")
	addr := ln.Addr().String()
	body := bytes.Repeat([]byte("a"), 64<<10)
	done := make(chan streamResult, 1)
	go func() {
		st, err := postStreamForwarded(addr, "/upload", "application/json", body, "203.0.113.1")
		done <- streamResult{st: st, err: err}
	}()
	deadline := time.Now().Add(3 * time.Second)
	for entered.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	st, err := postStreamForwarded(addr, "/upload", "application/json", body, "203.0.113.2")
	if err != nil {
		t.Fatalf("second post: %v", err)
	}
	close(release)
	first := <-done
	if first.err != nil {
		t.Fatalf("first post: %v", first.err)
	}
	if entered.Load() != 1 || st != 503 || first.st != 200 {
		t.Fatalf("entered=%d first=%d second=%d", entered.Load(), first.st, st)
	}
}

func TestPerClientShareWarnsWithoutTrustedProxy(t *testing.T) {
	app := NewApplication(t.TempDir())
	path := t.TempDir() + "/zatrano.log"
	logger, err := log.New("warning", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = logger.Close() })
	app.logger = logger
	app.router.Post("/upload", func(*http.Request) *http.Response {
		return http.Text("ok")
	})
	srv := mustServer(t, app)
	ln := serveTestServer(t, srv)
	body := bytes.Repeat([]byte("a"), 32)
	st, err := postStreamForwarded(ln.Addr().String(), "/upload", "application/json", body, "203.0.113.1")
	if err != nil {
		t.Fatal(err)
	}
	if st != 200 {
		t.Fatalf("status %d", st)
	}
	st, err = postStreamForwarded(ln.Addr().String(), "/upload", "application/json", body, "203.0.113.2")
	if err != nil {
		t.Fatal(err)
	}
	if st != 200 {
		t.Fatalf("second status %d", st)
	}
	_ = logger.Close()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte("not a distinct client")) {
		t.Fatalf("log %q", raw)
	}
	if bytes.Contains(raw, []byte("127.0.0.1")) || bytes.Contains(raw, []byte("203.0.113")) {
		t.Fatalf("warning must not include an address: %q", raw)
	}
	if bytes.Count(raw, []byte("not a distinct client")) != 1 {
		t.Fatalf("warning once, log %q", raw)
	}
}

type spoofListener struct {
	net.Listener
	ip net.IP
}

func (l spoofListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return &spoofConn{Conn: c, ip: l.ip}, nil
}

type spoofConn struct {
	net.Conn
	ip net.IP
}

func (c *spoofConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: c.ip, Port: 9}
}

func serveSpoofedPeer(t *testing.T, srv *rawhttp.Server, ip string) net.Listener {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = srv.Close() })
	wrapped := &spoofListener{Listener: ln, ip: net.ParseIP(ip)}
	go func() { _ = srv.Serve(wrapped) }()
	return ln
}

func postStreamForwarded(addr, path, contentType string, body []byte, forwarded string) (int, error) {
	c, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return -1, fmt.Errorf("dial: %w", err)
	}
	defer rstClose(c)
	_ = c.SetDeadline(time.Now().Add(60 * time.Second))
	if _, err := fmt.Fprintf(c, "POST %s HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\nContent-Type: %s\r\nContent-Length: %d\r\nX-Forwarded-For: %s\r\n\r\n", path, contentType, len(body), forwarded); err != nil {
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

type streamResult struct {
	st  int
	err error
}

func reportStreamErr(t *testing.T, errCh <-chan error) {
	t.Helper()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatal(err)
		}
	default:
	}
}
