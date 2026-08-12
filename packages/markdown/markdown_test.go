package markdown_test

import (
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/markdown"
)

func TestMarkdownToHTML(t *testing.T) {
	html := markdown.ToHTML("# Hello\n\nThis is **bold** and a [link](https://zatrano.test).")
	if !strings.Contains(html, "<h1>Hello</h1>") {
		t.Fatalf("missing h1: %s", html)
	}
	if !strings.Contains(html, "<strong>bold</strong>") {
		t.Fatalf("missing bold: %s", html)
	}
	if !strings.Contains(html, `<a href="https://zatrano.test">link</a>`) {
		t.Fatalf("missing link: %s", html)
	}
}

func TestMarkdownFencedCodeAndTable(t *testing.T) {
	html := markdown.ToHTML("```go\nfmt.Println(\"hi\")\n```\n\n| Key | Purpose |\n|-----|---------|\n| `app.name` | Name |")
	if !strings.Contains(html, `<pre><code class="language-go">`) {
		t.Fatalf("missing fenced code: %s", html)
	}
	if !strings.Contains(html, "fmt.Println") {
		t.Fatalf("missing code body: %s", html)
	}
	if !strings.Contains(html, "<table>") || !strings.Contains(html, "<th>Key</th>") {
		t.Fatalf("missing table: %s", html)
	}
}
