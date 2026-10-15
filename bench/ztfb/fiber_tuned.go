package main

import (
	"bytes"
	cryptorand "crypto/rand"
	"encoding/binary"
	mrand "math/rand/v2"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
)

// Tuned Fiber keeps the official server's routes, recover, and the four
// security headers. The request id is not fiber's requestid middleware:
// 16 random bytes are hex-encoded into a pooled [32]byte and written with
// SetBytesV. Locals is not used. A valid incoming id is copied through;
// an invalid one is replaced.

var jsonType = "application/json"

var hexIDPool = sync.Pool{
	New: func() any { return new([32]byte) },
}

var idSources = sync.Pool{
	New: func() any {
		var seed [32]byte
		if _, err := cryptorand.Read(seed[:]); err != nil {
			binary.LittleEndian.PutUint64(seed[:8], uint64(time.Now().UnixNano()))
		}
		return mrand.New(mrand.NewChaCha8(seed))
	},
}

func serveFiberTuned(addr string) error {
	if err := loadCodec(); err != nil {
		return err
	}
	stop := startProfiles()
	if sec, _ := strconv.Atoi(os.Getenv("ZTFB_PROFILE_SECONDS")); sec > 0 {
		go func() {
			time.Sleep(time.Duration(sec) * time.Second)
			stop()
			os.Exit(0)
		}()
	} else {
		defer stop()
	}
	return newFiberTunedApp().Listen(addr, fiber.ListenConfig{DisableStartupMessage: true})
}

func newFiberTunedApp() *fiber.App {
	return newFiberTuned(registerTunedRoutes)
}

func newFiberTuned(routes func(*fiber.App)) *fiber.App {
	app := fiber.New(fiber.Config{
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
		JSONEncoder:  codecMarshal,
		JSONDecoder:  codecUnmarshal,
	})
	app.Use(recover.New())
	app.Use(tunedRequestID)
	app.Use(tunedSecurity)
	if routes != nil {
		routes(app)
	}
	return app
}

func registerTunedRoutes(app *fiber.App) {
	app.Get("/plaintext", func(c fiber.Ctx) error {
		c.Set("Content-Type", "text/plain; charset=utf-8")
		return c.SendString(helloBody)
	})
	app.Get("/json", func(c fiber.Ctx) error {
		return c.JSON(map[string]string{"message": "Hello, World!"}, jsonType)
	})
	app.Get("/users/:id", func(c fiber.Ctx) error {
		return c.JSON(map[string]string{"id": c.Params("id")}, jsonType)
	})
	app.Post("/echo", func(c fiber.Ctx) error {
		var m map[string]any
		if err := codecUnmarshal(c.Body(), &m); err != nil {
			return c.Status(400).JSON(map[string]any{"message": "bad json"}, jsonType)
		}
		return c.JSON(m, jsonType)
	})
}

func tunedRequestID(c fiber.Ctx) error {
	id := fiberIncomingID(c)
	var pooled *[32]byte
	if len(id) == 0 {
		pooled = hexIDPool.Get().(*[32]byte)
		writeRequestHex(pooled)
		id = pooled[:]
	}
	c.Response().Header.SetBytesV("X-Request-Id", id)
	if pooled != nil {
		hexIDPool.Put(pooled)
	}
	err := c.Next()
	if tp := bytes.TrimSpace(c.Request().Header.Peek("Traceparent")); len(tp) > 0 {
		c.Response().Header.SetBytesV("Traceparent", tp)
	}
	return err
}

// tunedSecurity matches the official server's four headers. c.Set is the call
// that stays on the wire; SetBytesV on these four is dropped before the
// response is written. The request id above uses SetBytesV.
func tunedSecurity(c fiber.Ctx) error {
	c.Set("X-Frame-Options", "SAMEORIGIN")
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
	c.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
	return c.Next()
}

func fiberIncomingID(c fiber.Ctx) []byte {
	h := &c.Request().Header
	if id := bytes.TrimSpace(h.Peek("X-Request-ID")); validIDBytes(id) {
		return id
	}
	if id := bytes.TrimSpace(h.Peek("X-Correlation-ID")); validIDBytes(id) {
		return id
	}
	return traceIDBytes(h.Peek("Traceparent"))
}

func validIDBytes(b []byte) bool {
	if len(b) == 0 || len(b) > 128 {
		return false
	}
	for _, c := range b {
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case c == '.' || c == '_' || c == '-' || c == ':':
		default:
			return false
		}
	}
	return true
}

func traceIDBytes(header []byte) []byte {
	header = bytes.TrimSpace(header)
	if len(header) < 55 {
		return nil
	}
	parts := 0
	start := 0
	var trace []byte
	for i := 0; i <= len(header); i++ {
		if i == len(header) || header[i] == '-' {
			if parts == 1 {
				trace = header[start:i]
			}
			parts++
			start = i + 1
			if parts >= 4 {
				break
			}
		}
	}
	if parts < 4 || len(trace) != 32 || !validIDBytes(trace) {
		return nil
	}
	return trace
}

func writeRequestHex(dst *[32]byte) {
	src := idSources.Get().(*mrand.Rand)
	var raw [16]byte
	var n uint64
	for i := 0; i < 16; i++ {
		if i%8 == 0 {
			n = src.Uint64()
		}
		raw[i] = byte(n)
		n >>= 8
	}
	idSources.Put(src)
	const digits = "0123456789abcdef"
	for i := 0; i < 16; i++ {
		b := raw[i]
		dst[i*2] = digits[b>>4]
		dst[i*2+1] = digits[b&0x0f]
	}
}
