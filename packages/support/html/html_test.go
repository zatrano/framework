package html_test

import (
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/support/html"
)

func TestSanitize(t *testing.T) {
	raw := `<p onclick="alert(1)">Hi <script>evil()</script><b>there</b></p>`
	clean := html.Sanitize(raw, "p", "b")
	if strings.Contains(clean, "script") || strings.Contains(clean, "onclick") {
		t.Fatalf("%q", clean)
	}
	if !strings.Contains(clean, "<b>there</b>") {
		t.Fatalf("%q", clean)
	}
	if html.StripTags(raw) != "Hi there" && !strings.Contains(html.StripTags(raw), "Hi") {
		t.Fatalf("strip=%q", html.StripTags(raw))
	}
	if html.Escape("<x>") != "&lt;x&gt;" {
		t.Fatal(html.Escape("<x>"))
	}
}
