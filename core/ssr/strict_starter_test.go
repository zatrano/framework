package ssr_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/v3/core/kernel"
	"github.com/zatrano/framework/v3/core/ssr"
)

func TestStarterStubsRenderUnderEscapeStrict(t *testing.T) {
	dir := t.TempDir()
	app := kernel.NewApplication(dir)
	if err := ssr.WriteStarterTemplates(app); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(dir, "templates")
	eng := ssr.New(root)
	if eng == nil {
		t.Fatal("nil engine")
	}
	if eng.LangEscapeCatalog() {
		t.Fatal("boot/default must not force SetLangEscapeCatalog(true)")
	}
	eng.Share("appName", "ZATRANO")
	eng.Share("locale", "en")
	out, err := eng.Render("web.welcome", nil)
	if err != nil {
		t.Fatalf("starter under EscapeStrict (default): %v", err)
	}
	if !strings.Contains(out, "ZATRANO") {
		t.Fatalf("expected brand in output, got %q", out)
	}
	if !strings.Contains(out, `lang="en"`) {
		t.Fatalf("expected locale attr, got %q", out)
	}
	if strings.Contains(out, "@extends") || strings.Contains(out, "@yield") {
		t.Fatalf("directives not compiled: %q", out)
	}
}

func TestLangEscapePolicyViaCanvasDefault(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		path := filepath.Join(dir, name+".html")
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("lang_text", `<p>@lang('k')</p>`)
	write("lang_param", `<p>@lang('k', ['name' => $x])</p>`)
	write("lang_attr", `<div title="@lang('k')"></div>`)

	eng := ssr.New(dir)

	out, err := eng.Render("lang_text", map[string]any{
		"__trans": map[string]string{"k": `<b>ok</b>`},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `<b>ok</b>`) {
		t.Fatalf("catalog text must be trusted by default, got %q", out)
	}

	out, err = eng.Render("lang_param", map[string]any{
		"x":       `<script>x</script>`,
		"__trans": map[string]string{"k": `Hi <b>:name</b>`},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `<b>`) {
		t.Fatalf("catalog HTML stays in text, got %q", out)
	}
	if strings.Contains(out, `<script>`) || !strings.Contains(out, `&lt;script&gt;`) {
		t.Fatalf("param must be escaped, got %q", out)
	}

	out, err = eng.Render("lang_attr", map[string]any{
		"__trans": map[string]string{"k": `" onclick=x`},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, `title="" onclick`) || strings.Contains(out, `title="\" onclick`) {
		t.Fatalf("attr breakout, got %q", out)
	}
	if !strings.Contains(out, `&#34;`) && !strings.Contains(out, `&quot;`) {
		t.Fatalf("attr catalog quote must be escaped, got %q", out)
	}

	eng.SetLangEscapeCatalog(true)
	out, err = eng.Render("lang_text", map[string]any{
		"__trans": map[string]string{"k": `<b>ok</b>`},
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, `<b>ok</b>`) || !strings.Contains(out, `&lt;b&gt;`) {
		t.Fatalf("SetLangEscapeCatalog(true) must escape catalog, got %q", out)
	}
}
