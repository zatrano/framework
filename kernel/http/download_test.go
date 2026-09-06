package http_test

import (
	"strings"
	"testing"

	"github.com/zatrano/framework/v2/kernel/http"
)

func TestDownloadBytes(t *testing.T) {
	resp := http.DownloadBytes([]byte("a,b\n1,2\n"), "report.csv", "text/csv")
	if resp.Headers().Get("Content-Disposition") == "" || !strings.Contains(resp.Headers().Get("Content-Disposition"), "report.csv") {
		t.Fatalf("disposition=%q", resp.Headers().Get("Content-Disposition"))
	}
	if string(resp.Content()) != "a,b\n1,2\n" {
		t.Fatalf("body=%q", string(resp.Content()))
	}
}

func TestInlineBytesAndAsDisposition(t *testing.T) {
	inline := http.InlineBytes([]byte("hello"), `note"x.txt`, "text/plain")
	disp := inline.Headers().Get("Content-Disposition")
	if !strings.HasPrefix(disp, "inline;") || !strings.Contains(disp, "note'x.txt") {
		t.Fatalf("inline disposition=%q", disp)
	}
	if string(inline.Content()) != "hello" {
		t.Fatalf("inline body=%q", string(inline.Content()))
	}

	asDown := http.JSON(map[string]any{"ok": true}).AsDownload("payload.json")
	if !strings.Contains(asDown.Headers().Get("Content-Disposition"), "attachment;") ||
		!strings.Contains(asDown.Headers().Get("Content-Disposition"), "payload.json") {
		t.Fatalf("as download=%q", asDown.Headers().Get("Content-Disposition"))
	}

	asInline := http.Text("preview").AsInline("preview.txt")
	if !strings.Contains(asInline.Headers().Get("Content-Disposition"), "inline;") ||
		!strings.Contains(asInline.Headers().Get("Content-Disposition"), "preview.txt") {
		t.Fatalf("as inline=%q", asInline.Headers().Get("Content-Disposition"))
	}
}

func TestDownloadJSON(t *testing.T) {
	resp := http.DownloadJSON(map[string]any{"name": "Ada"}, "user")
	disp := resp.Headers().Get("Content-Disposition")
	if !strings.Contains(disp, "attachment;") || !strings.Contains(disp, "user.json") {
		t.Fatalf("disposition=%q", disp)
	}
	if resp.ContentType() != "application/json" {
		t.Fatalf("content-type=%q", resp.ContentType())
	}
	if !strings.Contains(string(resp.Content()), `"name":"Ada"`) {
		t.Fatalf("body=%q", string(resp.Content()))
	}
}
