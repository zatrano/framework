package http_test

import (
	"bytes"
	"mime/multipart"
	stdhttp "net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zatrano/framework/v2/kernel/http"
)

func TestAbortUnlessAndDownloadFile(t *testing.T) {
	if http.AbortUnless(true, 403) != nil {
		t.Fatal("unless true")
	}
	if http.AbortUnless(false, 403) == nil {
		t.Fatal("unless false")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(path, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	dl := http.Download(path, "")
	if dl == nil || !strings.Contains(dl.Headers().Get("Content-Disposition"), "attachment") {
		t.Fatal("download")
	}
	inline := http.InlineFile(path)
	if inline == nil || !strings.Contains(inline.Headers().Get("Content-Disposition"), "inline") {
		t.Fatal("inline")
	}
	emptyName := http.DownloadBytes(nil, "", "")
	if !strings.Contains(emptyName.Headers().Get("Content-Disposition"), "download") {
		t.Fatal("default filename")
	}
	if http.DownloadJSON(make(chan int), "").ContentType() != "application/json" {
		t.Fatal("download json marshal fail")
	}
	if http.InlineBytes(nil, "", "").Headers().Get("Content-Disposition") == "" {
		t.Fatal("inline bytes defaults")
	}
	var nilResp *http.Response
	if nilResp.AsDownload("x") != nil || nilResp.AsInline("x") != nil {
		t.Fatal("nil response")
	}
	if http.Text("x").AsDownload("").Headers().Get("Content-Disposition") == "" {
		t.Fatal("as download default")
	}
	if http.Text("x").AsInline("").Headers().Get("Content-Disposition") == "" {
		t.Fatal("as inline default")
	}
	pub := http.PublicFile(path, httptest.NewRequest("GET", "/", nil))
	if pub == nil {
		t.Fatal("public file")
	}
	hj := http.Hijack(func(w stdhttp.ResponseWriter) error { return nil })
	if hj.StatusCode() != 101 {
		t.Fatal("hijack")
	}
}

func TestCommitWriterUnwrapFlush(t *testing.T) {
	rec := httptest.NewRecorder()
	w := http.TrackCommit(rec)
	if w.Unwrap() != rec {
		t.Fatal("unwrap")
	}
	w.Flush()
	if !w.Committed() {
		t.Fatal("flush commits")
	}
	again := http.TrackCommit(w)
	if again != w {
		t.Fatal("track existing")
	}
	var nilW *http.CommitWriter
	if nilW.Unwrap() != nil {
		t.Fatal("nil unwrap")
	}
	nilW.Flush()
}

func TestRequestSurfaceAndTransform(t *testing.T) {
	raw := httptest.NewRequest(stdhttp.MethodPost, "https://app.example:8443/items/1?q=2&flag=true&n=3&f=1.5", strings.NewReader("name=Ada&empty="))
	raw.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	raw.Header.Set("Accept", "application/json, text/html")
	raw.Header.Set("X-Requested-With", "XMLHttpRequest")
	raw.AddCookie(&stdhttp.Cookie{Name: "sid", Value: "abc"})
	raw.AddCookie(&stdhttp.Cookie{Name: "theme", Value: "dark"})
	req := http.NewRequest(raw)
	req.SetRouteParams(map[string]string{"id": "1"})
	req.SetRouteName("items.show")
	req.Set("attr", 7)

	if !req.IsPost() || req.IsGet() || req.IsPut() || req.IsPatch() || req.IsDelete() || req.IsHead() || req.IsOptions() {
		t.Fatal("method flags")
	}
	if req.IsMethodSafe() || req.IsMethodIdempotent() {
		t.Fatal("safe/idempotent")
	}
	if req.Get("attr") != 7 || req.Route("id") != "1" || req.RouteName() != "items.show" {
		t.Fatal("attr/route")
	}
	if !req.HasQuery("q") || req.QueryInt("n") != 3 || req.QueryFloat("f") != 1.5 || !req.QueryBool("flag") {
		t.Fatal("query typed")
	}
	if req.Port() != "8443" || req.HttpHost() == "" || req.DecodedPath() == "" {
		t.Fatal("host/path")
	}
	if req.QueryString() == "" || req.RequestURI() == "" {
		t.Fatal("uri")
	}
	if req.FullUrlWithQuery(map[string]string{"extra": "1"}) == "" {
		t.Fatal("full with query")
	}
	if req.FullUrlWithoutQuery("q") == "" {
		t.Fatal("full without query")
	}
	if len(req.Ips()) == 0 || req.IsSecure() != req.Secure() {
		t.Fatal("ips/secure")
	}
	if !req.HasCookie("sid") || req.MissingCookie("nope") == false {
		t.Fatal("cookie flags")
	}
	if !req.HasHeader("Accept") || req.MissingHeader("Nope") == false {
		t.Fatal("header flags")
	}
	if req.HeadersMap()["Accept"] == "" || req.CookieMap()["sid"] != "abc" {
		t.Fatal("maps")
	}
	if !req.IsXmlHttpRequest() || !req.Ajax() {
		t.Fatal("xhr")
	}
	if req.ContentType() == "" || req.UserAgent() != raw.UserAgent() {
		t.Fatal("content/ua")
	}
	_ = req.AcceptsXml()
	_ = req.PrefersHtml()
	_ = req.PrefersJSON()
	_ = req.AcceptsJSON()
	_ = req.AcceptsHtml()
	_ = req.ExpectsJSON()
	_ = req.WantsJSON()
	_ = req.HasAnyHeader("Accept", "X")
	_ = req.HasAllHeaders("Accept")
	_ = req.MissingAnyHeader("Nope")
	_ = req.MissingAllHeaders("Nope")
	_ = req.HasAnyCookie("sid")
	_ = req.HasAllCookies("sid")
	_ = req.MissingAnyCookie("nope")
	_ = req.MissingAllCookies("nope")
	req.WhenHasCookie("sid", func(*http.Request) {})
	req.WhenMissingCookie("nope", func(*http.Request) {})
	req.WhenHasHeader("Accept", func(*http.Request) {})
	req.WhenMissingHeader("Nope", func(*http.Request) {})
	req.WhenHasAnyHeader([]string{"Accept"}, func(*http.Request) {})
	req.WhenMissingAnyHeader([]string{"Nope"}, func(*http.Request) {})
	req.WhenHasAnyCookie([]string{"sid"}, func(*http.Request) {})
	req.WhenMissingAnyCookie([]string{"nope"}, func(*http.Request) {})
	_ = req.Cookies()
	_ = req.RouteInt("id")
	_ = req.RouteParams()
	_ = req.Queries()
	_ = req.QueryAll()
	_ = req.Agent()
	_ = req.Old("name")
	_ = req.Segment(1)
	_ = req.Segments()
	_ = req.PathIs("items/*")
	_ = req.ExactPath("/items/1")
	_ = req.RouteIs("items.*")
	_ = req.IsMethod("POST")
	_ = req.Prefers("application/json", "text/html")
	_ = req.Accepts("application/json")
	_ = req.Root()
	_ = req.FullURL()
	_ = req.Host()
	_ = req.Scheme()
	_ = req.Pjax()
	_ = req.BearerToken()
	_ = req.IP()
	_ = req.RemoteIP()
	_ = req.HasFileAny("f")

	req.TransformInputs(func(key, value string) (string, bool) {
		if key == "empty" {
			return "", false
		}
		if key == "name" {
			return strings.ToUpper(value), true
		}
		return value, true
	})
	if req.Input("name") != "ADA" {
		t.Fatalf("transform name=%q", req.Input("name"))
	}
	req.WhenHas("name", func(*http.Request) {})
	req.WhenFilled("name", func(*http.Request) {})
	req.WhenMissing("nope", func(*http.Request) {})
	req.WhenBoolean("flag", func(*http.Request) {})
	req.WhenTrue("flag", func(*http.Request) {})
	req.WhenFalse("empty", func(*http.Request) {})
	req.WhenEmpty("empty", func(*http.Request) {})
	req.WhenNotFilled("empty", func(*http.Request) {})
	req.WhenNotEmpty("name", func(*http.Request) {})
	req.WhenEmptyAny([]string{"empty"}, func(*http.Request) {})
	req.WhenEmptyAll([]string{"empty"}, func(*http.Request) {})
	req.WhenHasAny([]string{"name"}, func(*http.Request) {})
	req.WhenFilledAny([]string{"name"}, func(*http.Request) {})
	req.WhenMissingAny([]string{"nope"}, func(*http.Request) {})
	req.WhenHasAll([]string{"name"}, func(*http.Request) {})
	req.WhenFilledAll([]string{"name"}, func(*http.Request) {})
	req.WhenMissingAll([]string{"nope"}, func(*http.Request) {})
	_ = req.Only("name")
	_ = req.OnlyFilled("name")
	_ = req.ExceptFilled("empty")
	_ = req.ExceptEmpty()
	_ = req.Exists("name")
	_ = req.AnyFilled("name")
	_ = req.EmptyAny("empty")
	_ = req.EmptyAll("empty")
	_, _ = req.IntegerOK("n")
	_, _ = req.FloatOK("f")
	_, _ = req.BooleanOK("flag")
	_ = req.DateOr("d", time.Now())
	_ = req.EnumOr("name", "x", "Ada", "ADA")
	_ = req.Pull("missing", "fb")
	req.MergeIfFilled(map[string]string{"name": "x"})
	_ = req.Keys()
	_ = req.Values()
	_ = req.IsEmpty()
	_ = req.IsNotEmpty()
	_ = req.String("name")
	_, _ = req.Enum("name", "ADA")
	_, _ = req.Date("d")
	_ = req.Strings("name")
	_ = req.Integers("n")
	_ = req.Floats("f")
	req.MergeIfMissing(map[string]string{"z": "1"})
	req.Forget("z")
	_ = req.JSONMap()
	_ = req.InputAny("name")
	_ = req.Dot("name")
	_ = req.HasNested("name")
	req.SetMaxBodyBytes(1024)
	_, _ = req.Body()
}

func TestResponseAndStreamSurface(t *testing.T) {
	t.Setenv("MAX_BODY_BYTES", "4096")
	_ = http.MaxBodyBytes()
	_ = http.MaxRequestBytes()
	raw := httptest.NewRequest(stdhttp.MethodGet, "/x?q=1", nil)
	raw.Header.Set("Authorization", "Bearer tok")
	raw.AddCookie(&stdhttp.Cookie{Name: "sid", Value: "abc"})
	req := http.NewRequest(raw)
	if req.URL() == "" || req.Cookie("sid") != "abc" || req.BearerToken() != "tok" {
		t.Fatal("url/cookie/bearer")
	}
	if req.Cookie("nope", "fb") != "fb" || req.Header("Nope", "h") != "h" {
		t.Fatal("fallbacks")
	}
	req.Set("_client_ip", "10.0.0.1")
	if req.IP() != "10.0.0.1" {
		t.Fatal("client ip")
	}
	view := http.View("home", map[string]any{"a": 1})
	if view.ViewName() != "home" || view.ViewData()["a"] != 1 || view.Error() != nil {
		t.Fatal("view")
	}
	_ = view.FilePath()
	redir := http.Redirect("/go", 302)
	if !redir.IsRedirect() || redir.RedirectURL() == "" {
		t.Fatal("redirect")
	}
	redir.Location("/elsewhere")
	_ = http.RedirectRoute("", 302)
	_ = http.RedirectRoute("/named")
	_ = http.Found("/x")
	_ = http.SeeOther("/x")
	_ = http.TemporaryRedirect("/x")
	rec := httptest.NewRecorder()
	if err := http.Text("hi").WriteTo(rec); err != nil {
		t.Fatal(err)
	}
	var nilResp *http.Response
	_ = nilResp.WriteTo(httptest.NewRecorder())
	fileResp := http.File(filepath.Join(t.TempDir(), "missing.txt"))
	_ = fileResp.WriteTo(httptest.NewRecorder())
	made := http.Make(201, []byte("x"), "")
	made.SetContent([]byte("y"), "text/plain")
	made.WithContentType("text/plain")
	made.NoCache()
	made.CacheFor(10 * time.Second)
	made.PrivateCache(5 * time.Second)
	made.Vary("Accept")
	made.AppendHeader("X-A", "1")
	made.WithoutHeaders("X-A")
	_ = made.HasHeader("Content-Type")
	_ = made.GetHeader("Content-Type")
	made.Allow("GET")
	made.ContentEncoding("gzip")
	made.WwwAuthenticate(`Basic realm="x"`)
	made.StaleWhileRevalidate(1)
	made.StaleIfError(1)
	made.MaxAge(10)
	made.SharedMaxAge(10)
	made.Private()
	_ = made.IsSuccessful()
	_ = made.IsOk()
	_ = made.IsEmpty()
	_ = made.IsRedirection()
	_ = made.IsClientError()
	_ = made.IsServerError()
	_ = made.IsForbidden()
	_ = made.IsNotFound()
	_ = made.IsUnauthorized()
	_ = made.IsInformational()
	_ = made.IsJSON()
	_ = made.IsHTML()
	_ = made.IsText()
	_ = made.ContentLength()
	_ = http.JSONP("cb", map[string]int{"n": 1})
	_ = http.JSONP("", make(chan int))
	stream := http.Stream("text/plain", func(w stdhttp.ResponseWriter, f stdhttp.Flusher) error { return nil })
	if !stream.IsStream() {
		t.Fatal("stream")
	}
	sse := http.SSE(func(send func(http.SSEEvent) error) error {
		return send(http.SSEEvent{Event: "x", Data: "a\nb", ID: "1", Retry: 1})
	})
	_ = sse.WriteTo(httptest.NewRecorder())
	tick := http.SSETick(1, time.Nanosecond)
	_ = tick
	_ = http.SSETick(0, 0)
	_ = http.StreamDownload("f.bin", func(w stdhttp.ResponseWriter, f stdhttp.Flusher) error { return nil })
	_ = http.RedirectBack(req, "/")
	_ = http.Away("https://example.com")
	_ = http.Refresh(req)
	_ = http.SecureRedirect(req, "/x")
	_ = http.PermanentRedirect("/x")
	html := http.HTML("<p>x</p>")
	_ = html.IsHTML()
	_ = http.Abort(401).IsUnauthorized()
	_ = http.Abort(403).IsForbidden()
	_ = http.Abort(404).IsNotFound()
	made.WithCookie(&stdhttp.Cookie{Name: "a", Value: "b"})
	made.ReplaceCookies([]*stdhttp.Cookie{{Name: "c", Value: "d"}})
	made.Charset("utf-8")
	made.Cookie("k", "v", 10)
	made.CookieForever("f", "v")
	made.CookieMinutes("m", "v", 1)
	made.SecureCookie("s", "v")
	made.WithCookieOptions("o", "v", http.CookieOptions{})
	made.WithHeaders(map[string]string{"X-B": "1"})
	made.ETag("w")
	made.LastModified(time.Now())
	made.ExpiresAt(time.Now())
	made.MustRevalidate()
	made.Immutable()
	made.Public()
	made.ContentLanguage("en")
	_ = made.Cookies()
}

func TestMultipartUploadHelpers(t *testing.T) {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, err := mw.CreateFormFile("doc", "note.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	raw := httptest.NewRequest(stdhttp.MethodPost, "/upload", &buf)
	raw.Header.Set("Content-Type", mw.FormDataContentType())
	req := http.NewRequest(raw)
	if !req.HasFile("doc") || !req.HasFileAny("doc") {
		t.Fatal("has file")
	}
	f, err := req.File("doc")
	if err != nil {
		t.Fatal(err)
	}
	if f.Name() != "note.txt" || f.Size() == 0 || f.Extension() != ".txt" {
		t.Fatalf("%s %d %s", f.Name(), f.Size(), f.Extension())
	}
	_ = f.Mime()
	rc, err := f.File()
	if err != nil {
		t.Fatal(err)
	}
	_ = rc.Close()
	files, err := req.Files("doc")
	if err != nil || len(files) != 1 {
		t.Fatal(err)
	}
	dest, err := f.Store(t.TempDir())
	if err != nil || dest == "" {
		t.Fatal(err)
	}
}
