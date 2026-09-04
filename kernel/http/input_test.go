package http_test

import (
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/zatrano/framework/kernel/http"
)

func TestInputHelpers(t *testing.T) {
	raw := httptest.NewRequest(stdhttp.MethodPost, "/?q=1", strings.NewReader("name=Ada&flag=true&age=30&rate=1.5&empty="))
	raw.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req := http.NewRequest(raw)

	if !req.Has("name") || !req.Filled("name") || req.Missing("name") {
		t.Fatal("Has/Filled/Missing")
	}
	if req.Filled("empty") {
		t.Fatal("empty should not be filled")
	}
	if !req.Boolean("flag") || req.Integer("age") != 30 || req.Float("rate") != 1.5 {
		t.Fatal("typed helpers")
	}
	except := req.Except("flag")
	if _, ok := except["flag"]; ok || except["name"] != "Ada" {
		t.Fatalf("except=%v", except)
	}

	req.Merge(map[string]string{"name": "Grace", "role": "admin"})
	if req.Input("name") != "Grace" || req.Input("role") != "admin" || !req.Boolean("flag") {
		t.Fatalf("merge failed: %#v", req.All())
	}
	req.Replace(map[string]string{"only": "x"})
	all := req.All()
	if len(all) != 1 || all["only"] != "x" {
		t.Fatalf("replace failed: %#v", all)
	}
}

func TestInputAnyAllHelpers(t *testing.T) {
	raw := httptest.NewRequest(stdhttp.MethodPost, "/", strings.NewReader("name=Ada&flag=true&empty="))
	raw.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req := http.NewRequest(raw)

	if !req.HasAny("missing", "name") || req.HasAny("x", "y") {
		t.Fatal("HasAny")
	}
	if !req.HasAll("name", "flag") || req.HasAll("name", "x") {
		t.Fatal("HasAll")
	}
	if !req.MissingAny("name", "x") || req.MissingAny("name", "flag") {
		t.Fatal("MissingAny")
	}
	if !req.FilledAny("empty", "name") || req.FilledAny("empty", "missing") {
		t.Fatal("FilledAny")
	}
	if !req.FilledAll("name", "flag") || req.FilledAll("name", "empty") {
		t.Fatal("FilledAll")
	}
}

func TestInputKeysValuesEmpty(t *testing.T) {
	raw := httptest.NewRequest(stdhttp.MethodPost, "/", strings.NewReader("name=Ada&flag=true"))
	raw.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req := http.NewRequest(raw)

	keys := req.Keys()
	if len(keys) != 2 || keys[0] != "flag" || keys[1] != "name" {
		t.Fatalf("keys=%v", keys)
	}
	values := req.Values()
	if len(values) != 2 || values[0] != "true" || values[1] != "Ada" {
		t.Fatalf("values=%v", values)
	}
	if !req.MissingAll("x", "y") || req.MissingAll("name", "x") {
		t.Fatal("MissingAll")
	}
	if req.IsEmpty() || !req.IsNotEmpty() {
		t.Fatal("IsEmpty/IsNotEmpty filled")
	}

	empty := http.NewRequest(httptest.NewRequest(stdhttp.MethodGet, "/", nil))
	if !empty.IsEmpty() || empty.IsNotEmpty() || len(empty.Keys()) != 0 {
		t.Fatal("empty request")
	}
}

func TestInputWhenHelpers(t *testing.T) {
	raw := httptest.NewRequest(stdhttp.MethodPost, "/", strings.NewReader("name=Ada&flag=1&empty="))
	raw.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req := http.NewRequest(raw)

	flags := map[string]bool{}
	req.WhenHas("name", func(*http.Request) { flags["has_name"] = true }).
		WhenHas("missing", func(*http.Request) { flags["has_missing"] = true }).
		WhenFilled("name", func(*http.Request) { flags["filled_name"] = true }).
		WhenFilled("empty", func(*http.Request) { flags["filled_empty"] = true }).
		WhenMissing("x", func(*http.Request) { flags["missing_x"] = true }).
		WhenMissing("name", func(*http.Request) { flags["missing_name"] = true }).
		WhenBoolean("flag", func(*http.Request) { flags["bool_flag"] = true }).
		WhenBoolean("name", func(*http.Request) { flags["bool_name"] = true })

	if !flags["has_name"] || flags["has_missing"] {
		t.Fatal("WhenHas")
	}
	if !flags["filled_name"] || flags["filled_empty"] {
		t.Fatal("WhenFilled")
	}
	if !flags["missing_x"] || flags["missing_name"] {
		t.Fatal("WhenMissing")
	}
	if !flags["bool_flag"] || flags["bool_name"] {
		t.Fatal("WhenBoolean")
	}
}

func TestMergeIfMissingAndWhenAny(t *testing.T) {
	raw := httptest.NewRequest(stdhttp.MethodPost, "/", strings.NewReader("name=Ada"))
	raw.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req := http.NewRequest(raw)

	req.MergeIfMissing(map[string]string{"name": "Grace", "role": "admin"})
	if req.Input("name") != "Ada" || req.Input("role") != "admin" {
		t.Fatalf("MergeIfMissing=%#v", req.All())
	}

	flags := map[string]bool{}
	req.WhenHasAny([]string{"x", "name"}, func(*http.Request) { flags["has_any"] = true }).
		WhenHasAny([]string{"x", "y"}, func(*http.Request) { flags["has_none"] = true }).
		WhenFilledAny([]string{"empty", "name"}, func(*http.Request) { flags["filled_any"] = true }).
		WhenFilledAny([]string{"x", "y"}, func(*http.Request) { flags["filled_none"] = true }).
		WhenMissingAny([]string{"name", "x"}, func(*http.Request) { flags["missing_any"] = true }).
		WhenMissingAny([]string{"name", "role"}, func(*http.Request) { flags["missing_none"] = true })

	if !flags["has_any"] || flags["has_none"] {
		t.Fatal("WhenHasAny")
	}
	if !flags["filled_any"] || flags["filled_none"] {
		t.Fatal("WhenFilledAny")
	}
	if !flags["missing_any"] || flags["missing_none"] {
		t.Fatal("WhenMissingAny")
	}

	flags = map[string]bool{}
	req.WhenHasAll([]string{"name", "role"}, func(*http.Request) { flags["has_all"] = true }).
		WhenHasAll([]string{"name", "x"}, func(*http.Request) { flags["has_all_fail"] = true }).
		WhenFilledAll([]string{"name", "role"}, func(*http.Request) { flags["filled_all"] = true }).
		WhenFilledAll([]string{"name", "x"}, func(*http.Request) { flags["filled_all_fail"] = true }).
		WhenMissingAll([]string{"x", "y"}, func(*http.Request) { flags["missing_all"] = true }).
		WhenMissingAll([]string{"name", "x"}, func(*http.Request) { flags["missing_all_fail"] = true })
	if !flags["has_all"] || flags["has_all_fail"] {
		t.Fatal("WhenHasAll")
	}
	if !flags["filled_all"] || flags["filled_all_fail"] {
		t.Fatal("WhenFilledAll")
	}
	if !flags["missing_all"] || flags["missing_all_fail"] {
		t.Fatal("WhenMissingAll")
	}

	onlyFilled := req.OnlyFilled("name", "ghost", "role")
	if len(onlyFilled) != 2 || onlyFilled["name"] != "Ada" || onlyFilled["role"] != "admin" {
		t.Fatalf("OnlyFilled=%#v", onlyFilled)
	}

	req.Merge(map[string]string{"note": "hi", "ghost": ""})
	exceptFilled := req.ExceptFilled("role")
	if len(exceptFilled) != 2 || exceptFilled["name"] != "Ada" || exceptFilled["note"] != "hi" {
		t.Fatalf("ExceptFilled=%#v", exceptFilled)
	}
	if _, ok := exceptFilled["role"]; ok {
		t.Fatal("ExceptFilled should omit role")
	}
	if _, ok := exceptFilled["ghost"]; ok {
		t.Fatal("ExceptFilled should omit empty")
	}

	if req.Empty("ghost") != true || req.Empty("name") {
		t.Fatal("Empty")
	}

	flags = map[string]bool{}
	req.WhenTrue("name", func(*http.Request) { flags["true_name"] = true }).
		WhenTrue("flag", func(*http.Request) { flags["true_flag"] = true }).
		WhenFalse("flag", func(*http.Request) { flags["false_flag"] = true }).
		WhenFalse("ghost", func(*http.Request) { flags["false_ghost"] = true })
	req.Merge(map[string]string{"flag": "1"})
	req.WhenTrue("flag", func(*http.Request) { flags["true_flag"] = true }).
		WhenFalse("flag", func(*http.Request) { flags["false_flag_after"] = true })
	if flags["true_name"] || !flags["true_flag"] || flags["false_flag_after"] || !flags["false_ghost"] {
		t.Fatalf("WhenTrue/WhenFalse=%#v", flags)
	}

	flags = map[string]bool{}
	req.WhenEmpty("ghost", func(*http.Request) { flags["empty_ghost"] = true }).
		WhenEmpty("name", func(*http.Request) { flags["empty_name"] = true })
	if !flags["empty_ghost"] || flags["empty_name"] {
		t.Fatal("WhenEmpty")
	}

	exceptEmpty := req.ExceptEmpty()
	if exceptEmpty["ghost"] != "" || exceptEmpty["name"] != "Ada" || exceptEmpty["note"] != "hi" {
		t.Fatalf("ExceptEmpty=%#v", exceptEmpty)
	}

	role, roleOK := req.Enum("role", "user", "admin")
	if !roleOK || role != "admin" {
		t.Fatalf("Enum=%q ok=%v", role, roleOK)
	}
	if _, ok := req.Enum("role", "guest"); ok {
		t.Fatal("Enum miss")
	}

	raw.Header.Set("X-Demo", "zatrano")
	if !req.HasHeader("X-Demo") || req.HasHeader("X-Missing") {
		t.Fatal("HasHeader")
	}
	if !req.MissingHeader("X-Missing") || req.MissingHeader("X-Demo") {
		t.Fatal("MissingHeader")
	}

	raw.AddCookie(&stdhttp.Cookie{Name: "session", Value: "abc"})
	if !req.HasCookie("session") || req.HasCookie("ghost") {
		t.Fatal("HasCookie")
	}
	if !req.MissingCookie("ghost") || req.MissingCookie("session") {
		t.Fatal("MissingCookie")
	}

	flags = map[string]bool{}
	req.WhenHasCookie("session", func(*http.Request) { flags["has_session"] = true }).
		WhenHasCookie("ghost", func(*http.Request) { flags["has_ghost"] = true }).
		WhenMissingCookie("ghost", func(*http.Request) { flags["missing_ghost"] = true }).
		WhenMissingCookie("session", func(*http.Request) { flags["missing_session"] = true }).
		WhenHasHeader("X-Demo", func(*http.Request) { flags["has_hdr"] = true }).
		WhenHasHeader("X-Missing", func(*http.Request) { flags["has_missing_hdr"] = true }).
		WhenMissingHeader("X-Missing", func(*http.Request) { flags["missing_hdr"] = true }).
		WhenMissingHeader("X-Demo", func(*http.Request) { flags["missing_demo"] = true })
	if !flags["has_session"] || flags["has_ghost"] || !flags["missing_ghost"] || flags["missing_session"] {
		t.Fatalf("WhenCookie=%#v", flags)
	}
	if !flags["has_hdr"] || flags["has_missing_hdr"] || !flags["missing_hdr"] || flags["missing_demo"] {
		t.Fatalf("WhenHeader=%#v", flags)
	}
}

func TestRequestCompletionHelpers(t *testing.T) {
	raw := httptest.NewRequest(stdhttp.MethodGet, "/api/demo/request/complete?q=1&flag=true&n=7&rate=1.5&empty=&tag=a&tag=b", nil)
	raw.Header.Set("Accept", "application/json,text/html")
	raw.Header.Set("Content-Type", "application/json")
	raw.Header.Set("X-Requested-With", "XMLHttpRequest")
	raw.Header.Set("X-Demo", "zatrano")
	raw.AddCookie(&stdhttp.Cookie{Name: "session", Value: "abc"})
	raw.AddCookie(&stdhttp.Cookie{Name: "theme", Value: "dark"})
	raw.Host = "example.test:8443"
	raw.TLS = nil
	req := http.NewRequest(raw)
	req.Merge(map[string]string{
		"name":  "Ada",
		"role":  "admin",
		"ghost": "",
		"day":   "2026-08-05",
		"age":   "30",
		"score": "2.5",
		"ok":    "yes",
		"bad":   "maybe",
	})

	if !req.Exists("name") || !req.AnyFilled("ghost", "name") {
		t.Fatal("Exists/AnyFilled")
	}
	if !req.EmptyAny("ghost", "name") || !req.EmptyAll("ghost", "missing") || req.EmptyAll("name", "ghost") {
		t.Fatal("EmptyAny/EmptyAll")
	}

	flags := map[string]bool{}
	req.WhenNotFilled("ghost", func(*http.Request) { flags["not_filled"] = true }).
		WhenNotEmpty("name", func(*http.Request) { flags["not_empty"] = true }).
		WhenEmptyAny([]string{"name", "ghost"}, func(*http.Request) { flags["empty_any"] = true }).
		WhenEmptyAll([]string{"ghost", "missing"}, func(*http.Request) { flags["empty_all"] = true })
	if !flags["not_filled"] || !flags["not_empty"] || !flags["empty_any"] || !flags["empty_all"] {
		t.Fatalf("when empty helpers=%#v", flags)
	}

	if n, ok := req.IntegerOK("age"); !ok || n != 30 {
		t.Fatalf("IntegerOK=%d ok=%v", n, ok)
	}
	if _, ok := req.IntegerOK("bad"); ok {
		t.Fatal("IntegerOK miss")
	}
	if f, ok := req.FloatOK("score"); !ok || f != 2.5 {
		t.Fatalf("FloatOK=%v ok=%v", f, ok)
	}
	if v, ok := req.BooleanOK("ok"); !ok || !v {
		t.Fatal("BooleanOK true")
	}
	if _, ok := req.BooleanOK("bad"); ok {
		t.Fatal("BooleanOK invalid")
	}
	fallback := time.Date(2000, 1, 1, 0, 0, 0, 0, time.Local)
	if got := req.DateOr("day", fallback); got.Format("2006-01-02") != "2026-08-05" {
		t.Fatalf("DateOr=%v", got)
	}
	if got := req.DateOr("missing", fallback); !got.Equal(fallback) {
		t.Fatal("DateOr fallback")
	}
	if req.EnumOr("role", "user", "user", "admin") != "admin" || req.EnumOr("role", "user", "guest") != "user" {
		t.Fatal("EnumOr")
	}

	req.MergeIfFilled(map[string]string{"city": "Ankara", "skip": ""})
	if req.Input("city") != "Ankara" || req.Filled("skip") {
		t.Fatalf("MergeIfFilled city=%q skip filled=%v", req.Input("city"), req.Filled("skip"))
	}

	pulled := req.Pull("city", "")
	if pulled != "Ankara" || req.Has("city") {
		t.Fatalf("Pull=%q has=%v", pulled, req.Has("city"))
	}
	req.Forget("ghost")
	if req.Has("ghost") {
		t.Fatal("Forget")
	}

	if !req.HasAnyHeader("X-Missing", "X-Demo") || !req.HasAllHeaders("Accept", "X-Demo") {
		t.Fatal("HasAny/AllHeaders")
	}
	if !req.MissingAnyHeader("Accept", "X-Missing") || !req.MissingAllHeaders("X-A", "X-B") {
		t.Fatal("MissingAny/AllHeaders")
	}
	hdrFlags := map[string]bool{}
	req.WhenHasAnyHeader([]string{"X-Missing", "Accept"}, func(*http.Request) { hdrFlags["has"] = true }).
		WhenMissingAnyHeader([]string{"Accept", "X-Missing"}, func(*http.Request) { hdrFlags["miss"] = true })
	if !hdrFlags["has"] || !hdrFlags["miss"] {
		t.Fatal("When*Header any")
	}
	if req.HeadersMap()["X-Demo"] != "zatrano" || !strings.Contains(req.ContentType(), "application/json") {
		t.Fatal("HeadersMap/ContentType")
	}
	if !req.AcceptsJSON() || req.AcceptsXml() || !req.IsXmlHttpRequest() {
		t.Fatal("accept/ajax aliases")
	}

	if !req.HasAnyCookie("ghost", "session") || !req.HasAllCookies("session", "theme") {
		t.Fatal("HasAny/AllCookies")
	}
	if !req.MissingAnyCookie("session", "ghost") || !req.MissingAllCookies("a", "b") {
		t.Fatal("MissingAny/AllCookies")
	}
	cookieFlags := map[string]bool{}
	req.WhenHasAnyCookie([]string{"ghost", "session"}, func(*http.Request) { cookieFlags["has"] = true }).
		WhenMissingAnyCookie([]string{"session", "ghost"}, func(*http.Request) { cookieFlags["miss"] = true })
	if !cookieFlags["has"] || !cookieFlags["miss"] {
		t.Fatal("When*Cookie any")
	}
	if req.CookieMap()["theme"] != "dark" {
		t.Fatal("CookieMap")
	}

	if !req.IsGet() || req.IsPost() || !req.IsMethodSafe() || !req.IsMethodIdempotent() {
		t.Fatal("method helpers")
	}
	if !req.HasQuery("empty") || !req.HasQuery("q") || req.HasQuery("nope") {
		t.Fatal("HasQuery")
	}
	if req.QueryInt("n") != 7 || req.QueryFloat("rate") != 1.5 || !req.QueryBool("flag") {
		t.Fatal("QueryInt/Float/Bool")
	}
	if req.Port() != "8443" || req.HttpHost() != "example.test:8443" {
		t.Fatalf("Port/HttpHost port=%q host=%q", req.Port(), req.HttpHost())
	}
	if req.DecodedPath() != "/api/demo/request/complete" || !strings.Contains(req.QueryString(), "q=1") {
		t.Fatal("DecodedPath/QueryString")
	}
	if !strings.HasPrefix(req.RequestURI(), "/api/demo/request/complete?") {
		t.Fatalf("RequestURI=%q", req.RequestURI())
	}
	with := req.FullUrlWithQuery(map[string]string{"extra": "1"})
	without := req.FullUrlWithoutQuery("tag", "empty")
	if !strings.Contains(with, "extra=1") || strings.Contains(without, "tag=") {
		t.Fatalf("url query helpers with=%q without=%q", with, without)
	}
	if len(req.Ips()) == 0 {
		t.Fatal("Ips")
	}
	if req.IsSecure() != req.Secure() {
		t.Fatal("IsSecure")
	}
}

func TestInputDateAndLists(t *testing.T) {
	raw := httptest.NewRequest(stdhttp.MethodGet, "/?day=2026-08-04&tags=a,%20b,%20c&nums=1,2,x,3&rates=1.5,2", nil)
	req := http.NewRequest(raw)

	day, ok := req.Date("day")
	if !ok || day.Format("2006-01-02") != "2026-08-04" {
		t.Fatalf("date=%v ok=%v", day, ok)
	}
	if _, ok := req.Date("missing"); ok {
		t.Fatal("missing date")
	}
	if _, ok := req.Date("tags"); ok {
		t.Fatal("invalid date")
	}
	tags := req.Strings("tags")
	if len(tags) != 3 || tags[0] != "a" || tags[1] != "b" || tags[2] != "c" {
		t.Fatalf("strings=%v", tags)
	}
	nums := req.Integers("nums")
	if len(nums) != 3 || nums[0] != 1 || nums[2] != 3 {
		t.Fatalf("integers=%v", nums)
	}
	rates := req.Floats("rates")
	if len(rates) != 2 || rates[0] != 1.5 || rates[1] != 2 {
		t.Fatalf("floats=%v", rates)
	}
}

func TestRedirectBack(t *testing.T) {
	raw := httptest.NewRequest(stdhttp.MethodPost, "/profile", nil)
	raw.Host = "localhost:8080"
	raw.Header.Set("Referer", "http://localhost:8080/demo/form")
	req := http.NewRequest(raw)
	resp := http.RedirectBack(req, "/")
	if resp.RedirectURL() != "/demo/form" {
		t.Fatalf("redirect=%q", resp.RedirectURL())
	}
}

func TestRedirectAwayRefreshSecurePermanent(t *testing.T) {
	away := http.Away("https://example.com/docs")
	if away.RedirectURL() != "https://example.com/docs" || away.StatusCode() != stdhttp.StatusFound {
		t.Fatalf("away=%q status=%d", away.RedirectURL(), away.StatusCode())
	}

	raw := httptest.NewRequest(stdhttp.MethodGet, "/api/demo/path?x=1", nil)
	raw.Host = "zatrano.test"
	req := http.NewRequest(raw)
	refresh := http.Refresh(req)
	if refresh.RedirectURL() != "http://zatrano.test/api/demo/path?x=1" {
		t.Fatalf("refresh=%q", refresh.RedirectURL())
	}

	secure := http.SecureRedirect(req, "dashboard")
	if secure.RedirectURL() != "https://zatrano.test/dashboard" {
		t.Fatalf("secure=%q", secure.RedirectURL())
	}

	perm := http.PermanentRedirect("/gone")
	if perm.RedirectURL() != "/gone" || perm.StatusCode() != stdhttp.StatusMovedPermanently {
		t.Fatalf("permanent=%q status=%d", perm.RedirectURL(), perm.StatusCode())
	}
}

func TestNestedJSONInput(t *testing.T) {
	body := `{"user":{"profile":{"name":"Ada"}},"flag":true}`
	raw := httptest.NewRequest(stdhttp.MethodPost, "/api", strings.NewReader(body))
	raw.Header.Set("Content-Type", "application/json")
	req := http.NewRequest(raw)
	if req.Dot("user.profile.name") != "Ada" {
		t.Fatalf("dot=%q", req.Dot("user.profile.name"))
	}
	if !req.HasNested("user.profile.name") {
		t.Fatal("HasNested")
	}
	m := req.JSONMap()
	user, _ := m["user"].(map[string]any)
	if user == nil {
		t.Fatal("JSONMap user")
	}
	if req.Input("user.profile.name") != "Ada" {
		t.Fatalf("flattened input=%q", req.Input("user.profile.name"))
	}
	if v, ok := req.InputAny("flag").(bool); !ok || !v {
		t.Fatalf("InputAny flag=%v", req.InputAny("flag"))
	}
}
