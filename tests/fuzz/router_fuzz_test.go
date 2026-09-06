// Package fuzz_test holds long-running Go fuzz targets for security-sensitive parsers.
//
// CI runs each Fuzz* with -fuzztime=3m; locally you may use longer, e.g.:
//
//	go test ./tests/fuzz/ -run=^$ -fuzz=FuzzRouterPath -fuzztime=30m
//	go test ./tests/fuzz/ -run=^$ -fuzz=FuzzDotenvParse -fuzztime=30m
package fuzz_test

import (
	stdhttp "net/http"
	"testing"

	"github.com/zatrano/framework/v2/kernel/http"
	"github.com/zatrano/framework/v2/kernel/routing"
)

// FuzzRouterPath exercises route compilation and dispatch on arbitrary paths.
func FuzzRouterPath(f *testing.F) {
	seeds := []string{
		"/",
		"/users/{id}",
		"/posts/{slug?}",
		"/files/{*path}",
		"/a/b/c",
		"//",
		"/users/{id}/edit",
		"/{x}/{y}",
	}
	for _, s := range seeds {
		f.Add(s, "/users/1")
	}
	f.Fuzz(func(t *testing.T, routePath, requestPath string) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("router panic route=%q req=%q: %v", routePath, requestPath, r)
			}
		}()
		if len(routePath) > 512 || len(requestPath) > 2048 {
			return
		}
		r := routing.New()
		r.Add(stdhttp.MethodGet, routePath, func(req *http.Request) *http.Response {
			return http.Text("ok")
		})
		raw, err := stdhttp.NewRequest(stdhttp.MethodGet, "http://fuzz.example"+normalizeFuzzPath(requestPath), nil)
		if err != nil {
			return
		}
		_ = r.Dispatch(http.NewRequest(raw))
	})
}

func normalizeFuzzPath(p string) string {
	if p == "" || p[0] != '/' {
		return "/" + p
	}
	return p
}
