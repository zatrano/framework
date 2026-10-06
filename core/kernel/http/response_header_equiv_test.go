package http

import (
	"bufio"
	"io"
	"math/rand"
	"net"
	stdhttp "net/http"
	"net/textproto"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zatrano/rawhttp"
)

func TestCanonicalHeaderNamesMatchMIME(t *testing.T) {
	names := []string{
		"X-Request-ID", "x-request-id", "X-Request-Id",
		"ETag", "etag",
		"X-CSRF-TOKEN", "X-Csrf-Token",
		"X-RateLimit-Limit", "X-RateLimit-Remaining",
		"X-Frame-Options", "Vary", "Content-Type",
	}
	for _, name := range names {
		if got, want := canonicalResponseHeader(name), textproto.CanonicalMIMEHeaderKey(name); got != want {
			t.Fatalf("%q canonical=%q mime=%q", name, got, want)
		}
	}
}

func TestHeaderOpsListMatchesMap(t *testing.T) {
	names := []string{"X-Frame-Options", "x-request-id", "Vary", "Set-Cookie", "Weird_Name"}
	rng := rand.New(rand.NewSource(1))
	for trial := 0; trial < 40; trial++ {
		list := &Response{}
		mapped := &Response{}
		_ = mapped.Headers()
		if !mapped.HeaderMapBuilt() || list.HeaderMapBuilt() {
			t.Fatal("source flags")
		}
		for step := 0; step < 24; step++ {
			name := names[rng.Intn(len(names))]
			value := "v" + string(rune('a'+rng.Intn(26)))
			switch rng.Intn(3) {
			case 0:
				list.SetHeader(name, value)
				mapped.SetHeader(name, value)
			case 1:
				list.AddHeader(name, value)
				mapped.AddHeader(name, value)
			default:
				list.WithoutHeader(name)
				mapped.WithoutHeader(name)
			}
		}
		for _, name := range names {
			if list.GetHeader(name) != mapped.GetHeader(name) {
				t.Fatalf("get %s list=%q map=%q", name, list.GetHeader(name), mapped.GetHeader(name))
			}
		}
		promoted := list.Headers()
		if !list.HeaderMapBuilt() {
			t.Fatal("Headers did not take the map")
		}
		list.SetHeader("X-After", "only-map")
		if promoted.Get("X-After") != "only-map" {
			t.Fatal("write after Headers missed the map")
		}
		for key, values := range mapped.Headers() {
			if strings.Join(promoted.Values(key), ",") != strings.Join(values, ",") {
				t.Fatalf("key %s list=%v map=%v", key, promoted.Values(key), values)
			}
		}
	}
}

func TestThreeSetCookiesRoundTrip(t *testing.T) {
	resp := Text("ok")
	resp.WithCookie(&stdhttp.Cookie{Name: "a", Value: "1", Path: "/"})
	resp.WithCookie(&stdhttp.Cookie{Name: "b", Value: "2", Path: "/"})
	resp.WithCookie(&stdhttp.Cookie{Name: "c", Value: "3", Path: "/"})
	if resp.HeaderMapBuilt() {
		t.Fatal("cookies built the header map")
	}
	parsed, body := commitResponse(t, resp)
	if body != "ok" {
		t.Fatalf("body=%q", body)
	}
	cookies := parsed.Cookies()
	got := map[string]string{}
	for _, c := range cookies {
		got[c.Name] = c.Value
	}
	if got["a"] != "1" || got["b"] != "2" || got["c"] != "3" {
		t.Fatalf("cookies=%v", got)
	}
}

func TestCarrierRejectsHeaderAndCookieBreaks(t *testing.T) {
	ctx := &rawhttp.Ctx{}
	for _, value := range []string{"a\r\nX-Injected: z", "a\nX-Injected: z", "a\x00b", "a\r\n"} {
		if err := ctx.AddHeader("X-Trace", value); err != rawhttp.ErrHeaderInvalid {
			t.Fatalf("AddHeader %q err=%v", value, err)
		}
		if err := ctx.SetHeader("X-Trace", value); err != rawhttp.ErrHeaderInvalid {
			t.Fatalf("SetHeader %q err=%v", value, err)
		}
		if err := ctx.SetCookie(&rawhttp.Cookie{Name: "sid", Value: value, Path: "/"}); err != rawhttp.ErrHeaderInvalid {
			t.Fatalf("SetCookie %q err=%v", value, err)
		}
	}
	if err := ctx.AddHeader("X-Trace", "ok"); err != nil {
		t.Fatal(err)
	}
	if err := ctx.SetCookie(&rawhttp.Cookie{Name: "sid", Value: "ok", Path: "/"}); err != nil {
		t.Fatal(err)
	}
}

func FuzzResponseHeaderInjection(f *testing.F) {
	f.Add("X-Ok", "yes")
	f.Add("X-A", "a\r\nb")
	f.Add("X-A", "a\nb")
	f.Add("X-A", "a\x00b")
	f.Add("X-A", "%0d%0aX-Injected: z")
	f.Add("Bad\r\nName", "v")
	f.Add("X-Utf", "héllo\u2028")
	f.Add("X-Long", strings.Repeat("a", 4096))
	f.Fuzz(func(t *testing.T, name, value string) {
		resp := Text("ok")
		resp.SetHeader("X-Ok", "yes")
		resp.AddHeader(name, value)
		resp.WithCookie(&stdhttp.Cookie{Name: "sid", Value: value, Path: "/"})
		parsed, body := fuzzCommit(t, resp)
		if parsed.StatusCode != 200 || body != "ok" {
			t.Fatalf("status=%d body=%q", parsed.StatusCode, body)
		}
		if parsed.Header.Get("X-Ok") != "yes" {
			t.Fatalf("lost safe header: %v", parsed.Header)
		}
		if len(parsed.Header) > 8 {
			t.Fatalf("too many headers: %v", parsed.Header)
		}
		for key, vals := range parsed.Header {
			for _, v := range vals {
				if strings.ContainsAny(key, "\r\n\x00") || strings.ContainsAny(v, "\r\n\x00") {
					t.Fatalf("split %q=%q", key, v)
				}
			}
		}
		for _, cookie := range parsed.Cookies() {
			if strings.ContainsAny(cookie.Name, "\r\n") || strings.ContainsAny(cookie.Value, "\r\n") {
				t.Fatalf("cookie split %#v", cookie)
			}
		}
	})
}

// fuzzGate is one server for the whole process. Fuzz workers serialize on mu
// so a failed dial is not saved as a crashing input.
var fuzzGate struct {
	once sync.Once
	ln   net.Listener
	mu   sync.Mutex
	resp *Response
}

func fuzzCommit(t *testing.T, resp *Response) (*stdhttp.Response, string) {
	t.Helper()
	fuzzGate.once.Do(func() {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			panic(err)
		}
		fuzzGate.ln = ln
		srv := &rawhttp.Server{
			ReadTimeout:  time.Second,
			WriteTimeout: time.Second,
			IdleTimeout:  time.Second,
			Handler: func(ctx *rawhttp.Ctx) {
				_ = fuzzGate.resp.Commit(ctx)
			},
		}
		go func() { _ = srv.Serve(ln) }()
	})
	fuzzGate.mu.Lock()
	defer fuzzGate.mu.Unlock()
	fuzzGate.resp = resp

	var conn net.Conn
	var err error
	for attempt := 0; attempt < 20; attempt++ {
		conn, err = net.DialTimeout("tcp", fuzzGate.ln.Addr().String(), time.Second)
		if err == nil {
			break
		}
		time.Sleep(2 * time.Millisecond)
	}
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := io.WriteString(conn, "GET / HTTP/1.1\r\nHost: localhost\r\nConnection: close\r\n\r\n"); err != nil {
		t.Fatal(err)
	}
	parsed, err := stdhttp.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer parsed.Body.Close()
	raw, err := io.ReadAll(parsed.Body)
	if err != nil {
		t.Fatal(err)
	}
	return parsed, string(raw)
}
