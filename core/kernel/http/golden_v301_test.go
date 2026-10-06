package http

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	stdhttp "net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/zatrano/rawhttp"
)

// goldenV301 is the header set and accessor bytes produced by v3.0.1 for
// these same calls. Query values are not trimmed and an empty query value
// stays empty, on both trees.
const goldenV301 = `== header cookies
status=200
body="ok"
Content-Length=2
Content-Type=text/plain; charset=utf-8
Set-Cookie=a=1; Path=/ | b=2; Path=/ | c=3; Path=/
cookie-count=3
cookie a=1
cookie b=2
cookie c=3
== header add-set-del-case
status=200
body="ok"
Content-Language=tr
Content-Length=2
Content-Type=text/plain; charset=utf-8
Vary=Origin | Accept
X-Frame-Options=DENY
cookie-count=0
== header after-headers-call
status=200
body="ok"
Content-Length=2
Content-Type=text/plain; charset=utf-8
X-A=1 | 2 | 4
X-C=only-map
cookie-count=0
== accessors
query-only spaced="  " blank="" missing="fb" multi="a"
query-after-transform spaced="  " blank=""
form query.q=" raw " input.name="Ada" input.note="" post.name="Ada" post.note="" post.keep=""
form.all{keep="",multi="1",name="Ada",q="raw"}
form.only{name="Ada"}
form.except{keep="",multi="1",q="raw"}
form.body="name=%20Ada%20&note=&keep=%20%20&multi=9"
json.dest={"note":"","tags":[" a ","b"],"user":{"name":"  Ada  "}}
json.all{tags="[\" a \",\"b\"]",user="{\"name\":\"  Ada  \"}",user.name="Ada"}
file.all{title="Ada"}
file.name="notes.txt"
merge input.name="  Ada  " attr=kept
later input.name=" Ada "
`

func TestHeaderAndAccessorsMatchV301(t *testing.T) {
	got := goldenDump(t)
	if got != goldenV301 {
		t.Fatalf("v3.0.1 mismatch\n got:\n%s\nwant:\n%s", got, goldenV301)
	}
}

func goldenDump(t *testing.T) string {
	t.Helper()
	var b strings.Builder
	dumpHeaderCase(t, &b, "cookies", func() *Response {
		resp := Text("ok")
		resp.WithCookie(&stdhttp.Cookie{Name: "a", Value: "1", Path: "/"})
		resp.WithCookie(&stdhttp.Cookie{Name: "b", Value: "2", Path: "/"})
		resp.WithCookie(&stdhttp.Cookie{Name: "c", Value: "3", Path: "/"})
		return resp
	})
	dumpHeaderCase(t, &b, "add-set-del-case", func() *Response {
		resp := Text("ok")
		resp.Header("X-Request-ID", "one")
		resp.AppendHeader("x-request-id", "two")
		resp.Header("Vary", "Origin")
		resp.AppendHeader("vary", "Accept")
		resp.WithoutHeader("X-Request-ID")
		resp.Header("x-frame-options", "DENY")
		resp.Header("Content-Language", "tr")
		return resp
	})
	dumpHeaderCase(t, &b, "after-headers-call", func() *Response {
		resp := Text("ok")
		resp.Header("X-A", "1")
		resp.AppendHeader("X-A", "2")
		_ = resp.Headers()
		resp.Header("X-B", "3")
		resp.AppendHeader("x-a", "4")
		resp.WithoutHeader("X-B")
		resp.Header("X-C", "only-map")
		return resp
	})
	b.WriteString("== accessors\n")
	b.WriteString(dumpAccessors())
	return b.String()
}

func dumpHeaderCase(t *testing.T, b *strings.Builder, name string, build func() *Response) {
	t.Helper()
	parsed, body := dumpCommit(t, build())
	fmt.Fprintf(b, "== header %s\nstatus=%d\nbody=%q\n", name, parsed.StatusCode, body)
	keys := make([]string, 0, len(parsed.Header))
	for key := range parsed.Header {
		if strings.EqualFold(key, "Date") {
			continue
		}
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Fprintf(b, "%s=%s\n", key, strings.Join(parsed.Header[key], " | "))
	}
	cookies := parsed.Cookies()
	fmt.Fprintf(b, "cookie-count=%d\n", len(cookies))
	for _, c := range cookies {
		fmt.Fprintf(b, "cookie %s=%s\n", c.Name, c.Value)
	}
}

func dumpCommit(t *testing.T, resp *Response) (*stdhttp.Response, string) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := &rawhttp.Server{
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
		IdleTimeout:  time.Second,
		Handler: func(ctx *rawhttp.Ctx) {
			_ = resp.Commit(ctx)
		},
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	conn, err := net.DialTimeout("tcp", ln.Addr().String(), time.Second)
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

func dumpAccessors() string {
	var b strings.Builder
	trim := func(key, value string) (string, bool) {
		value = strings.TrimSpace(value)
		if value == "" && key != "keep" {
			return "", false
		}
		return value, true
	}
	qonly := httptest.NewRequest(stdhttp.MethodGet, "/?spaced=%20%20&blank=&multi=a&multi=b", nil)
	qr := RequestFromHTTP(qonly)
	fmt.Fprintf(&b, "query-only spaced=%q blank=%q missing=%q multi=%q\n",
		qr.Query("spaced"), qr.Query("blank"), qr.Query("missing", "fb"), qr.Query("multi"))
	qr.TransformInputs(trim)
	fmt.Fprintf(&b, "query-after-transform spaced=%q blank=%q\n", qr.Query("spaced"), qr.Query("blank"))

	form := httptest.NewRequest(stdhttp.MethodPost, "/?q=%20raw%20&multi=1&multi=2", strings.NewReader("name=%20Ada%20&note=&keep=%20%20&multi=9"))
	form.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req := RequestFromHTTP(form)
	req.TransformInputs(trim)
	fmt.Fprintf(&b, "form query.q=%q input.name=%q input.note=%q post.name=%q post.note=%q post.keep=%q\n",
		req.Query("q"), req.Input("name"), req.Input("note"), req.PostForm("name"), req.PostForm("note"), req.PostForm("keep"))
	writeMap(&b, "form.all", req.All())
	writeMap(&b, "form.only", req.Only("name"))
	writeMap(&b, "form.except", req.Except("name"))
	body, err := req.Body()
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(&b, "form.body=%q\n", string(body))

	rawJSON := httptest.NewRequest(stdhttp.MethodPost, "/", strings.NewReader(`{"user":{"name":"  Ada  "},"note":"","tags":[" a ","b"]}`))
	rawJSON.Header.Set("Content-Type", "application/json")
	jreq := RequestFromHTTP(rawJSON)
	jreq.TransformInputs(trim)
	var dest map[string]any
	if err := jreq.JSON(&dest); err != nil {
		panic(err)
	}
	enc, err := json.Marshal(dest)
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(&b, "json.dest=%s\n", enc)
	writeMap(&b, "json.all", jreq.All())

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	if err := w.WriteField("title", "  Ada  "); err != nil {
		panic(err)
	}
	fw, err := w.CreateFormFile("doc", "notes.txt")
	if err != nil {
		panic(err)
	}
	if _, err := fw.Write([]byte("file-bytes")); err != nil {
		panic(err)
	}
	if err := w.Close(); err != nil {
		panic(err)
	}
	mreq := testRequest(stdhttp.MethodPost, "/", buf.Bytes())
	mreq.SetHeader("Content-Type", w.FormDataContentType())
	mreq.TransformInputs(trim)
	writeMap(&b, "file.all", mreq.All())
	file, err := mreq.File("doc")
	if err != nil {
		panic(err)
	}
	fmt.Fprintf(&b, "file.name=%q\n", file.Header.Filename)

	empty := testRequest(stdhttp.MethodGet, "/", nil)
	empty.TransformInputs(trim)
	empty.Set("attr", "kept")
	empty.Merge(map[string]string{"name": "  Ada  "})
	fmt.Fprintf(&b, "merge input.name=%q attr=%v\n", empty.Input("name"), empty.Get("attr"))

	later := testRequest(stdhttp.MethodGet, "/", nil)
	later.TransformInputs(trim)
	_ = later.Input("missing")
	later.SetBody([]byte("name=%20Ada%20"))
	later.SetHeader("Content-Type", "application/x-www-form-urlencoded")
	fmt.Fprintf(&b, "later input.name=%q\n", later.Input("name"))
	return b.String()
}

func writeMap(b *strings.Builder, name string, m map[string]string) {
	if m == nil {
		fmt.Fprintf(b, "%s=<nil>\n", name)
		return
	}
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fmt.Fprintf(b, "%s{", name)
	for i, key := range keys {
		if i > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(b, "%s=%q", key, m[key])
	}
	b.WriteString("}\n")
}
