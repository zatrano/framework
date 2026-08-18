package view

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIfInsideHTMLAttributeCompilesWithBackticks(t *testing.T) {
	e := New(t.TempDir())
	src := `<button class="@if($save_disabled)opacity-50 cursor-not-allowed@endif" href="@if($status == 'draft')/edit@else/view@endif">x</button>`
	compiled, err := e.compileBladeLike(src)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(compiled, `"save_disabled"`) || strings.Contains(compiled, `"status"`) {
		t.Fatalf("path lits must use backticks inside attributes, got:\n%s", compiled)
	}
	if !strings.Contains(compiled, "`save_disabled`") || !strings.Contains(compiled, "`status`") {
		t.Fatalf("expected backtick path lits, got:\n%s", compiled)
	}
	if !strings.Contains(compiled, "`draft`") {
		t.Fatalf("expected backtick comparison lit, got:\n%s", compiled)
	}
}

func TestIfInsideHTMLAttributeRenders(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "btn.html"), []byte(
		`<a class="@if($save_disabled)opacity-50@endif" href="@if($ok)/yes@else/no@endif">x</a>`,
	), 0o644)
	e := New(dir)
	out, err := e.Render("btn", map[string]any{"save_disabled": true, "ok": false})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `class="opacity-50"`) {
		t.Fatalf("class=%q", out)
	}
	if !strings.Contains(out, `href="/no"`) {
		t.Fatalf("href=%q", out)
	}
}
