package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"runtime"
	"runtime/pprof"
	"runtime/trace"
	"strconv"
	"time"

	goccyjson "github.com/goccy/go-json"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/valyala/fasthttp"
	"github.com/zatrano/framework/v3/core/kernel"
	khttp "github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/rawhttp"
)

var (
	codecMarshal   = json.Marshal
	codecUnmarshal = json.Unmarshal
)

// loadCodec selects one codec for both servers. Empty and encodingjson keep
// encoding/json and do not call SetJSONCodec. goccy is set on both.
func loadCodec() error {
	switch os.Getenv("ZTFB_CODEC") {
	case "", "encodingjson":
		codecMarshal = json.Marshal
		codecUnmarshal = json.Unmarshal
		return nil
	case "goccy":
		codecMarshal = goccyjson.Marshal
		codecUnmarshal = goccyjson.Unmarshal
		khttp.SetJSONCodec(codecMarshal, codecUnmarshal)
		return nil
	default:
		return fmt.Errorf("ZTFB_CODEC %q", os.Getenv("ZTFB_CODEC"))
	}
}

const (
	helloBody = "Hello, World!"
	echoBody  = `{"message":"Hello, World!"}`
)

func cmdServe(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("serve <zatrano|fiber|fiber-tuned|rawhttp|fasthttp> <addr>")
	}
	startMemSnap()
	switch args[0] {
	case "zatrano":
		return serveZatrano(args[1])
	case "fiber":
		return serveFiber(args[1])
	case "fiber-tuned":
		return serveFiberTuned(args[1])
	case "rawhttp":
		return serveRawHTTP(args[1])
	case "fasthttp":
		return serveFastHTTP(args[1])
	default:
		return fmt.Errorf("unknown server %q", args[0])
	}
}

func serveZatrano(addr string) error {
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
	dir, err := os.MkdirTemp("", "ztfb-")
	if err != nil {
		return err
	}
	app := kernel.NewApplication(dir)
	app.Router().Get("/plaintext", func(*khttp.Request) *khttp.Response {
		return khttp.Text(helloBody)
	})
	app.Router().Get("/json", func(*khttp.Request) *khttp.Response {
		return khttp.JSON(map[string]string{"message": "Hello, World!"})
	})
	app.Router().Get("/users/{id}", func(req *khttp.Request) *khttp.Response {
		return khttp.JSON(map[string]string{"id": req.Route("id")})
	})
	app.Router().Post("/echo", func(req *khttp.Request) *khttp.Response {
		var m map[string]any
		// BindJSON decodes the connection buffer. Fiber's c.Body() also
		// does not copy. Body() on this request is unchanged.
		if err := req.BindJSON(&m); err != nil {
			return khttp.BadRequest("bad json")
		}
		return khttp.JSON(m)
	})
	return app.Run(addr)
}

func serveFiber(addr string) error {
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
			var b [16]byte
			if _, err := rand.Read(b[:]); err != nil {
				return "00000000000000000000000000000000"
			}
			return hex.EncodeToString(b[:])
		},
	}))
	app.Use(func(c fiber.Ctx) error {
		c.Set("X-Frame-Options", "SAMEORIGIN")
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		return c.Next()
	})
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
	return app.Listen(addr, fiber.ListenConfig{DisableStartupMessage: true})
}

// startProfiles writes runtime profiles when the env paths are set.
// It does not import net/http/pprof, so nothing listens for /debug/pprof.
func startProfiles() func() {
	var cpu *os.File
	if path := os.Getenv("ZTFB_CPU_PROFILE"); path != "" {
		f, err := os.Create(path)
		if err == nil {
			if err := pprof.StartCPUProfile(f); err != nil {
				_ = f.Close()
			} else {
				cpu = f
			}
		}
	}
	var tr *os.File
	if path := os.Getenv("ZTFB_TRACE"); path != "" {
		f, err := os.Create(path)
		if err == nil {
			if err := trace.Start(f); err != nil {
				_ = f.Close()
			} else {
				tr = f
			}
		}
	}
	return func() {
		if tr != nil {
			trace.Stop()
			_ = tr.Close()
			tr = nil
		}
		if cpu != nil {
			pprof.StopCPUProfile()
			_ = cpu.Close()
			cpu = nil
		}
		writeProfile("ZTFB_ALLOC_PROFILE", "allocs")
		writeProfile("ZTFB_HEAP_PROFILE", "heap")
		writeProfile("ZTFB_GOROUTINE_PROFILE", "goroutine")
	}
}

func serveRawHTTP(addr string) error {
	srv := &rawhttp.Server{
		Handler: func(ctx *rawhttp.Ctx) {
			ctx.SetContentType("text/plain; charset=utf-8")
			ctx.SetBodyString(helloBody)
		},
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return srv.Serve(ln)
}

func serveFastHTTP(addr string) error {
	srv := &fasthttp.Server{
		Handler: func(ctx *fasthttp.RequestCtx) {
			ctx.SetContentType("text/plain; charset=utf-8")
			ctx.SetBodyString(helloBody)
		},
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	return srv.ListenAndServe(addr)
}

func writeProfile(envKey, name string) {
	path := os.Getenv(envKey)
	if path == "" {
		return
	}
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()
	runtime.GC()
	if p := pprof.Lookup(name); p != nil {
		_ = p.WriteTo(f, 0)
	}
}
