package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/zatrano/framework/v3/core/kernel"
	khttp "github.com/zatrano/framework/v3/core/kernel/http"
)

const (
	helloBody = "Hello, World!"
	echoBody  = `{"message":"Hello, World!"}`
)

func cmdServe(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("serve <zatrano|fiber> <addr>")
	}
	switch args[0] {
	case "zatrano":
		return serveZatrano(args[1])
	case "fiber":
		return serveFiber(args[1])
	default:
		return fmt.Errorf("unknown server %q", args[0])
	}
}

func serveZatrano(addr string) error {
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
		if err := req.JSON(&m); err != nil {
			return khttp.BadRequest("bad json")
		}
		return khttp.JSON(m)
	})
	return app.Run(addr)
}

func serveFiber(addr string) error {
	app := fiber.New(fiber.Config{
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
		JSONEncoder:  json.Marshal,
		JSONDecoder:  json.Unmarshal,
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
		if err := json.Unmarshal(c.Body(), &m); err != nil {
			return c.Status(400).SendString("bad json")
		}
		return c.JSON(m)
	})
	return app.Listen(addr, fiber.ListenConfig{DisableStartupMessage: true})
}
