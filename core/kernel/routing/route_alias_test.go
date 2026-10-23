package routing_test

import (
	"testing"

	"github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/framework/v3/core/kernel/routing"
	"github.com/zatrano/rawhttp"
)

// Route parameters are views of the request's own copy of the path, so
// rawhttp reusing its read buffer for the next request cannot change them.
func TestRouteParamSurvivesCarrierBufferReuse(t *testing.T) {
	r := routing.New()
	var got string
	var kept *http.Request
	r.Get("/users/{id}", func(req *http.Request) *http.Response {
		got = req.Route("id")
		kept = req
		return http.Text("ok")
	})
	if err := r.Freeze(); err != nil {
		t.Fatal(err)
	}
	path := []byte("/users/42")
	ctx := &rawhttp.Ctx{Method: []byte("GET"), Path: path}
	req := http.NewRequest(ctx)
	resp := r.Dispatch(req)
	req.Detach()
	if resp == nil || resp.StatusCode() != 200 {
		t.Fatalf("status=%v", resp)
	}
	copy(path, "/users/99")
	for i := range ctx.Method {
		ctx.Method[i] = 'X'
	}
	if got != "42" || kept.Route("id") != "42" {
		t.Fatalf("parameter followed the carrier buffer: got=%q kept=%q", got, kept.Route("id"))
	}
	if kept.Path() != "/users/42" || kept.Method() != "GET" {
		t.Fatalf("retained request followed the carrier buffer: %s %s", kept.Method(), kept.Path())
	}
	if kept.Ctx() != nil {
		t.Fatal("Ctx is still reachable after Detach")
	}
}

func BenchmarkRouteParam(b *testing.B) {
	r := routing.New()
	r.Get("/users/{id}", func(req *http.Request) *http.Response {
		return http.Text(req.Route("id"))
	})
	if err := r.Freeze(); err != nil {
		b.Fatal(err)
	}
	ctx := &rawhttp.Ctx{Method: []byte("GET"), Path: []byte("/users/42")}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if r.Dispatch(http.NewRequest(ctx)) == nil {
			b.Fatal("nil response")
		}
	}
}
