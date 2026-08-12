package httpclient_test

import (
	stdhttp "net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/zatrano/framework/packages/httpclient"
)

func TestHTTPClientRetry(t *testing.T) {
	var hits atomic.Int64
	server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		n := hits.Add(1)
		if n < 3 {
			w.WriteHeader(stdhttp.StatusServiceUnavailable)
			_, _ = w.Write([]byte("busy"))
			return
		}
		w.WriteHeader(stdhttp.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := httpclient.New()
	resp, err := client.AsJSON().RetryTimes(3).Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if !resp.OK() {
		t.Fatalf("status=%d body=%s hits=%d", resp.StatusCode, resp.String(), hits.Load())
	}
	if hits.Load() != 3 {
		t.Fatalf("expected 3 attempts, got %d", hits.Load())
	}
}
