package http_test

import (
	"strings"
	"testing"

	"github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/rawhttp"
)

func TestStreamBodyChunkedCommit(t *testing.T) {
	t.Parallel()
	payload := strings.Repeat("chunk-", 8)
	er, err := http.ExchangeForTest(func(ctx *rawhttp.Ctx) {
		resp := http.StreamBody("text/plain", strings.NewReader(payload), -1)
		_ = resp.Commit(ctx)
	}, "GET / HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n")
	if err != nil {
		t.Fatal(err)
	}
	if er.Status != 200 {
		t.Fatalf("status=%d", er.Status)
	}
	te := strings.ToLower(er.Header.Get("Transfer-Encoding"))
	if !strings.Contains(te, "chunked") {
		t.Fatalf("Transfer-Encoding=%q want chunked; raw=%q", te, er.Raw)
	}
	if !strings.Contains(string(er.Raw), payload) {
		t.Fatalf("raw missing payload: %q", er.Raw)
	}
}

func TestStreamBodySizedCommit(t *testing.T) {
	t.Parallel()
	payload := "hello-stream"
	status, body, err := http.ServeConnForTest(func(ctx *rawhttp.Ctx) {
		resp := http.StreamBody("text/plain", strings.NewReader(payload), len(payload))
		_ = resp.Commit(ctx)
	}, "GET / HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n")
	if err != nil {
		t.Fatal(err)
	}
	if status != 200 || string(body) != payload {
		t.Fatalf("status=%d body=%q", status, body)
	}
}
