package docs_test

import (
	stdhttp "net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zatrano/framework/packages/docs"
	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

func TestDocsListAndGet(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "index.md"), []byte("# Hello\n\nWelcome"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "routing.md"), []byte("# Routing\n\nGroups"), 0o644)
	repo := docs.New(dir)
	pages, err := repo.List()
	if err != nil || len(pages) != 2 {
		t.Fatalf("pages=%v err=%v", pages, err)
	}
	html, page, err := repo.HTML("index")
	if err != nil || page.Title != "Hello" || html == "" {
		t.Fatalf("html=%q page=%#v err=%v", html, page, err)
	}
}

func TestDocsNestedSlugAndNavigation(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "digging-deeper")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(dir, "installation.md"), []byte("# Installation\n\nGo get"), 0o644)
	_ = os.WriteFile(filepath.Join(nested, "queues.md"), []byte("# Queues\n\nJobs"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "navigation.json"), []byte(`{
  "sections": [
    {"title": "Getting Started", "pages": [{"title": "Installation", "slug": "installation"}]},
    {"title": "Digging Deeper", "pages": [{"title": "Queues", "slug": "digging-deeper/queues"}]}
  ]
}`), 0o644)

	repo := docs.New(dir)
	pages, err := repo.List()
	if err != nil || len(pages) != 2 {
		t.Fatalf("pages=%v err=%v", pages, err)
	}

	page, err := repo.Get("digging-deeper/queues")
	if err != nil || page.Title != "Queues" {
		t.Fatalf("nested page=%#v err=%v", page, err)
	}

	nav, err := repo.Navigation()
	if err != nil || len(nav) != 2 || nav[1].Pages[0].Slug != "digging-deeper/queues" {
		t.Fatalf("nav=%#v err=%v", nav, err)
	}

	prev, next, err := repo.Neighbors("installation")
	if err != nil || prev != nil || next == nil || next.Slug != "digging-deeper/queues" {
		t.Fatalf("prev=%v next=%v err=%v", prev, next, err)
	}
	prev, next, err = repo.Neighbors("digging-deeper/queues")
	if err != nil || prev == nil || prev.Slug != "installation" || next != nil {
		t.Fatalf("prev=%v next=%v err=%v", prev, next, err)
	}
}

func TestDocsRegisterWithViewRenderer(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "index.md"), []byte("# Docs Home\n\nStart here"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "routing.md"), []byte("# Routing\n\nRoutes"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "navigation.json"), []byte(`{
  "sections": [
    {"title": "Basics", "pages": [
      {"title": "Home", "slug": "index"},
      {"title": "Routing", "slug": "routing"}
    ]}
  ]
}`), 0o644)

	repo := docs.New(dir)
	router := routing.New()
	repo.Register(router, docs.Options{
		Prefix: "/docs",
		ViewRenderer: func(data docs.ViewData) *http.Response {
			return http.HTML("<h1>" + data.Page.Title + "</h1>" + data.HTML)
		},
	})

	raw, _ := stdhttp.NewRequest(stdhttp.MethodGet, "/docs/routing", nil)
	resp := router.Dispatch(http.NewRequest(raw))
	if resp == nil || resp.StatusCode() != 200 {
		t.Fatalf("unexpected response %#v", resp)
	}
	body := string(resp.Content())
	if !strings.Contains(body, "Routing") || !strings.Contains(body, "Routes") {
		t.Fatalf("body=%q", body)
	}
}
