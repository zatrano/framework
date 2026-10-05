package bench

import (
	"crypto/tls"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gofiber/fiber/v2"
	"github.com/labstack/echo/v4"
	"github.com/valyala/fasthttp"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

func applyTier1(h http.Header, c captured) {
	h.Set("Content-Type", c.contentType)
	h.Set("X-Request-ID", requestID)
	for name, value := range c.headers {
		h.Set(name, value)
	}
}

var fiberApps struct {
	once  sync.Once
	t0    *fiber.App
	t1    *fiber.App
	t1gen *fiber.App
	err   error
}

func fiberReady() error {
	fiberApps.once.Do(func() {
		probe, err := zatranoProbe()
		if err != nil {
			fiberApps.err = err
			return
		}
		t0 := fiber.New(fiber.Config{DisableStartupMessage: true})
		t0.Get("/plaintext", func(c *fiber.Ctx) error {
			if err := c.SendString(hello); err != nil {
				return err
			}
			c.Set("Content-Type", probe.contentType)
			return nil
		})
		t1 := fiber.New(fiber.Config{DisableStartupMessage: true})
		t1.Get("/plaintext", func(c *fiber.Ctx) error {
			return fiberPlaintext(c, probe, false)
		})
		t1gen := fiber.New(fiber.Config{DisableStartupMessage: true})
		t1gen.Get("/plaintext", func(c *fiber.Ctx) error {
			return fiberPlaintext(c, probe, true)
		})
		fiberApps.t0 = t0
		fiberApps.t1 = t1
		fiberApps.t1gen = t1gen
	})
	return fiberApps.err
}

func serveFiber(app *fiber.App, conn net.Conn) error {
	srv := &fasthttp.Server{Handler: app.Handler()}
	return srv.ServeConn(conn)
}

// serveFiberEqual sets the same read, write, and idle durations as ZATRANO
// production. fasthttp has no ReadHeaderTimeout; header time is inside
// ReadTimeout (60s), not a separate 10s.
func serveFiberEqual(app *fiber.App, conn net.Conn) error {
	srv := &fasthttp.Server{
		Handler:      app.Handler(),
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	return srv.ServeConn(conn)
}

func serveFiberTier0(conn net.Conn) error {
	if err := fiberReady(); err != nil {
		return err
	}
	return serveFiber(fiberApps.t0, conn)
}

func serveFiberTier1(conn net.Conn) error {
	if err := fiberReady(); err != nil {
		return err
	}
	return serveFiber(fiberApps.t1, conn)
}

func serveFiberTier1Gen(conn net.Conn) error {
	if err := fiberReady(); err != nil {
		return err
	}
	return serveFiber(fiberApps.t1gen, conn)
}

func serveFiberTier0Eq(conn net.Conn) error {
	if err := fiberReady(); err != nil {
		return err
	}
	return serveFiberEqual(fiberApps.t0, conn)
}

func serveFiberTier1Eq(conn net.Conn) error {
	if err := fiberReady(); err != nil {
		return err
	}
	return serveFiberEqual(fiberApps.t1, conn)
}

func serveFiberTier1GenEq(conn net.Conn) error {
	if err := fiberReady(); err != nil {
		return err
	}
	return serveFiberEqual(fiberApps.t1gen, conn)
}

func fiberPlaintext(c *fiber.Ctx, probe captured, generate bool) error {
	if err := c.SendString(hello); err != nil {
		return err
	}
	c.Set("Content-Type", probe.contentType)
	id := c.Get("X-Request-ID")
	if !idPattern.MatchString(id) {
		if generate {
			id = newHexID()
		} else {
			id = ""
		}
	}
	if id != "" {
		c.Set("X-Request-ID", id)
	}
	for name, value := range probe.headers {
		c.Set(name, value)
	}
	return nil
}

type onceListener struct {
	conn net.Conn
	addr net.Addr
	done chan struct{}
}

func (l *onceListener) Accept() (net.Conn, error) {
	if l.conn != nil {
		c := l.conn
		l.conn = nil
		return &closeConn{Conn: c, done: l.done}, nil
	}
	<-l.done
	return nil, net.ErrClosed
}

func (l *onceListener) Close() error   { return nil }
func (l *onceListener) Addr() net.Addr { return l.addr }

// closeConn reports when net/http finishes the connection so Accept can return.
type closeConn struct {
	net.Conn
	once sync.Once
	done chan struct{}
}

func (c *closeConn) Close() error {
	c.once.Do(func() { close(c.done) })
	return c.Conn.Close()
}

func serveHTTP(h http.Handler, conn net.Conn) error {
	done := make(chan struct{})
	srv := &http.Server{
		Handler: h,
		// Empty map keeps the server on HTTP/1.1 for this one connection.
		TLSNextProto: map[string]func(*http.Server, *tls.Conn, http.Handler){},
	}
	return srv.Serve(&onceListener{conn: conn, addr: benchAddr, done: done})
}

func ginTier0() http.Handler {
	r := gin.New()
	r.GET("/plaintext", func(c *gin.Context) {
		c.Data(http.StatusOK, tier0ContentType(), []byte(hello))
	})
	return r
}

func ginTier1() http.Handler {
	r := gin.New()
	r.GET("/plaintext", func(c *gin.Context) {
		probe, err := zatranoProbe()
		if err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
		applyTier1(c.Writer.Header(), probe)
		c.Status(http.StatusOK)
		_, _ = c.Writer.Write([]byte(hello))
	})
	return r
}

func echoTier0() http.Handler {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.GET("/plaintext", func(c echo.Context) error {
		return c.Blob(http.StatusOK, tier0ContentType(), []byte(hello))
	})
	return e
}

func echoTier1() http.Handler {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.GET("/plaintext", func(c echo.Context) error {
		probe, err := zatranoProbe()
		if err != nil {
			return err
		}
		applyTier1(c.Response().Header(), probe)
		return c.Blob(http.StatusOK, probe.contentType, []byte(hello))
	})
	return e
}

func tier0ContentType() string {
	return plainContentType()
}

var plainContentType = sync.OnceValue(func() string {
	c, buf := captureConn()
	_ = serveZatranoTier0(c)
	got, err := readCaptured(buf)
	if err != nil || got.contentType == "" {
		return "text/plain; charset=utf-8"
	}
	return got.contentType
})

var (
	gin0  = sync.OnceValue(ginTier0)
	gin1  = sync.OnceValue(ginTier1)
	echo0 = sync.OnceValue(echoTier0)
	echo1 = sync.OnceValue(echoTier1)
)

func serveGinTier0(conn net.Conn) error  { return serveHTTP(gin0(), conn) }
func serveGinTier1(conn net.Conn) error  { return serveHTTP(gin1(), conn) }
func serveEchoTier0(conn net.Conn) error { return serveHTTP(echo0(), conn) }
func serveEchoTier1(conn net.Conn) error { return serveHTTP(echo1(), conn) }
