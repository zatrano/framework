package bench

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"testing"

	khttp "github.com/zatrano/framework/v3/core/kernel/http"
)

func TestSceneHeaderBodyEqual(t *testing.T) {
	scenes := []struct {
		name string
		req  []byte
		body string
	}{
		{"plaintext", request, hello},
		{"json", requestJSON, `{"message":"Hello, World!"}`},
		{"param", requestUser, `{"id":42,"name":"user"}`},
		{"echo", requestEcho, string(sceneEchoBody)},
		{"search", requestSearch, `{"q":" Ada ","page":"1"}`},
	}
	for _, sc := range scenes {
		t.Run(sc.name, func(t *testing.T) {
			khttp.ConfigureRequestPool(false)
			off := mustCaptureReq(t, serveZatranoTier1Run, sc.req)
			khttp.ConfigureRequestPool(true)
			on := mustCaptureReq(t, serveZatranoTier1Run, sc.req)
			khttp.ConfigureRequestPool(false)
			if off.body != sc.body {
				t.Fatalf("pool off body %q", off.body)
			}
			assertSameResponse(t, "pool on", off, on)
			eq := mustCaptureReq(t, serveFiberTier1Eq, sc.req)
			assertSameResponse(t, "fiber equal-timeout", off, eq)
			def := mustCaptureReq(t, serveFiberTier1, sc.req)
			assertSameResponse(t, "fiber default", off, def)
		})
	}
}

func assertSameResponse(t *testing.T, name string, want, got captured) {
	t.Helper()
	if got.status != want.status || got.body != want.body {
		t.Fatalf("%s status=%d body=%q; zatrano status=%d body=%q", name, got.status, got.body, want.status, want.body)
	}
	if got.contentType != want.contentType {
		t.Fatalf("%s content-type %q; zatrano %q", name, got.contentType, want.contentType)
	}
	if got.requestID != want.requestID {
		t.Fatalf("%s X-Request-ID %q; zatrano %q", name, got.requestID, want.requestID)
	}
	for key, value := range want.headers {
		if got.headers[key] != value {
			t.Errorf("%s %s=%q; zatrano %q", name, key, got.headers[key], value)
		}
	}
	for key, value := range got.headers {
		if _, ok := want.headers[key]; !ok {
			t.Errorf("%s extra header %s=%q", name, key, value)
		}
	}
}

func TestSceneFullHeaders(t *testing.T) {
	scenes := []struct {
		name string
		req  []byte
	}{
		{"plaintext", request},
		{"json", requestJSON},
		{"param", requestUser},
		{"echo", requestEcho},
		{"search", requestSearch},
	}
	khttp.ConfigureRequestPool(false)
	t.Cleanup(func() { khttp.ConfigureRequestPool(false) })
	for _, sc := range scenes {
		t.Run(sc.name, func(t *testing.T) {
			base := readAllHeaders(t, serveZatranoTier1Run, sc.req)
			khttp.ConfigureRequestPool(true)
			on := readAllHeaders(t, serveZatranoTier1Run, sc.req)
			khttp.ConfigureRequestPool(false)
			eq := readAllHeaders(t, serveFiberTier1Eq, sc.req)
			def := readAllHeaders(t, serveFiberTier1, sc.req)
			sameHeaders(t, "pool on", base, on)
			sameHeaders(t, "fiber equal-timeout", base, eq)
			sameHeaders(t, "fiber default", base, def)
		})
	}
}

func BenchmarkSceneJSON(b *testing.B) { benchFaith(b, serveZatranoTier1Run, requestJSON) }
func BenchmarkSceneUser(b *testing.B) { benchFaith(b, serveZatranoTier1Run, requestUser) }
func BenchmarkSceneEcho(b *testing.B) { benchFaith(b, serveZatranoTier1Run, requestEcho) }
func BenchmarkSceneSearch(b *testing.B) {
	benchFaith(b, serveZatranoTier1Run, requestSearch)
}
func BenchmarkSceneJSONFiberEq(b *testing.B) {
	benchFaith(b, serveFiberTier1Eq, requestJSON)
}
func BenchmarkSceneUserFiberEq(b *testing.B) {
	benchFaith(b, serveFiberTier1Eq, requestUser)
}
func BenchmarkSceneEchoFiberEq(b *testing.B) {
	benchFaith(b, serveFiberTier1Eq, requestEcho)
}
func BenchmarkSceneSearchFiberEq(b *testing.B) {
	benchFaith(b, serveFiberTier1Eq, requestSearch)
}
func BenchmarkSceneJSONFiber(b *testing.B) { benchFaith(b, serveFiberTier1, requestJSON) }
func BenchmarkSceneUserFiber(b *testing.B) { benchFaith(b, serveFiberTier1, requestUser) }
func BenchmarkSceneEchoFiber(b *testing.B) { benchFaith(b, serveFiberTier1, requestEcho) }
func BenchmarkSceneSearchFiber(b *testing.B) {
	benchFaith(b, serveFiberTier1, requestSearch)
}

type fullResponse struct {
	status  int
	body    string
	headers http.Header
}

func readAllHeaders(t *testing.T, serve serveFunc, req []byte) fullResponse {
	t.Helper()
	c, buf := captureRequest(req)
	err := serve(c)
	if buf.Len() == 0 {
		t.Fatalf("empty response: %v", err)
	}
	resp, rerr := http.ReadResponse(bufio.NewReader(bytes.NewReader(buf.Bytes())), nil)
	if rerr != nil {
		t.Fatalf("read: %v", rerr)
	}
	defer resp.Body.Close()
	body, rerr := io.ReadAll(resp.Body)
	if rerr != nil {
		t.Fatal(rerr)
	}
	return fullResponse{status: resp.StatusCode, body: string(body), headers: resp.Header}
}

func sameHeaders(t *testing.T, name string, want, got fullResponse) {
	t.Helper()
	if got.status != want.status || got.body != want.body {
		t.Fatalf("%s status=%d body=%q; zatrano status=%d body=%q", name, got.status, got.body, want.status, want.body)
	}
	for key, values := range want.headers {
		if key == "Date" || key == "Server" {
			continue
		}
		if got.headers.Get(key) != want.headers.Get(key) {
			t.Errorf("%s %s=%q; zatrano %q", name, key, got.headers.Get(key), want.headers.Get(key))
		}
		_ = values
	}
	for key := range got.headers {
		if key == "Date" || key == "Server" {
			continue
		}
		if want.headers.Get(key) == "" && got.headers.Get(key) != "" {
			t.Errorf("%s extra %s=%q", name, key, got.headers.Get(key))
		}
	}
}
