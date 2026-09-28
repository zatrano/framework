package http

import (
	"bytes"
	"io"

	"github.com/zatrano/rawhttp"
)

// testCtx builds a minimal rawhttp.Ctx for unit tests.
func testCtx(method, path string) *rawhttp.Ctx {
	if method == "" {
		method = "GET"
	}
	if path == "" {
		path = "/"
	}
	return &rawhttp.Ctx{
		Method: []byte(method),
		Path:   []byte(path),
	}
}

// testRequest builds a Request for tests, attaching body on the synthetic Raw().
func testRequest(method, path string, body []byte) *Request {
	req := NewRequest(testCtx(method, path))
	if req.raw != nil && len(body) > 0 {
		req.raw.Body = io.NopCloser(bytes.NewReader(body))
		req.bodyRead = false
		req.bodyCached = nil
	}
	return req
}
