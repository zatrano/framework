package middleware_test

import (
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/zatrano/framework/v2/kernel/http"
	"github.com/zatrano/framework/v2/kernel/middleware"
	"github.com/zatrano/framework/v2/kernel/routing"
)

func TestNegotiateMiddleware(t *testing.T) {
	r := routing.New()
	r.Use(middleware.Negotiate(http.FormatJSON, http.FormatHTML))
	r.Get("/item", func(req *http.Request) *http.Response {
		if !http.WantsFormat(req, http.FormatHTML) {
			t.Errorf("format=%q", http.NegotiatedFormat(req))
		}
		return http.Text(http.NegotiatedFormat(req))
	})
	if err := r.Freeze(); err != nil {
		t.Fatal(err)
	}

	raw := httptest.NewRequest(stdhttp.MethodGet, "/item", nil)
	raw.Header.Set("Accept", "text/html")
	resp := r.Dispatch(http.NewRequest(raw))
	if resp.StatusCode() != 200 || string(resp.Content()) != http.FormatHTML {
		t.Fatalf("status=%d body=%s", resp.StatusCode(), resp.Content())
	}
	if resp.GetHeader("Vary") != "Accept" || resp.GetHeader("X-Negotiated-Format") != http.FormatHTML {
		t.Fatalf("headers vary=%q format=%q", resp.GetHeader("Vary"), resp.GetHeader("X-Negotiated-Format"))
	}
}

func TestNegotiateMiddlewareFallbackAndUnsupported(t *testing.T) {
	r := routing.New()
	r.Use(middleware.Negotiate(http.FormatJSON, http.FormatHTML))
	r.Get("/item", func(req *http.Request) *http.Response {
		return http.Text(http.NegotiatedFormat(req))
	})
	if err := r.Freeze(); err != nil {
		t.Fatal(err)
	}

	empty := httptest.NewRequest(stdhttp.MethodGet, "/item", nil)
	resp := r.Dispatch(http.NewRequest(empty))
	if string(resp.Content()) != http.FormatJSON {
		t.Fatalf("empty Accept fallback=%s", resp.Content())
	}

	xml := httptest.NewRequest(stdhttp.MethodGet, "/item", nil)
	xml.Header.Set("Accept", "application/xml")
	resp = r.Dispatch(http.NewRequest(xml))
	if string(resp.Content()) != http.FormatJSON {
		t.Fatalf("unsupported fallback=%s", resp.Content())
	}

	ranked := httptest.NewRequest(stdhttp.MethodGet, "/item", nil)
	ranked.Header.Set("Accept", "application/json;q=0.2, text/html")
	resp = r.Dispatch(http.NewRequest(ranked))
	if string(resp.Content()) != http.FormatHTML {
		t.Fatalf("q-rank=%s", resp.Content())
	}
}
