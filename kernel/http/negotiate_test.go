package http_test

import (
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/zatrano/framework/v2/kernel/http"
)

func TestNegotiateJSONHTMLTextAndWildcards(t *testing.T) {
	req := func(accept string) *http.Request {
		raw := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
		if accept != "" {
			raw.Header.Set("Accept", accept)
		}
		return http.NewRequest(raw)
	}

	if got := http.Negotiate(req("text/html,application/json;q=0.9"), http.FormatJSON, http.FormatHTML); got != http.FormatHTML {
		t.Fatalf("html first in Accept: %s", got)
	}
	if got := http.Negotiate(req("application/json"), http.FormatJSON, http.FormatHTML); got != http.FormatJSON {
		t.Fatalf("json: %s", got)
	}
	if got := http.Negotiate(req("text/plain"), http.FormatJSON, http.FormatText); got != http.FormatText {
		t.Fatalf("text: %s", got)
	}
	if got := http.Negotiate(req("application/vnd.api+json"), http.FormatJSON, http.FormatHTML); got != http.FormatJSON {
		t.Fatalf("+json: %s", got)
	}
	if got := http.Negotiate(req("*/*"), http.FormatHTML, http.FormatJSON); got != http.FormatHTML {
		t.Fatalf("wildcard: %s", got)
	}
	if got := http.Negotiate(req(""), http.FormatJSON, http.FormatHTML); got != http.FormatJSON {
		t.Fatalf("empty Accept: %s", got)
	}
	if got := http.Negotiate(req("application/xml"), http.FormatJSON, http.FormatHTML); got != http.FormatJSON {
		t.Fatalf("no match falls back: %s", got)
	}
	if got := http.Negotiate(req("not a valid token ;;;"), http.FormatHTML); got != http.FormatHTML {
		t.Fatalf("malformed Accept: %s", got)
	}
	if got := http.Negotiate(nil); got != http.FormatJSON {
		t.Fatalf("nil request: %s", got)
	}
	if got := http.Negotiate(req("application/json;q=0.9, text/html"), http.FormatJSON, http.FormatHTML); got != http.FormatHTML {
		t.Fatalf("higher q wins: %s", got)
	}
	if got := http.Negotiate(req("application/json;q=0, text/html"), http.FormatJSON, http.FormatHTML); got != http.FormatHTML {
		t.Fatalf("q=0 json is not acceptable: %s", got)
	}
	if got := http.Negotiate(req("application/json;q=0"), http.FormatJSON, http.FormatHTML); got != http.FormatJSON {
		t.Fatalf("q=0 only still falls back: %s", got)
	}
	if got := http.Negotiate(req("application/*"), http.FormatJSON, http.FormatHTML); got != http.FormatJSON {
		t.Fatalf("application/*: %s", got)
	}
	if got := http.Negotiate(req("text/html;q=1, application/json;q=0.9"), http.FormatJSON, http.FormatHTML); got != http.FormatHTML {
		t.Fatalf("q=1 html: %s", got)
	}
	if got := http.Negotiate(req("application/json;q=bogus"), http.FormatJSON, http.FormatHTML); got != http.FormatJSON {
		t.Fatalf("malformed q defaults to 1: %s", got)
	}
}

func TestWantsJSONUsesAcceptParser(t *testing.T) {
	req := func(accept string) *http.Request {
		raw := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
		if accept != "" {
			raw.Header.Set("Accept", accept)
		}
		return http.NewRequest(raw)
	}
	if req("").WantsJSON() {
		t.Fatal("empty Accept is not JSON-specific")
	}
	if req("*/*").WantsJSON() {
		t.Fatal("*/* is not JSON-specific")
	}
	if req("application/json;q=0").WantsJSON() {
		t.Fatal("q=0 json is not wanted")
	}
	if !req("application/json;q=0.9").WantsJSON() {
		t.Fatal("json with q>0")
	}
	jsonBody := httptest.NewRequest(stdhttp.MethodPost, "/", nil)
	jsonBody.Header.Set("Content-Type", "application/json")
	if !http.NewRequest(jsonBody).WantsJSON() {
		t.Fatal("JSON body")
	}
}

func TestNegotiatedFormatAttribute(t *testing.T) {
	req := http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/", nil))
	if http.NegotiatedFormat(req) != "" || http.WantsFormat(req, http.FormatJSON) {
		t.Fatal("expected empty until set")
	}
	req.Set(http.AttrNegotiatedFormat, http.FormatJSON)
	if !http.WantsFormat(req, "JSON") || http.NegotiatedFormat(req) != http.FormatJSON {
		t.Fatal("stored format")
	}
}
