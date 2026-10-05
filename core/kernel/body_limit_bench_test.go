package kernel

import (
	"strconv"
	"testing"

	"github.com/zatrano/framework/v3/core/kernel/http"
)

func BenchmarkBodyLimitNeedsRouteGET(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if bodyLimitNeedsRoute("GET", 0, 2<<20) {
			b.Fatal("GET consulted the router")
		}
	}
}

func BenchmarkBodyLimitNeedsRouteHEAD(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if bodyLimitNeedsRoute("HEAD", 1<<20, 2<<20) {
			b.Fatal("HEAD consulted the router")
		}
	}
}

func BenchmarkBodyLimitNeedsRouteSmallPOST(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if bodyLimitNeedsRoute("POST", 1024, 2<<20) {
			b.Fatal("small POST consulted the router")
		}
	}
}

func BenchmarkBodyLimitRouteLookup(b *testing.B) {
	app := NewApplication(b.TempDir())
	for i := 0; i < 32; i++ {
		app.router.Get("/s/"+strconv.Itoa(i), func(req *http.Request) *http.Response {
			return http.Text("ok")
		})
	}
	app.router.Post("/webhook", func(req *http.Request) *http.Response {
		return http.Text("ok")
	}).BodyLimit(8 << 20)
	if err := app.router.Freeze(); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, ok := app.router.BodyLimitFor("POST", "/webhook"); !ok {
			b.Fatal("missing limit")
		}
	}
}
