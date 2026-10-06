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
	ln := serveTestServer(t, srv)
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
	if _, err := app.httpServer(ListenOptions{}); err != nil {
		t.Fatal(err)
	}
	_ = logger.Close()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(raw, []byte("share applies to the proxy")) {
		t.Fatalf("log %q", raw)
	}
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
