package exceptions_test

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/kernel/exceptions"
	"github.com/zatrano/framework/v2/kernel/http"
)

func TestHTTPErrorMessagesAndUnwrap(t *testing.T) {
	nf := exceptions.NotFound()
	if nf.Error() != "Not Found" || nf.Status != 404 {
		t.Fatalf("%+v", nf)
	}
	custom := exceptions.NotFound("gone")
	if custom.Error() != "gone" {
		t.Fatal(custom.Error())
	}
	if exceptions.Forbidden().Error() != "Forbidden" {
		t.Fatal("forbidden default")
	}
	if exceptions.Forbidden("no").Error() != "no" {
		t.Fatal("forbidden custom")
	}
	if exceptions.Unauthorized().Error() != "Unauthenticated." {
		t.Fatal("unauthorized default")
	}
	if exceptions.Unauthorized("auth").Error() != "auth" {
		t.Fatal("unauthorized custom")
	}
	aborted := exceptions.Abort(418, "teapot")
	if aborted.Status != 418 || aborted.Error() != "teapot" {
		t.Fatalf("%+v", aborted)
	}
	cause := errors.New("root")
	wrapped := &exceptions.HTTPError{Status: 500, Cause: cause}
	if wrapped.Error() != "root" {
		t.Fatal(wrapped.Error())
	}
	if !errors.Is(wrapped, cause) {
		t.Fatal("unwrap")
	}
	bare := &exceptions.HTTPError{Status: 503}
	if bare.Error() != "HTTP 503" {
		t.Fatal(bare.Error())
	}
}

func TestReportAndRender(t *testing.T) {
	h := exceptions.New(false)
	h.Report(nil, nil)
	var seen error
	h.ReportUsing(func(err error, req *http.Request) { seen = err })
	h.Report(errors.New("logged"), nil)
	if seen == nil || seen.Error() != "logged" {
		t.Fatal(seen)
	}

	raw := httptest.NewRequest("GET", "/x", nil)
	req := http.NewRequest(raw)
	html := h.Render(req, exceptions.NotFound())
	if html == nil || html.StatusCode() != 404 || !strings.Contains(string(html.Content()), "Not Found") {
		t.Fatalf("html=%v", html)
	}
	if h.Render(req, nil) != nil {
		t.Fatal("nil err")
	}

	raw.Header.Set("Accept", "application/json")
	jsonReq := http.NewRequest(raw)
	js := h.Render(jsonReq, errors.New("boom"))
	if js.StatusCode() != 500 || !strings.Contains(string(js.Content()), "Server Error") {
		t.Fatalf("json prod=%s", js.Content())
	}

	debug := exceptions.New(true)
	debugJSON := debug.Render(jsonReq, errors.New("secret"))
	if !strings.Contains(string(debugJSON.Content()), "secret") {
		t.Fatalf("debug json=%s", debugJSON.Content())
	}
	htmlDebug := debug.Render(http.NewRequest(httptest.NewRequest("GET", "/", nil)), errors.New("stack"))
	if htmlDebug.StatusCode() != 500 || !strings.Contains(string(htmlDebug.Content()), "stack") {
		t.Fatal("debug html")
	}

	h.RenderUsing(419, func(req *http.Request, err error) *http.Response {
		return http.Text("expired").Status(419)
	})
	custom := h.Render(req, exceptions.Abort(419, "x"))
	if custom.StatusCode() != 419 || string(custom.Content()) != "expired" {
		t.Fatalf("%v %s", custom.StatusCode(), custom.Content())
	}

	for _, status := range []int{401, 403, 404, 419, 429, 503, 400, 500} {
		page := h.Render(req, exceptions.Abort(status, "m"))
		if page.StatusCode() != status {
			t.Fatalf("status %d -> %d", status, page.StatusCode())
		}
	}
}

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

func TestExceptionMiddlewareNonErrorPanic(t *testing.T) {
	h := exceptions.New(true)
	handler := h.Middleware()(func(req *http.Request) *http.Response {
		panic("plain")
	})
	resp := handler(http.NewRequest(httptest.NewRequest("GET", "/", nil)))
	if resp.StatusCode() != 500 || !strings.Contains(string(resp.Content()), "plain") {
		t.Fatalf("status=%d body=%s", resp.StatusCode(), resp.Content())
	}
}
