package middleware_test

import (
	"bytes"
	"io"
	"mime/multipart"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zatrano/framework/kernel/http"
	"github.com/zatrano/framework/kernel/middleware"
)

type countingBody struct {
	r     io.Reader
	reads int
}

func (b *countingBody) Read(p []byte) (int, error) {
	b.reads++
	return b.r.Read(p)
}

func (b *countingBody) Close() error { return nil }

func applyPOST(t *testing.T, body io.Reader, contentType, header string) *http.Request {
	t.Helper()
	raw := httptest.NewRequest(stdhttp.MethodPost, "/", body)
	if contentType != "" {
		raw.Header.Set("Content-Type", contentType)
	}
	if header != "" {
		raw.Header.Set("X-HTTP-Method-Override", header)
	}
	req := http.NewRequest(raw)
	middleware.ApplyMethodOverride(req)
	return req
}

func TestMethodOverrideHeaderPATCH(t *testing.T) {
	req := applyPOST(t, nil, "", "PATCH")
	if req.Method() != stdhttp.MethodPatch {
		t.Fatalf("method=%q", req.Method())
	}
}

func TestMethodOverrideHeaderDELETE(t *testing.T) {
	req := applyPOST(t, nil, "", "DELETE")
	if req.Method() != stdhttp.MethodDelete {
		t.Fatalf("method=%q", req.Method())
	}
}

func TestMethodOverrideFormPATCH(t *testing.T) {
	req := applyPOST(t, strings.NewReader("_method=PATCH"), "application/x-www-form-urlencoded", "")
	if req.Method() != stdhttp.MethodPatch {
		t.Fatalf("method=%q", req.Method())
	}
}

func TestMethodOverrideHeaderBeatsForm(t *testing.T) {
	req := applyPOST(t, strings.NewReader("_method=DELETE"), "application/x-www-form-urlencoded", "PATCH")
	if req.Method() != stdhttp.MethodPatch {
		t.Fatalf("header must win: method=%q", req.Method())
	}
}

func TestMethodOverrideIgnoresQuery(t *testing.T) {
	raw := httptest.NewRequest(stdhttp.MethodPost, "/?_method=DELETE", nil)
	req := http.NewRequest(raw)
	middleware.ApplyMethodOverride(req)
	if req.Method() != stdhttp.MethodPost {
		t.Fatalf("query _method must be ignored: method=%q", req.Method())
	}
}

func TestMethodOverrideIgnoresJSON(t *testing.T) {
	req := applyPOST(t, strings.NewReader(`{"_method":"DELETE"}`), "application/json", "")
	if req.Method() != stdhttp.MethodPost {
		t.Fatalf("JSON _method must be ignored: method=%q", req.Method())
	}
}

func TestMethodOverrideIgnoresMultipart(t *testing.T) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if err := w.WriteField("_method", "DELETE"); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	req := applyPOST(t, &buf, w.FormDataContentType(), "")
	if req.Method() != stdhttp.MethodPost {
		t.Fatalf("multipart _method must be ignored: method=%q", req.Method())
	}
}

func TestMethodOverrideGETHeaderIgnored(t *testing.T) {
	raw := httptest.NewRequest(stdhttp.MethodGet, "/", nil)
	raw.Header.Set("X-HTTP-Method-Override", "DELETE")
	req := http.NewRequest(raw)
	middleware.ApplyMethodOverride(req)
	if req.Method() != stdhttp.MethodGet {
		t.Fatalf("method=%q", req.Method())
	}
}

func TestMethodOverridePUTHeaderIgnored(t *testing.T) {
	raw := httptest.NewRequest(stdhttp.MethodPut, "/", nil)
	raw.Header.Set("X-HTTP-Method-Override", "DELETE")
	req := http.NewRequest(raw)
	middleware.ApplyMethodOverride(req)
	if req.Method() != stdhttp.MethodPut {
		t.Fatalf("method=%q", req.Method())
	}
}

func TestMethodOverrideInvalidHeaderIgnored(t *testing.T) {
	req := applyPOST(t, strings.NewReader("_method=DELETE"), "application/x-www-form-urlencoded", "GET")
	if req.Method() != stdhttp.MethodPost {
		t.Fatalf("invalid header must not fall through to form: method=%q", req.Method())
	}
}

func TestMethodOverrideInvalidFormIgnored(t *testing.T) {
	req := applyPOST(t, strings.NewReader("_method=GET"), "application/x-www-form-urlencoded", "")
	if req.Method() != stdhttp.MethodPost {
		t.Fatalf("method=%q", req.Method())
	}
}

func TestMethodOverrideHeaderDoesNotReadBody(t *testing.T) {
	body := &countingBody{r: strings.NewReader(`{"_method":"DELETE"}`)}
	raw := httptest.NewRequest(stdhttp.MethodPost, "/", body)
	raw.Header.Set("Content-Type", "application/json")
	raw.Header.Set("X-HTTP-Method-Override", "PATCH")
	req := http.NewRequest(raw)
	middleware.ApplyMethodOverride(req)
	if req.Method() != stdhttp.MethodPatch {
		t.Fatalf("method=%q", req.Method())
	}
	if body.reads != 0 {
		t.Fatalf("header override must not read body: reads=%d", body.reads)
	}
}

func TestMethodOverrideJSONDoesNotParseBody(t *testing.T) {
	body := &countingBody{r: strings.NewReader(`{"_method":"DELETE"}`)}
	raw := httptest.NewRequest(stdhttp.MethodPost, "/", body)
	raw.Header.Set("Content-Type", "application/json")
	req := http.NewRequest(raw)
	middleware.ApplyMethodOverride(req)
	if req.Method() != stdhttp.MethodPost {
		t.Fatalf("method=%q", req.Method())
	}
	if body.reads != 0 {
		t.Fatalf("JSON POST must not be parsed for _method: reads=%d", body.reads)
	}
}

func TestMethodOverrideMultipartDoesNotParseBody(t *testing.T) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if err := w.WriteField("_method", "DELETE"); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	body := &countingBody{r: bytes.NewReader(buf.Bytes())}
	raw := httptest.NewRequest(stdhttp.MethodPost, "/", body)
	raw.Header.Set("Content-Type", w.FormDataContentType())
	req := http.NewRequest(raw)
	middleware.ApplyMethodOverride(req)
	if req.Method() != stdhttp.MethodPost {
		t.Fatalf("method=%q", req.Method())
	}
	if body.reads != 0 {
		t.Fatalf("multipart POST must not be parsed for _method: reads=%d", body.reads)
	}
}

func TestMethodOverrideUnsupportedContentTypeDoesNotParseBody(t *testing.T) {
	body := &countingBody{r: strings.NewReader("_method=DELETE")}
	raw := httptest.NewRequest(stdhttp.MethodPost, "/", body)
	raw.Header.Set("Content-Type", "text/plain")
	req := http.NewRequest(raw)
	middleware.ApplyMethodOverride(req)
	if req.Method() != stdhttp.MethodPost {
		t.Fatalf("method=%q", req.Method())
	}
	if body.reads != 0 {
		t.Fatalf("unsupported content type must not parse body: reads=%d", body.reads)
	}
}

func TestMethodOverrideFormWithCharset(t *testing.T) {
	req := applyPOST(t, strings.NewReader("_method=PUT"), "application/x-www-form-urlencoded; charset=UTF-8", "")
	if req.Method() != stdhttp.MethodPut {
		t.Fatalf("method=%q", req.Method())
	}
}
