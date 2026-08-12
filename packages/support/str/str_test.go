package str_test

import (
	"testing"

	"github.com/zatrano/framework/packages/support/str"
)

func TestSlugAndSnake(t *testing.T) {
	if got := str.Slug("Hello Zatrano!"); got != "hello-zatrano" {
		t.Fatalf("slug=%q", got)
	}
	if got := str.Snake("HelloWorld"); got != "hello_world" {
		t.Fatalf("snake=%q", got)
	}
	if got := str.Studly("hello_world"); got != "HelloWorld" {
		t.Fatalf("studly=%q", got)
	}
}

func TestMaskAndExcerpt(t *testing.T) {
	if got := str.Mask("1234567890", 2, 2); got != "12******90" {
		t.Fatalf("mask=%q", got)
	}
	if got := str.Excerpt("hello world from zatrano", 11); got != "hello..." {
		t.Fatalf("excerpt=%q", got)
	}
	if got := str.PadLeft("7", 3, "0"); got != "007" {
		t.Fatalf("padLeft=%q", got)
	}
	if got := str.PadRight("7", 3, "0"); got != "700" {
		t.Fatalf("padRight=%q", got)
	}
	if !str.ContainsAll("hello world", "hello", "world") || str.ContainsAll("hello", "bye") {
		t.Fatal("ContainsAll")
	}
	if !str.ContainsAny("hello", "x", "lo") || str.Headline("hello_world") != "Hello World" {
		t.Fatalf("ContainsAny/Headline failed")
	}
	if !str.StartsWithAny("hello_world", "hi", "hello") || str.StartsWithAny("hello", "x", "y") {
		t.Fatal("StartsWithAny")
	}
	if !str.EndsWithAny("hello_world", "planet", "world") || str.EndsWithAny("hello", "x", "y") {
		t.Fatal("EndsWithAny")
	}
	if str.Position("café-café", "fé") != 2 || str.Position("hello", "x") != -1 {
		t.Fatal("Position")
	}
	if str.LastPosition("café-café", "fé") != 7 || str.LastPosition("hello", "x") != -1 {
		t.Fatal("LastPosition")
	}
	if got := str.Trim("  hi  "); got != "hi" {
		t.Fatalf("trim=%q", got)
	}
	if got := str.Trim("--hi--", "-"); got != "hi" {
		t.Fatalf("trim chars=%q", got)
	}
	if got := str.LTrim("  hi  "); got != "hi  " {
		t.Fatalf("ltrim=%q", got)
	}
	if got := str.LTrim("--hi--", "-"); got != "hi--" {
		t.Fatalf("ltrim chars=%q", got)
	}
	if got := str.RTrim("  hi  "); got != "  hi" {
		t.Fatalf("rtrim=%q", got)
	}
	if got := str.RTrim("--hi--", "-"); got != "--hi" {
		t.Fatalf("rtrim chars=%q", got)
	}
	if got := str.Indent("a\nb", 2); got != "  a\n  b" {
		t.Fatalf("indent=%q", got)
	}
	if got := str.Indent("a\nb", 2, ">"); got != ">>a\n>>b" {
		t.Fatalf("indent pad=%q", got)
	}
	if got := str.Dedent("  a\n  b\n"); got != "a\nb\n" {
		t.Fatalf("dedent=%q", got)
	}
	if got := str.Dedent("\ta\n\t\tb"); got != "a\n\tb" {
		t.Fatalf("dedent tab=%q", got)
	}
	parts := str.Explode(",", "a,b,c")
	if len(parts) != 3 || parts[1] != "b" {
		t.Fatalf("explode=%v", parts)
	}
	limited := str.Explode(",", "a,b,c", 2)
	if len(limited) != 2 || limited[1] != "b,c" {
		t.Fatalf("explode limit=%v", limited)
	}
	if got := str.Join([]string{"a", "b", "c"}, "-"); got != "a-b-c" {
		t.Fatalf("join=%q", got)
	}
	lines := str.Lines("a\r\nb\nc")
	if len(lines) != 3 || lines[0] != "a" || lines[1] != "b" || lines[2] != "c" {
		t.Fatalf("lines=%v", lines)
	}
	if got := str.Squish("  a   b  "); got != "a b" {
		t.Fatalf("squish=%q", got)
	}
	if str.WordCount("one two three") != 3 {
		t.Fatal("word count")
	}
	if got := str.Swap("a_b", map[string]string{"_": "-"}); got != "a-b" {
		t.Fatalf("swap=%q", got)
	}
	if got := str.Remove("a-b-c", "-"); got != "abc" {
		t.Fatalf("remove=%q", got)
	}
	if got := str.Finish("path", "/"); got != "path/" {
		t.Fatalf("finish=%q", got)
	}
	if got := str.Start("path", "/"); got != "/path" {
		t.Fatalf("start=%q", got)
	}
	if got := str.ChopStart("www.example.com", "www.", "http://"); got != "example.com" {
		t.Fatalf("chop start=%q", got)
	}
	if got := str.ChopStart("example.com", "www."); got != "example.com" {
		t.Fatalf("chop start miss=%q", got)
	}
	if got := str.ChopEnd("file.bak.tmp", ".tmp", ".bak"); got != "file.bak" {
		t.Fatalf("chop end=%q", got)
	}
	if got := str.ChopEnd("file", ".tmp"); got != "file" {
		t.Fatalf("chop end miss=%q", got)
	}
	encoded := str.ToBase64("zatrano")
	if encoded != "emF0cmFubw==" {
		t.Fatalf("to base64=%q", encoded)
	}
	if got := str.FromBase64(encoded); got != "zatrano" {
		t.Fatalf("from base64=%q", got)
	}
	if got := str.FromBase64("!!!", "fallback"); got != "fallback" {
		t.Fatalf("from base64 fallback=%q", got)
	}
}

func TestReverseWrapWordsAscii(t *testing.T) {
	if got := str.Reverse("abcğ"); got != "ğcba" {
		t.Fatalf("reverse=%q", got)
	}
	if got := str.Wrap("x", "*", "+"); got != "*x+" {
		t.Fatalf("wrap=%q", got)
	}
	if got := str.Wrap("x", "\""); got != `"x"` {
		t.Fatalf("wrap same=%q", got)
	}
	if got := str.Words("one two three four", 2); got != "one two..." {
		t.Fatalf("words=%q", got)
	}
	if got := str.Words("one two", 5); got != "one two" {
		t.Fatalf("words short=%q", got)
	}
	if !str.IsAscii("hello") || str.IsAscii("şeker") {
		t.Fatal("is ascii")
	}
	if got := str.Ascii("Café Şeker"); got != "Cafe Seker" {
		t.Fatalf("ascii=%q", got)
	}
}

func TestAfterBeforeLastChopSubstr(t *testing.T) {
	if got := str.AfterLast("a.b.c.txt", "."); got != "txt" {
		t.Fatalf("after last=%q", got)
	}
	if got := str.BeforeLast("a.b.c.txt", "."); got != "a.b.c" {
		t.Fatalf("before last=%q", got)
	}
	if got := str.Chop("helloğ"); got != "hello" {
		t.Fatalf("chop=%q", got)
	}
	if got := str.Substr("abcdef", 2, 3); got != "cde" {
		t.Fatalf("substr=%q", got)
	}
	if got := str.Substr("abcdef", -2); got != "ef" {
		t.Fatalf("substr neg=%q", got)
	}
}

func TestBetweenTakeUnwrap(t *testing.T) {
	if got := str.Between("[a] bc [d]", "[", "]"); got != "a] bc [d" {
		t.Fatalf("between=%q", got)
	}
	if got := str.BetweenFirst("[a] bc [d]", "[", "]"); got != "a" {
		t.Fatalf("between first=%q", got)
	}
	if got := str.Take("Laravel", 4); got != "Lara" {
		t.Fatalf("take=%q", got)
	}
	if got := str.Take("Laravel", -5); got != "ravel" {
		t.Fatalf("take neg=%q", got)
	}
	if got := str.TakeLast("Laravel", 3); got != "vel" {
		t.Fatalf("take last=%q", got)
	}
	if got := str.Unwrap(`"Laravel"`, `"`); got != "Laravel" {
		t.Fatalf("unwrap=%q", got)
	}
	if got := str.Unwrap("{Laravel}", "{", "}"); got != "Laravel" {
		t.Fatalf("unwrap pair=%q", got)
	}
}

func TestMatchIsMatchMatchAllNumbers(t *testing.T) {
	if got := str.Match(`foo (.*)`, "foo bar"); got != "bar" {
		t.Fatalf("match capture=%q", got)
	}
	if got := str.Match(`foo`, "foo bar"); got != "foo" {
		t.Fatalf("match full=%q", got)
	}
	if !str.IsMatch(`\d+`, "abc123") || str.IsMatch(`^abc$`, "abcd") {
		t.Fatal("IsMatch")
	}
	all := str.MatchAll(`#(\w+)`, "a #one b #two")
	if len(all) != 2 || all[0] != "one" || all[1] != "two" {
		t.Fatalf("match all=%v", all)
	}
	if got := str.Numbers("Tel: +90 (212) 555-0199"); got != "902125550199" {
		t.Fatalf("numbers=%q", got)
	}
	if str.Match(`[`, "x") != "" || str.IsMatch(`[`, "x") || str.MatchAll(`[`, "x") != nil {
		t.Fatal("invalid pattern")
	}
}

func TestWordWrapPadBothCharAtSubstrCount(t *testing.T) {
	if got := str.PadBoth("go", 6, "-"); got != "--go--" {
		t.Fatalf("pad both=%q", got)
	}
	if got := str.CharAt("Laravel", 0); got != "L" {
		t.Fatalf("charAt=%q", got)
	}
	if got := str.CharAt("Laravel", -1); got != "l" {
		t.Fatalf("charAt neg=%q", got)
	}
	if str.CharAt("ab", 5) != "" {
		t.Fatal("charAt oob")
	}
	if got := str.SubstrCount("ababab", "ab"); got != 3 {
		t.Fatalf("substr count=%d", got)
	}
	if str.SubstrCount("aaa", "") != 0 {
		t.Fatal("substr count empty needle")
	}
	wrapped := str.WordWrap("the quick brown fox", 10)
	if wrapped != "the quick\nbrown fox" {
		t.Fatalf("word wrap=%q", wrapped)
	}
	if str.WordWrap("hi", 0) != "hi" {
		t.Fatal("word wrap zero width")
	}
}

func TestRepeatAndBlankHelpers(t *testing.T) {
	if got := str.Repeat("ab", 3); got != "ababab" {
		t.Fatalf("repeat=%q", got)
	}
	if str.Repeat("x", 0) != "" {
		t.Fatal("repeat zero")
	}
	if !str.IsEmpty("") || str.IsEmpty("a") || !str.IsNotEmpty("a") {
		t.Fatal("empty")
	}
	if !str.IsBlank("  \t") || str.IsBlank("a") || !str.IsNotBlank(" a ") {
		t.Fatal("blank")
	}
}

func TestReplaceFirstLastAndCaseFirst(t *testing.T) {
	if got := str.ReplaceFirst("foo bar foo", "foo", "baz"); got != "baz bar foo" {
		t.Fatalf("replace first=%q", got)
	}
	if got := str.ReplaceLast("foo bar foo", "foo", "baz"); got != "foo bar baz" {
		t.Fatalf("replace last=%q", got)
	}
	if got := str.UcFirst("hello"); got != "Hello" {
		t.Fatalf("ucfirst=%q", got)
	}
	if got := str.LcFirst("Hello"); got != "hello" {
		t.Fatalf("lcfirst=%q", got)
	}
	if str.UcFirst("") != "" || str.LcFirst("") != "" {
		t.Fatal("empty case first")
	}
	if str.ReplaceFirst("abc", "x", "y") != "abc" || str.ReplaceLast("abc", "", "y") != "abc" {
		t.Fatal("replace edge")
	}
	if got := str.ReplaceStart("Hello world", "Hello", "Hi"); got != "Hi world" {
		t.Fatalf("replace start=%q", got)
	}
	if got := str.ReplaceEnd("Hello world", "world", "ZATRANO"); got != "Hello ZATRANO" {
		t.Fatalf("replace end=%q", got)
	}
	if str.ReplaceStart("Hello", "Hi", "X") != "Hello" || str.ReplaceEnd("Hello", "Hi", "X") != "Hello" {
		t.Fatal("replace start/end miss")
	}
	if got := str.ReplaceArray("? and ?", "?", []string{"A", "B"}); got != "A and B" {
		t.Fatalf("replace array=%q", got)
	}
	if got := str.ReplaceArray("? ?", "?", []string{"A"}); got != "A ?" {
		t.Fatalf("replace array partial=%q", got)
	}
	if got := str.SubstrReplace("1300", ":", 2); got != "13:" {
		t.Fatalf("substr replace=%q", got)
	}
	if got := str.SubstrReplace("The Event", "Annual ", 4, 0); got != "The Annual Event" {
		t.Fatalf("substr insert=%q", got)
	}
	if got := str.SubstrReplace("abcdef", "X", -2); got != "abcdX" {
		t.Fatalf("substr neg=%q", got)
	}
}

func TestIsUpperLowerAlpha(t *testing.T) {
	if !str.IsUpper("HELLO") || str.IsUpper("Hello") || !str.IsUpper("123!") {
		t.Fatal("is upper")
	}
	if !str.IsLower("hello") || str.IsLower("Hello") || !str.IsLower("123!") {
		t.Fatal("is lower")
	}
	if !str.IsAlpha("Zatrano") || str.IsAlpha("Go1") || str.IsAlpha("") {
		t.Fatal("is alpha")
	}
	if !str.IsAlphaNumeric("Go1") || str.IsAlphaNumeric("Go-1") || str.IsAlphaNumeric("") {
		t.Fatal("is alphanumeric")
	}
	if !str.IsDigit("42") || str.IsDigit("4a") || str.IsDigit("") {
		t.Fatal("is digit")
	}
	if !str.IsHex("0Ff") || str.IsHex("0G") || str.IsHex("") {
		t.Fatal("is hex")
	}
}

func TestIsUuidUlidJsonUrl(t *testing.T) {
	if !str.IsUuid("550e8400-e29b-41d4-a716-446655440000") || str.IsUuid("not-a-uuid") {
		t.Fatal("uuid")
	}
	if !str.IsUlid("01ARZ3NDEKTSV4RRFFQ69G5FAV") || str.IsUlid("short") {
		t.Fatal("ulid")
	}
	if !str.IsJson(`{"ok":true}`) || !str.IsJson(`[1,2]`) || str.IsJson("{") {
		t.Fatal("json")
	}
	if !str.IsUrl("https://zatrano.test/path") || str.IsUrl("/relative") || str.IsUrl("not a url") {
		t.Fatal("url")
	}
	if !str.IsEmail("admin@zatrano.test") || str.IsEmail("not-an-email") || str.IsEmail("") {
		t.Fatal("email")
	}
	if !str.IsIp("127.0.0.1") || !str.IsIp("::1") || str.IsIp("999.0.0.1") {
		t.Fatal("ip")
	}
	if !str.IsMac("01:23:45:67:89:ab") || str.IsMac("bad-mac") {
		t.Fatal("mac")
	}
	if !str.IsSemver("1.2.3") || !str.IsSemver("v1.0.0-alpha") || str.IsSemver("1.2") || str.IsSemver("") {
		t.Fatal("semver")
	}
}
