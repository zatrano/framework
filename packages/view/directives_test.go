package view_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/validation"
	"github.com/zatrano/framework/packages/view"
)

func TestFormDirectives(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "form.html"), []byte(`
@csrf
@method('PUT')
<input value="@old('email', 'n/a')">
@error('email')
<span>{{ error . "email" }}</span>
@enderror
@auth
<p>auth</p>
@endauth
@guest
<p>guest</p>
@endguest
`), 0o644)

	engine := view.New(dir)
	out, err := engine.Render("form", map[string]any{
		"_token": "tok",
		"old":    map[string]string{"email": "a@b.c"},
		"errors": validation.NewMessageBag(validation.Errors{"email": {"bad"}}),
		"auth":   true,
		"guest":  false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `name="_token"`) || !strings.Contains(out, "tok") {
		t.Fatalf("csrf missing: %s", out)
	}
	if !strings.Contains(out, `name="_method"`) || !strings.Contains(out, "PUT") {
		t.Fatalf("method missing: %s", out)
	}
	if !strings.Contains(out, "a@b.c") || !strings.Contains(out, "bad") || !strings.Contains(out, "auth") {
		t.Fatalf("old/error/auth missing: %s", out)
	}
	if strings.Contains(out, "guest") {
		t.Fatalf("guest should be hidden: %s", out)
	}
}
