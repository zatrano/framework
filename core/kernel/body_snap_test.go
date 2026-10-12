package kernel

import (
	"fmt"
	"testing"

	"github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/rawhttp"
)

func TestRequestBodyUsesBootLimitNotLiveEnv(t *testing.T) {
	t.Setenv("MAX_BODY_BYTES", "1048576")
	app := NewApplication(t.TempDir())
	t.Cleanup(func() {
		if c, ok := app.Logger().(interface{ Close() error }); ok && c != nil {
			_ = c.Close()
		}
	})
	const payload = `{"message":"Hello, World!"}`
	app.Router().Post("/in", func(req *http.Request) *http.Response {
		b, err := req.Body()
		if err != nil {
			return http.Text("limited").Status(400)
		}
		return http.Text(string(b))
	})
	if err := app.Bootstrap(); err != nil {
		t.Fatal(err)
	}
	// Smaller than the payload. A live env read would reject the body.
	// The boot snapshot is 1 MiB, so Handle must still accept it.
	t.Setenv("MAX_BODY_BYTES", "4")
	raw := fmt.Sprintf("POST /in HTTP/1.1\r\nHost: localhost\r\nContent-Type: application/json\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s", len(payload), payload)
	er, err := http.ExchangeForTest(func(ctx *rawhttp.Ctx) { app.Handle(ctx) }, raw)
	if err != nil {
		t.Fatal(err)
	}
	if er.Status != 200 || string(er.Body) != payload {
		t.Fatalf("status=%d body=%q raw=%q", er.Status, er.Body, er.Raw)
	}
}
