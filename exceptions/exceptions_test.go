package exceptions_test

import (
	"net/http/httptest"
	"testing"

	"github.com/zatrano/framework/exceptions"
	"github.com/zatrano/framework/http"
)

func TestExceptionMiddlewareRendersHTTPError(t *testing.T) {
	h := exceptions.New(true)
	handler := h.Middleware()(func(req *http.Request) *http.Response {
		panic(exceptions.NotFound("missing"))
	})
	raw := httptest.NewRequest("GET", "/api/x", nil)
	raw.Header.Set("Accept", "application/json")
	resp := handler(http.NewRequest(raw))
	if resp.StatusCode() != 404 {
		t.Fatalf("status=%d body=%s", resp.StatusCode(), resp.Content())
	}
}
