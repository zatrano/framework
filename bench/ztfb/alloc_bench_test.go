package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"io"
	"net"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

func BenchmarkFiberPlaintext(b *testing.B) {
	fiberBench(b, "GET /plaintext HTTP/1.1\r\nHost: localhost\r\n\r\n", fiberRoutesSame)
}

func BenchmarkFiberJSON(b *testing.B) {
	fiberBench(b, "GET /json HTTP/1.1\r\nHost: localhost\r\n\r\n", fiberRoutesSame)
}

func BenchmarkFiberUsers(b *testing.B) {
	fiberBench(b, "GET /users/42 HTTP/1.1\r\nHost: localhost\r\n\r\n", fiberRoutesSame)
}

func BenchmarkFiberEcho(b *testing.B) {
	fiberBench(b, "POST /echo HTTP/1.1\r\nHost: localhost\r\nContent-Type: application/json\r\nContent-Length: 27\r\n\r\n{\"message\":\"Hello, World!\"}", fiberRoutesSame)
}

func BenchmarkFiberJSONFramework(b *testing.B) {
	fiberBench(b, "GET /json HTTP/1.1\r\nHost: localhost\r\n\r\n", fiberRoutesFramework)
}

func BenchmarkFiberUsersFramework(b *testing.B) {
	fiberBench(b, "GET /users/42 HTTP/1.1\r\nHost: localhost\r\n\r\n", fiberRoutesFramework)
}

var (
	fiberJSONPayload = map[string]string{"message": "Hello, World!"}
	fiberUserPayload = map[string]string{"id": ""}
)

func fiberRoutesSame(app *fiber.App) {
	app.Get("/plaintext", func(c fiber.Ctx) error {
		c.Set("Content-Type", "text/plain; charset=utf-8")
		return c.SendString(helloBody)
	})
	app.Get("/json", func(c fiber.Ctx) error {
		return c.JSON(map[string]string{"message": "Hello, World!"})
	})
	app.Get("/users/:id", func(c fiber.Ctx) error {
		return c.JSON(map[string]string{"id": c.Params("id")})
	})
	app.Post("/echo", func(c fiber.Ctx) error {
		var m map[string]any
		if err := codecUnmarshal(c.Body(), &m); err != nil {
			return c.Status(400).SendString("bad json")
		}
		return c.JSON(m)
	})
}

func fiberRoutesFramework(app *fiber.App) {
	app.Get("/json", func(c fiber.Ctx) error {
		return c.JSON(fiberJSONPayload)
	})
	app.Get("/users/:id", func(c fiber.Ctx) error {
		fiberUserPayload["id"] = c.Params("id")
		return c.JSON(fiberUserPayload)
	})
}

func fiberBench(b *testing.B, raw string, routes func(*fiber.App)) {
	b.Helper()
	app := fiber.New(fiber.Config{
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
		JSONEncoder:  codecMarshal,
		JSONDecoder:  codecUnmarshal,
	})
	app.Use(recover.New())
	app.Use(requestid.New(requestid.Config{
		Header: "X-Request-Id",
		Generator: func() string {
			var buf [16]byte
			if _, err := rand.Read(buf[:]); err != nil {
				return "00000000000000000000000000000000"
			}
			return hex.EncodeToString(buf[:])
		},
	}))
	app.Use(func(c fiber.Ctx) error {
		c.Set("X-Frame-Options", "SAMEORIGIN")
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		return c.Next()
	})
	routes(app)

	left, right := net.Pipe()
	ln := &oneConnListener{conn: left, stop: make(chan struct{})}
	errCh := make(chan error, 1)
	go func() { errCh <- app.Listener(ln, fiber.ListenConfig{DisableStartupMessage: true}) }()
	b.Cleanup(func() {
		ln.Close()
		_ = app.Shutdown()
		_ = left.Close()
		_ = right.Close()
	})

	req := []byte(raw)
	ack := make([]byte, 2048)
	for i := 0; i < 64; i++ {
		if _, err := right.Write(req); err != nil {
			b.Fatal(err)
		}
		if err := drainOneResponse(right, ack); err != nil {
			b.Fatal(err)
		}
	}
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := right.Write(req); err != nil {
			b.Fatal(err)
		}
		if err := drainOneResponse(right, ack); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	runtime.ReadMemStats(&after)
	if b.N > 0 {
		b.ReportMetric(float64(after.Mallocs-before.Mallocs)/float64(b.N), "mallocs/req")
		b.ReportMetric(float64(after.TotalAlloc-before.TotalAlloc)/float64(b.N), "bytes/req")
	}
	select {
	case err := <-errCh:
		if err != nil {
			b.Log(err)
		}
	default:
	}
}

type oneConnListener struct {
	conn net.Conn
	once sync.Once
	stop chan struct{}
}

func (l *oneConnListener) Accept() (net.Conn, error) {
	var conn net.Conn
	l.once.Do(func() { conn = l.conn })
	if conn != nil {
		return conn, nil
	}
	<-l.stop
	return nil, io.EOF
}

func (l *oneConnListener) Close() error {
	select {
	case <-l.stop:
	default:
		close(l.stop)
	}
	return nil
}

func (l *oneConnListener) Addr() net.Addr { return pipeAddr{} }

type pipeAddr struct{}

func (pipeAddr) Network() string { return "tcp" }
func (pipeAddr) String() string  { return "127.0.0.1:1" }

func drainOneResponse(c net.Conn, buf []byte) error {
	n := 0
	for {
		if n == len(buf) {
			return io.ErrShortBuffer
		}
		r, err := c.Read(buf[n:])
		n += r
		if err != nil {
			return err
		}
		if i := bytes.Index(buf[:n], headerSplit); i >= 0 {
			body := n - (i + 4)
			need := contentLen(buf[:i])
			for body < need {
				if n == len(buf) {
					return io.ErrShortBuffer
				}
				r, err = c.Read(buf[n:])
				n += r
				body += r
				if err != nil {
					return err
				}
			}
			return nil
		}
	}
}

var headerSplit = []byte("\r\n\r\n")
var contentLenKey = []byte("Content-Length: ")

func contentLen(header []byte) int {
	i := bytes.Index(header, contentLenKey)
	if i < 0 {
		return 0
	}
	rest := header[i+len(contentLenKey):]
	n := 0
	for _, c := range rest {
		if c < '0' || c > '9' {
			break
		}
		n = n*10 + int(c-'0')
	}
	return n
}
