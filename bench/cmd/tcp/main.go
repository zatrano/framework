// Command frameworkbench serves one framework or drives a sequential comparison.
//
//	go run . -mode all
//	go run . -mode serve -framework gin -addr 127.0.0.1:18102
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/beego/beego/v2/server/web"
	"github.com/gin-gonic/gin"
	"github.com/gofiber/fiber/v2"
	"github.com/kataras/iris/v12"
	"github.com/labstack/echo/v4"
	"github.com/revel/revel"
	"github.com/zatrano/framework/v3/core/contracts"
	"github.com/zatrano/framework/v3/core/kernel"
	zhttp "github.com/zatrano/framework/v3/core/kernel/http"
)

const hello = "Hello, World!"

func main() {
	mode := flag.String("mode", "all", "all or serve")
	framework := flag.String("framework", "zatrano", "zatrano|gin|fiber|echo|beego|iris|revel")
	addr := flag.String("addr", "127.0.0.1:18101", "listen address")
	flag.Parse()

	if *mode == "serve" {
		if err := serve(*framework, *addr); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if err := runAll(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func serve(name, addr string) error {
	switch name {
	case "zatrano":
		return serveZatrano(addr)
	case "gin":
		return serveGin(addr)
	case "fiber":
		return serveFiber(addr)
	case "echo":
		return serveEcho(addr)
	case "beego":
		return serveBeego(addr)
	case "iris":
		return serveIris(addr)
	case "revel":
		return serveRevel(addr)
	default:
		return fmt.Errorf("unknown framework %s", name)
	}
}

type routeProvider struct{}

func (routeProvider) Register(app contracts.App) error {
	app.Router().Get("/plaintext", func(*zhttp.Request) *zhttp.Response {
		return zhttp.Text(hello)
	})
	app.Router().Get("/json", func(*zhttp.Request) *zhttp.Response {
		return zhttp.JSON(map[string]string{"message": hello})
	})
	app.Router().Get("/users/{id}", func(req *zhttp.Request) *zhttp.Response {
		return zhttp.JSON(map[string]string{"id": req.Route("id")})
	})
	app.Router().Post("/echo", func(req *zhttp.Request) *zhttp.Response {
		raw, err := req.Body()
		if err != nil {
			return zhttp.JSON(map[string]string{"error": err.Error()})
		}
		var v map[string]any
		if err := json.Unmarshal(raw, &v); err != nil {
			return zhttp.JSON(map[string]string{"error": err.Error()})
		}
		return zhttp.JSON(v)
	})
	return nil
}

func (routeProvider) Boot(contracts.App) error { return nil }

func serveZatrano(addr string) error {
	dir, err := os.MkdirTemp("", "zatrano-bench")
	if err != nil {
		return err
	}
	app := kernel.NewApplication(dir)
	app.RegisterProviders(routeProvider{})
	return app.Run(addr)
}

func serveGin(addr string) error {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.GET("/plaintext", func(c *gin.Context) { c.String(http.StatusOK, hello) })
	r.GET("/json", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": hello}) })
	r.GET("/users/:id", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"id": c.Param("id")}) })
	r.POST("/echo", func(c *gin.Context) {
		var v map[string]any
		if err := c.BindJSON(&v); err != nil {
			return
		}
		c.JSON(http.StatusOK, v)
	})
	return r.Run(addr)
}

func serveFiber(addr string) error {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/plaintext", func(c *fiber.Ctx) error { return c.SendString(hello) })
	app.Get("/json", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"message": hello}) })
	app.Get("/users/:id", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"id": c.Params("id")}) })
	app.Post("/echo", func(c *fiber.Ctx) error {
		var v map[string]any
		if err := c.BodyParser(&v); err != nil {
			return err
		}
		return c.JSON(v)
	})
	return app.Listen(addr)
}

func serveEcho(addr string) error {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.GET("/plaintext", func(c echo.Context) error { return c.String(http.StatusOK, hello) })
	e.GET("/json", func(c echo.Context) error { return c.JSON(http.StatusOK, map[string]string{"message": hello}) })
	e.GET("/users/:id", func(c echo.Context) error { return c.JSON(http.StatusOK, map[string]string{"id": c.Param("id")}) })
	e.POST("/echo", func(c echo.Context) error {
		var v map[string]any
		if err := c.Bind(&v); err != nil {
			return err
		}
		return c.JSON(http.StatusOK, v)
	})
	return e.Start(addr)
}

type beePlain struct{ web.Controller }
type beeJSON struct{ web.Controller }
type beeUser struct{ web.Controller }
type beeEcho struct{ web.Controller }

func (c *beePlain) Get() { c.Ctx.WriteString(hello) }
func (c *beeJSON) Get()  { c.Data["json"] = map[string]string{"message": hello}; c.ServeJSON() }
func (c *beeUser) Get() {
	c.Data["json"] = map[string]string{"id": c.Ctx.Input.Param(":id")}
	c.ServeJSON()
}
func (c *beeEcho) Post() {
	var v map[string]any
	_ = json.Unmarshal(c.Ctx.Input.RequestBody, &v)
	c.Data["json"] = v
	c.ServeJSON()
}

func serveBeego(addr string) error {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return err
	}
	web.BConfig.RunMode = "prod"
	web.BConfig.Listen.EnableAdmin = false
	web.BConfig.Listen.HTTPAddr = host
	web.BConfig.Listen.HTTPPort = port
	web.BConfig.CopyRequestBody = true
	web.BConfig.Log.AccessLogs = false
	web.Router("/plaintext", &beePlain{})
	web.Router("/json", &beeJSON{})
	web.Router("/users/:id", &beeUser{})
	web.Router("/echo", &beeEcho{})
	web.Run()
	return nil
}

func serveIris(addr string) error {
	app := iris.New()
	app.Logger().SetLevel("error")
	app.Get("/plaintext", func(ctx iris.Context) { ctx.Text(hello) })
	app.Get("/json", func(ctx iris.Context) { ctx.JSON(iris.Map{"message": hello}) })
	app.Get("/users/{id}", func(ctx iris.Context) { ctx.JSON(iris.Map{"id": ctx.Params().Get("id")}) })
	app.Post("/echo", func(ctx iris.Context) {
		var v map[string]any
		_ = ctx.ReadJSON(&v)
		ctx.JSON(v)
	})
	return app.Listen(addr, iris.WithoutStartupLog, iris.WithoutBanner, iris.WithoutInterruptHandler)
}

func serveRevel(addr string) error {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "revel-bench")
	if err != nil {
		return err
	}
	base := filepath.Join(dir, "frameworkbench")
	if err := os.MkdirAll(filepath.Join(base, "conf"), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(base, "app"), 0o755); err != nil {
		return err
	}
	conf := fmt.Sprintf(`[prod]
app.name = bench
http.addr = %s
http.port = %s
watch = false
watch.templates = false
mode.dev = false
revel.controller.reuse = false
log.info.output = off
log.debug.output = off
log.warn.output = off
log.request.output = off
log.error.output = stderr
`, host, portStr)
	if err := os.WriteFile(filepath.Join(base, "conf", "app.conf"), []byte(conf), 0o644); err != nil {
		return err
	}
	// Revel loads conf/routes during startup even when Filters is replaced.
	if err := os.WriteFile(filepath.Join(base, "conf", "routes"), []byte("# bench routes are handled in-process\n"), 0o644); err != nil {
		return err
	}
	revel.Init("prod", "frameworkbench", dir)
	revel.Filters = []revel.Filter{revelBench}
	port, _ := strconv.Atoi(portStr)
	revel.Run(port)
	return nil
}

func revelBench(c *revel.Controller, _ []revel.Filter) {
	path := c.Request.GetPath()
	switch path {
	case "/plaintext":
		c.Response.Status = http.StatusOK
		c.Response.ContentType = "text/plain"
		_, _ = c.Response.Out.Write([]byte(hello))
	case "/json":
		c.Response.Status = http.StatusOK
		c.Response.ContentType = "application/json"
		b, _ := json.Marshal(map[string]string{"message": hello})
		_, _ = c.Response.Out.Write(b)
	default:
		if strings.HasPrefix(path, "/users/") {
			id := strings.TrimPrefix(path, "/users/")
			c.Response.Status = http.StatusOK
			c.Response.ContentType = "application/json"
			b, _ := json.Marshal(map[string]string{"id": id})
			_, _ = c.Response.Out.Write(b)
			return
		}
		if path == "/echo" && c.Request.Method == http.MethodPost {
			raw, _ := io.ReadAll(c.Request.GetBody())
			c.Response.Status = http.StatusOK
			c.Response.ContentType = "application/json"
			_, _ = c.Response.Out.Write(raw)
			return
		}
		c.Response.Status = http.StatusNotFound
	}
}

type sample struct {
	Framework string  `json:"framework"`
	Route     string  `json:"route"`
	Run       int     `json:"run"`
	RPS       float64 `json:"rps"`
	P50       string  `json:"p50"`
	P99       string  `json:"p99"`
	Errors    int     `json:"errors"`
	Requests  int     `json:"requests"`
}

func runAll() error {
	names := []string{"zatrano", "gin", "fiber", "echo", "beego", "iris", "revel"}
	routes := []struct {
		name, method, path string
		body               []byte
	}{
		{"GET /plaintext", http.MethodGet, "/plaintext", nil},
		{"GET /json", http.MethodGet, "/json", nil},
		{"GET /users/{id}", http.MethodGet, "/users/42", nil},
		{"POST /echo", http.MethodPost, "/echo", []byte(`{"message":"Hello, World!"}`)},
	}
	var out []sample
	port := 18101
	for _, name := range names {
		addr := "127.0.0.1:" + strconv.Itoa(port)
		port++
		cmd := exec.Command(os.Args[0], "-mode", "serve", "-framework", name, "-addr", addr)
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		cmd.Env = append(os.Environ(),
			"APP_ENV=production",
			"APP_DEBUG=false",
			"APP_KEY=bench-key-0123456789abcdef012345",
			"LOG_LEVEL=error",
			"CORS_ENABLED=true",
		)
		if err := cmd.Start(); err != nil {
			return err
		}
		if err := waitReady(addr, 30*time.Second); err != nil {
			_ = cmd.Process.Kill()
			return fmt.Errorf("%s: %w", name, err)
		}
		for _, rt := range routes {
			url := "http://" + addr + rt.path
			if err := probe(url, rt.method, rt.body); err != nil {
				_ = cmd.Process.Kill()
				return fmt.Errorf("%s %s: %w", name, rt.name, err)
			}
			for run := 1; run <= 3; run++ {
				// Short bursts with a rest. A 2-core laptop throttles if the
				// load runs for minutes, and that penalty lands on whoever is last.
				time.Sleep(8 * time.Second)
				s := measure(url, rt.method, rt.body, 64, 400*time.Millisecond, 2*time.Second)
				s.Framework = name
				s.Route = rt.name
				s.Run = run
				out = append(out, s)
				fmt.Printf("%s\t%s\trun %d\trps %.0f\tp50 %s\tp99 %s\terr %d\tn %d\n",
					name, rt.name, run, s.RPS, s.P50, s.P99, s.Errors, s.Requests)
			}
		}
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		time.Sleep(300 * time.Millisecond)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func waitReady(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s", addr)
}

func probe(url, method string, body []byte) error {
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d body %s", resp.StatusCode, raw)
	}
	return nil
}

func measure(url, method string, body []byte, conns int, warm, dur time.Duration) sample {
	tr := &http.Transport{
		MaxIdleConns:        conns,
		MaxIdleConnsPerHost: conns,
		MaxConnsPerHost:     conns,
		DisableCompression:  true,
		IdleConnTimeout:     30 * time.Second,
	}
	client := &http.Client{Transport: tr, Timeout: 5 * time.Second}
	ctx, cancel := context.WithTimeout(context.Background(), warm+dur)
	defer cancel()
	start := time.Now()
	measureAt := start.Add(warm)
	var mu sync.Mutex
	var lat []time.Duration
	var errors, requests int
	var wg sync.WaitGroup
	for i := 0; i < conns; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				if ctx.Err() != nil {
					return
				}
				req, err := http.NewRequest(method, url, bytes.NewReader(body))
				if err != nil {
					return
				}
				if body != nil {
					req.Header.Set("Content-Type", "application/json")
				}
				t0 := time.Now()
				resp, err := client.Do(req)
				el := time.Since(t0)
				if time.Now().Before(measureAt) {
					if resp != nil {
						io.Copy(io.Discard, resp.Body)
						resp.Body.Close()
					}
					continue
				}
				mu.Lock()
				if err != nil || resp.StatusCode != http.StatusOK {
					errors++
				} else {
					requests++
					lat = append(lat, el)
				}
				mu.Unlock()
				if resp != nil {
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
				}
			}
		}()
	}
	wg.Wait()
	sort.Slice(lat, func(i, j int) bool { return lat[i] < lat[j] })
	s := sample{Errors: errors, Requests: requests}
	if dur > 0 {
		s.RPS = float64(requests) / dur.Seconds()
	}
	if n := len(lat); n > 0 {
		s.P50 = lat[n*50/100].String()
		s.P99 = lat[n*99/100].String()
	}
	return s
}
