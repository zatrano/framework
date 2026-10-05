package http

import (
	"io"
	"net"
	stdhttp "net/http"
	"strings"
	"testing"
	"time"

	"github.com/zatrano/rawhttp"
)

func TestStreamRearmsWriteDeadlinePerChunk(t *testing.T) {
	const writeTO = 200 * time.Millisecond
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &rawhttp.Server{
		ReadTimeout:  -1,
		WriteTimeout: writeTO,
		IdleTimeout:  -1,
		Handler: func(ctx *rawhttp.Ctx) {
			resp := Stream("text/plain", func(w stdhttp.ResponseWriter, flusher stdhttp.Flusher) error {
				time.Sleep(500 * time.Millisecond)
				if _, err := io.WriteString(w, "late-ok"); err != nil {
					return err
				}
				flusher.Flush()
				return nil
			})
			resp.PrepareStreamDeadline(writeTO)
			_ = resp.Commit(ctx)
		},
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	body := getBody(t, ln.Addr().String())
	if !strings.Contains(body, "late-ok") {
		t.Fatalf("re-armed stream was cut: %q", body)
	}
}

func TestResponseWithoutClearWriteDeadlineIsCut(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &rawhttp.Server{
		ReadTimeout:  -1,
		WriteTimeout: 200 * time.Millisecond,
		IdleTimeout:  -1,
		Handler: func(ctx *rawhttp.Ctx) {
			resp := &Response{
				status:      stdhttp.StatusOK,
				contentType: "text/plain",
				headers:     make(stdhttp.Header),
				stream: func(w stdhttp.ResponseWriter, flusher stdhttp.Flusher) error {
					time.Sleep(500 * time.Millisecond)
					_, err := io.WriteString(w, "late-ok")
					flusher.Flush()
					return err
				},
			}
			_ = resp.Commit(ctx)
		},
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	body := getBody(t, ln.Addr().String())
	if strings.Contains(body, "late-ok") {
		t.Fatal("write timeout did not apply to a response that kept the deadline")
	}
}

func TestStreamClosesWhenClientStopsReading(t *testing.T) {
	const writeTO = 300 * time.Millisecond
	errCh := make(chan error, 1)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &rawhttp.Server{
		ReadTimeout:  -1,
		WriteTimeout: writeTO,
		IdleTimeout:  -1,
		Handler: func(ctx *rawhttp.Ctx) {
			resp := Stream("application/octet-stream", func(w stdhttp.ResponseWriter, flusher stdhttp.Flusher) error {
				buf := make([]byte, 32<<10)
				for i := 0; i < 256; i++ {
					if _, err := w.Write(buf); err != nil {
						errCh <- err
						return err
					}
					flusher.Flush()
				}
				errCh <- nil
				return nil
			})
			resp.PrepareStreamDeadline(writeTO)
			_ = resp.Commit(ctx)
		},
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	c, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	start := time.Now()
	if _, err := io.WriteString(c, "GET / HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("stalled client did not trip the write deadline")
		}
		if elapsed := time.Since(start); elapsed > 2*time.Second {
			t.Fatalf("connection stayed open for %v, want within the write timeout", elapsed)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("stalled stream did not close")
	}
}

func TestSlowStreamIsNotCut(t *testing.T) {
	const writeTO = 500 * time.Millisecond
	const chunks = 8
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &rawhttp.Server{
		ReadTimeout:  -1,
		WriteTimeout: writeTO,
		IdleTimeout:  -1,
		Handler: func(ctx *rawhttp.Ctx) {
			resp := Stream("application/octet-stream", func(w stdhttp.ResponseWriter, flusher stdhttp.Flusher) error {
				buf := make([]byte, 1024)
				for i := 0; i < chunks; i++ {
					if _, err := w.Write(buf); err != nil {
						return err
					}
					flusher.Flush()
				}
				return nil
			})
			resp.PrepareStreamDeadline(writeTO)
			_ = resp.Commit(ctx)
		},
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	c, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(8 * time.Second))
	if _, err := io.WriteString(c, "GET / HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"); err != nil {
		t.Fatal(err)
	}
	got := 0
	tmp := make([]byte, 256)
	deadline := time.Now().Add(6 * time.Second)
	for got < chunks*1024 && time.Now().Before(deadline) {
		_ = c.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		n, rerr := c.Read(tmp)
		got += n
		if rerr != nil && n == 0 {
			time.Sleep(40 * time.Millisecond)
		}
	}
	if got < chunks*1024 {
		t.Fatalf("slow reader got %d bytes, want the full stream", got)
	}
}

func getBody(t *testing.T, addr string) string {
	t.Helper()
	c, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(3 * time.Second))
	if _, err := io.WriteString(c, "GET / HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"); err != nil {
		t.Fatal(err)
	}
	buf, _ := io.ReadAll(c)
	return string(buf)
}
