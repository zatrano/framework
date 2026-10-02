package http

import (
	"strings"

	"github.com/zatrano/rawhttp"
)

// testCtx builds a minimal rawhttp.Ctx for unit tests.
func testCtx(method, path string) *rawhttp.Ctx {
	if method == "" {
		method = "GET"
	}
	query := ""
	if i := strings.IndexByte(path, '?'); i >= 0 {
		query = path[i+1:]
		path = path[:i]
	}
	if path == "" {
		path = "/"
	}
	return &rawhttp.Ctx{
		Method: []byte(method),
		Path:   []byte(path),
		Query:  []byte(query),
	}
}

// testRequest builds a Request for tests with optional body overlay.
// path may include a query string (?a=1).
func testRequest(method, path string, body []byte) *Request {
	req := NewRequest(testCtx(method, path))
	if len(body) > 0 {
		req.SetBody(body)
	}
	return req
}
