package routing_test

import (
	"testing"

	"github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/framework/v3/core/kernel/routing"
	"github.com/zatrano/rawhttp"
)

func TestRouteParamAliasesCarrierBuffer(t *testing.T) {
	r := routing.New()
	var got string
	r.Get("/users/{id}", func(req *http.Request) *http.Response {
		got = req.Route("id")
		return http.Text("ok")
	})
	if err := r.Freeze(); err != nil {
		t.Fatal(err)
	}
	path := []byte("/users/42")
	ctx := &rawhttp.Ctx{Method: []byte("GET"), Path: path}
	resp := r.Dispatch(http.NewRequest(ctx))
	if resp == nil || resp.StatusCode() != 200 {
		t.Fatalf("status=%v", resp)
	}
	if got != "42" {
		t.Fatalf("param=%q", got)
	}
	path[7] = '9'
	if got != "92" {
		t.Fatalf("parameter was copied out of the carrier buffer: %q", got)
	}
	for i := range path {
		path[i] = 0
	}
	if got == "92" || got == "42" {
		t.Fatal("zeroing the carrier buffer left the parameter readable")
	}
}

func BenchmarkRouteParam(b *testing.B) {
	http.SetRequestPooling(true, false)
	b.Cleanup(func() { http.SetRequestPooling(true, false) })
	r := routing.New()
	r.Get("/users/{id}", func(req *http.Request) *http.Response {
		return http.Text(req.Route("id"))
	})
	if err := r.Freeze(); err != nil {
		b.Fatal(err)
	}
	path := []byte("/users/42")
	ctx := &rawhttp.Ctx{Method: []byte("GET"), Path: path}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		req := http.AcquireRequest(ctx)
		resp := r.Dispatch(req)
		if resp == nil {
			b.Fatal("nil response")
		}
		http.ReleaseResponse(resp)
		http.ReleaseRequest(req)
	}
}
