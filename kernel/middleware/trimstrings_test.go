package middleware_test

import (
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/kernel/http"
	"github.com/zatrano/framework/v2/kernel/middleware"
)

func TestTrimAndEmptyToNull(t *testing.T) {
	raw := httptest.NewRequest(stdhttp.MethodPost, "/", strings.NewReader("name=%20Ada%20&note=&keep=  x  "))
	raw.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req := http.NewRequest(raw)

	handler := middleware.TrimStrings()(middleware.ConvertEmptyStringsToNull("keep")(func(r *http.Request) *http.Response {
		all := r.All()
		if all["name"] != "Ada" {
			t.Fatalf("name=%q", all["name"])
		}
		if _, ok := all["note"]; ok {
			t.Fatalf("note should be removed, got %#v", all)
		}
		if all["keep"] != "x" {
			t.Fatalf("keep=%q", all["keep"])
		}
		return http.Text("ok")
	}))
	_ = handler(req)
}
