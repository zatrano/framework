package i18n

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFlattenAndT(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("en.json", `{"app":{"title":"Hello"},"single":"x"}`)
	write("tr.json", `{"app":{"title":"Merhaba"}}`)

	b, err := LoadDir(dir, "en", []string{"en", "tr"})
	if err != nil {
		t.Fatal(err)
	}
	if got := b.T("en", "app.title"); got != "Hello" {
		t.Fatalf("en app.title: %q", got)
	}
	if got := b.T("tr", "app.title"); got != "Merhaba" {
		t.Fatalf("tr app.title: %q", got)
	}
	if got := b.T("tr", "missing.key"); got != "missing.key" {
		t.Fatalf("fallback key: %q", got)
	}
	// tr missing key falls back to en
	if got := b.T("tr", "single"); got != "x" {
		t.Fatalf("fallback to default locale: %q", got)
	}
}

func TestFormatMapAndStruct(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("en.json", `{"greet":"Hello, {{.Name}}!","nested":"{{.A}} {{.B}}"}`)
	write("tr.json", `{"greet":"Merhaba, {{.Name}}!"}`)

	b, err := LoadDir(dir, "en", []string{"en", "tr"})
	if err != nil {
		t.Fatal(err)
	}
	s, err := b.Format("en", "greet", map[string]any{"Name": "Ada"})
	if err != nil {
		t.Fatal(err)
	}
	if s != "Hello, Ada!" {
		t.Fatalf("map: %q", s)
	}
	s, err = b.Format("tr", "greet", map[string]string{"Name": "Ada"})
	if err != nil || s != "Merhaba, Ada!" {
		t.Fatalf("map string: %q err %v", s, err)
	}
	type data struct {
		Name string
	}
	s, err = b.Format("en", "greet", data{Name: "Bob"})
	if err != nil || s != "Hello, Bob!" {
		t.Fatalf("struct: %q err %v", s, err)
	}
	s, err = b.Format("en", "nested", map[string]any{"A": "x", "B": "y"})
	if err != nil || s != "x y" {
		t.Fatalf("multi: %q err %v", s, err)
	}
}

func TestPickLocale(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "en.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tr.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := LoadDir(dir, "en", []string{"en", "tr"})
	if err != nil {
		t.Fatal(err)
	}
	if got := b.PickLocale("tr-TR,en;q=0.5", "", ""); got != "tr" {
		t.Fatalf("Accept-Language: %q", got)
	}
	if got := b.PickLocale("", "tr", ""); got != "tr" {
		t.Fatalf("query: %q", got)
	}
	if got := b.PickLocale("", "", "en"); got != "en" {
		t.Fatalf("cookie: %q", got)
	}
}

