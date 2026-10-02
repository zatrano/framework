package http_test

import (
	"fmt"
	"testing"

	"github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/rawhttp"
)

func TestBodyOwnedCopyDoesNotMutateRawHTTPBuffer(t *testing.T) {
	t.Parallel()
	const payload = `{"n":1}`
	raw := fmt.Sprintf(
		"POST / HTTP/1.1\r\nHost: localhost\r\nContent-Type: application/json\r\nContent-Length: %d\r\nConnection: close\r\n\r\n%s",
		len(payload), payload,
	)
	status, body, err := http.ServeConnForTest(func(ctx *rawhttp.Ctx) {
		req := http.NewRequest(ctx)
		b, rerr := req.Body()
		if rerr != nil {
			t.Errorf("Body: %v", rerr)
			return
		}
		if string(b) != payload {
			t.Errorf("Body=%q want %q", b, payload)
			return
		}
		if len(b) == 0 {
			t.Fatal("empty body")
			return
		}
		b[0] = '!'
		if string(ctx.Body()) != payload {
			t.Errorf("mutated rawhttp buffer: %q", ctx.Body())
		}
		b2, _ := req.Body()
		if b2[0] != '!' {
			t.Errorf("cached owned copy should reflect caller mutation of returned slice: %q", b2)
		}
		ctx.SetStatusCode(200)
		ctx.SetBodyString("ok")
	}, raw)
	if err != nil {
		t.Fatal(err)
	}
	if status != 200 || string(body) != "ok" {
		t.Fatalf("status=%d body=%q", status, body)
	}
}
