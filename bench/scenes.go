package bench

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/zatrano/framework/v3/core/contracts"
	khttp "github.com/zatrano/framework/v3/core/kernel/http"
)

// sceneEcho is the JSON document POST /echo reads and writes back.
type sceneEcho struct {
	Name string `json:"name"`
	Pad  string `json:"pad"`
}

type sceneUser struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type sceneSearch struct {
	Q    string `json:"q"`
	Page string `json:"page"`
}

type sceneMessage struct {
	Message string `json:"message"`
}

const sceneEchoBytes = 1024

var (
	sceneEchoBody = mustEchoBody(sceneEchoBytes)
	requestJSON   = sceneGet("/json")
	requestUser   = sceneGet("/users/42")
	requestSearch = sceneGet("/search?q=%20Ada%20&page=1")
	requestEcho   = scenePost("/echo", "application/json", sceneEchoBody)
)

func mustEchoBody(n int) []byte {
	// {"name":"Ada","pad":""} is the empty-pad encoding. Each pad byte adds one.
	empty, err := json.Marshal(sceneEcho{Name: "Ada"})
	if err != nil {
		panic(err)
	}
	padLen := n - len(empty)
	if padLen < 0 {
		panic(fmt.Sprintf("echo document %d is below %d", len(empty), n))
	}
	body, err := json.Marshal(sceneEcho{Name: "Ada", Pad: strings.Repeat("x", padLen)})
	if err != nil {
		panic(err)
	}
	if len(body) != n {
		panic(fmt.Sprintf("echo body %d, want %d", len(body), n))
	}
	return body
}

func sceneGet(target string) []byte {
	return []byte("GET " + target + " HTTP/1.1\r\nHost: bench\r\nX-Request-ID: " + requestID + "\r\nConnection: keep-alive\r\n\r\n")
}

func scenePost(target, contentType string, body []byte) []byte {
	head := fmt.Sprintf("POST %s HTTP/1.1\r\nHost: bench\r\nX-Request-ID: %s\r\nContent-Type: %s\r\nContent-Length: %d\r\nConnection: keep-alive\r\n\r\n",
		target, requestID, contentType, len(body))
	return append([]byte(head), body...)
}

func registerZatranoScenes(r contracts.Router) {
	r.Get("/json", func(*khttp.Request) *khttp.Response {
		return khttp.JSON(sceneMessage{Message: hello})
	})
	r.Get("/users/{id}", func(req *khttp.Request) *khttp.Response {
		id, _ := strconv.Atoi(req.Route("id"))
		return khttp.JSON(sceneUser{ID: id, Name: "user"})
	})
	r.Post("/echo", func(req *khttp.Request) *khttp.Response {
		raw, err := req.Body()
		if err != nil {
			return khttp.JSON(map[string]string{"error": err.Error()})
		}
		var doc sceneEcho
		if err := json.Unmarshal(raw, &doc); err != nil {
			return khttp.JSON(map[string]string{"error": err.Error()})
		}
		return khttp.JSON(doc)
	})
	r.Get("/search", func(req *khttp.Request) *khttp.Response {
		return khttp.JSON(sceneSearch{Q: req.Query("q"), Page: req.Query("page")})
	})
}

func registerFiberScenes(app *fiber.App, probe captured) {
	app.Get("/json", func(c *fiber.Ctx) error {
		body, err := json.Marshal(sceneMessage{Message: hello})
		if err != nil {
			return err
		}
		return fiberScene(c, probe, "application/json", body)
	})
	app.Get("/users/:id", func(c *fiber.Ctx) error {
		id, _ := strconv.Atoi(c.Params("id"))
		body, err := json.Marshal(sceneUser{ID: id, Name: "user"})
		if err != nil {
			return err
		}
		return fiberScene(c, probe, "application/json", body)
	})
	app.Post("/echo", func(c *fiber.Ctx) error {
		var doc sceneEcho
		if err := c.BodyParser(&doc); err != nil {
			return err
		}
		body, err := json.Marshal(doc)
		if err != nil {
			return err
		}
		return fiberScene(c, probe, "application/json", body)
	})
	app.Get("/search", func(c *fiber.Ctx) error {
		body, err := json.Marshal(sceneSearch{Q: c.Query("q"), Page: c.Query("page")})
		if err != nil {
			return err
		}
		return fiberScene(c, probe, "application/json", body)
	})
}

func fiberScene(c *fiber.Ctx, probe captured, contentType string, body []byte) error {
	c.Set("Content-Type", contentType)
	if err := c.Send(body); err != nil {
		return err
	}
	id := c.Get("X-Request-ID")
	if idPattern.MatchString(id) {
		c.Set("X-Request-ID", id)
	}
	for name, value := range probe.headers {
		c.Set(name, value)
	}
	return nil
}
