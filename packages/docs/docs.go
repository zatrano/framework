package docs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/markdown"
	"github.com/zatrano/framework/packages/routing"
)

// Page is a documentation markdown page.
type Page struct {
	Slug    string `json:"slug"`
	Title   string `json:"title"`
	Path    string `json:"path"`
	Content string `json:"content,omitempty"`
}

// NavLink is a single documentation link in the sidebar.
type NavLink struct {
	Title string `json:"title"`
	Slug  string `json:"slug"`
}

// NavSection groups documentation links under a heading.
type NavSection struct {
	Title string    `json:"title"`
	Pages []NavLink `json:"pages"`
}

// Neighbor is the previous or next page in navigation order.
type Neighbor struct {
	Title string `json:"title"`
	Slug  string `json:"slug"`
}

// ViewData is passed to a custom documentation view renderer.
type ViewData struct {
	Page  *Page
	HTML  string
	Nav   []NavSection
	Prev  *Neighbor
	Next  *Neighbor
	Pages []Page
	Slug  string
}

// Options configures route registration for documentation.
type Options struct {
	// Prefix is the URL prefix (default "/documentation").
	Prefix string
	// ViewRenderer, when set, replaces the built-in HTML chrome for index/show.
	ViewRenderer func(data ViewData) *http.Response
}

// Repository loads markdown docs from a directory.
type Repository struct {
	root string
}

// New creates a docs repository rooted at dir.
func New(root string) *Repository {
	return &Repository{root: root}
}

// Root returns the documentation root directory.
func (r *Repository) Root() string {
	return r.root
}

// List returns available doc pages (without full content).
func (r *Repository) List() ([]Page, error) {
	pages := make([]Page, 0)
	err := filepath.WalkDir(r.root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".md") {
			return nil
		}
		if strings.HasPrefix(name, "_") {
			return nil
		}
		rel, err := filepath.Rel(r.root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		slug := strings.TrimSuffix(rel, filepath.Ext(rel))
		pages = append(pages, Page{
			Slug:  slug,
			Title: humanize(filepath.Base(slug)),
			Path:  path,
		})
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return []Page{}, nil
		}
		return nil, err
	}
	sort.Slice(pages, func(i, j int) bool { return pages[i].Slug < pages[j].Slug })
	return pages, nil
}

// Navigation loads ordered sidebar sections from navigation.json when present.
// Falls back to a single "Documentation" section from List().
func (r *Repository) Navigation() ([]NavSection, error) {
	path := filepath.Join(r.root, "navigation.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return r.fallbackNavigation()
		}
		return nil, err
	}
	var payload struct {
		Sections []NavSection `json:"sections"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	if len(payload.Sections) == 0 {
		return r.fallbackNavigation()
	}
	for i := range payload.Sections {
		for j := range payload.Sections[i].Pages {
			page := &payload.Sections[i].Pages[j]
			page.Slug = strings.Trim(page.Slug, "/")
			if page.Title == "" {
				page.Title = humanize(filepath.Base(page.Slug))
			}
		}
	}
	return payload.Sections, nil
}

func (r *Repository) fallbackNavigation() ([]NavSection, error) {
	pages, err := r.List()
	if err != nil {
		return nil, err
	}
	links := make([]NavLink, 0, len(pages))
	for _, page := range pages {
		links = append(links, NavLink{Title: page.Title, Slug: page.Slug})
	}
	return []NavSection{{Title: "Documentation", Pages: links}}, nil
}

// FlatNav returns navigation links in sidebar order.
func (r *Repository) FlatNav() ([]NavLink, error) {
	sections, err := r.Navigation()
	if err != nil {
		return nil, err
	}
	links := make([]NavLink, 0)
	for _, section := range sections {
		links = append(links, section.Pages...)
	}
	return links, nil
}

// Neighbors returns the previous and next pages for slug in navigation order.
func (r *Repository) Neighbors(slug string) (prev, next *Neighbor, err error) {
	slug = strings.Trim(slug, "/")
	if slug == "" {
		slug = "index"
	}
	links, err := r.FlatNav()
	if err != nil {
		return nil, nil, err
	}
	for i, link := range links {
		if link.Slug != slug {
			continue
		}
		if i > 0 {
			prev = &Neighbor{Title: links[i-1].Title, Slug: links[i-1].Slug}
		}
		if i+1 < len(links) {
			next = &Neighbor{Title: links[i+1].Title, Slug: links[i+1].Slug}
		}
		return prev, next, nil
	}
	return nil, nil, nil
}

// Get loads a page by slug (supports nested slugs like digging-deeper/queues).
func (r *Repository) Get(slug string) (*Page, error) {
	slug = strings.Trim(slug, "/")
	if slug == "" {
		slug = "index"
	}
	if strings.Contains(slug, "..") {
		return nil, os.ErrNotExist
	}
	path := filepath.Join(r.root, filepath.FromSlash(slug)+".md")
	cleanRoot, err := filepath.Abs(r.root)
	if err != nil {
		return nil, err
	}
	cleanPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(cleanPath, cleanRoot+string(os.PathSeparator)) && cleanPath != cleanRoot {
		return nil, os.ErrNotExist
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	title := humanize(filepath.Base(slug))
	if lines := strings.SplitN(string(raw), "\n", 2); len(lines) > 0 && strings.HasPrefix(strings.TrimSpace(lines[0]), "# ") {
		title = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[0]), "# "))
	}
	return &Page{
		Slug:    slug,
		Title:   title,
		Path:    path,
		Content: string(raw),
	}, nil
}

// HTML returns rendered HTML for a page.
func (r *Repository) HTML(slug string) (string, *Page, error) {
	page, err := r.Get(slug)
	if err != nil {
		return "", nil, err
	}
	return markdown.ToHTML(page.Content), page, nil
}

// ViewPayload builds ViewData for a documentation slug.
func (r *Repository) ViewPayload(slug string) (ViewData, error) {
	slug = strings.Trim(slug, "/")
	if slug == "" {
		slug = "index"
	}
	html, page, err := r.HTML(slug)
	if err != nil {
		return ViewData{}, err
	}
	nav, err := r.Navigation()
	if err != nil {
		return ViewData{}, err
	}
	pages, err := r.List()
	if err != nil {
		return ViewData{}, err
	}
	prev, next, err := r.Neighbors(slug)
	if err != nil {
		return ViewData{}, err
	}
	return ViewData{
		Page:  page,
		HTML:  html,
		Nav:   nav,
		Prev:  prev,
		Next:  next,
		Pages: pages,
		Slug:  slug,
	}, nil
}

// Register mounts documentation routes on the router.
func (r *Repository) Register(router *routing.Router, opts Options) {
	prefix := strings.TrimRight(opts.Prefix, "/")
	if prefix == "" {
		prefix = "/documentation"
	}

	indexHandler := r.IndexHandler()
	showHandler := r.ShowHandler()
	if opts.ViewRenderer != nil {
		renderer := opts.ViewRenderer
		indexHandler = func(req *http.Request) *http.Response {
			if req.WantsJSON() {
				return r.IndexHandler()(req)
			}
			data, err := r.ViewPayload("index")
			if err != nil {
				// Index markdown is optional; still show navigation shell.
				nav, navErr := r.Navigation()
				pages, listErr := r.List()
				if navErr != nil || listErr != nil {
					return http.HTML("<h1>Not Found</h1>").Status(404)
				}
				data = ViewData{Nav: nav, Pages: pages, Slug: "index"}
			}
			return renderer(data)
		}
		showHandler = func(req *http.Request) *http.Response {
			slug := req.Route("slug", "index")
			if req.WantsJSON() {
				return r.ShowHandler()(req)
			}
			data, err := r.ViewPayload(slug)
			if err != nil {
				return http.HTML("<h1>Not Found</h1>").Status(404)
			}
			return renderer(data)
		}
	}

	router.Get(prefix, indexHandler).As("docs.index")
	router.Get(prefix+"/{*slug}", showHandler).As("docs.show")
}

// IndexHandler lists documentation pages.
func (r *Repository) IndexHandler() routing.HandlerFunc {
	return func(req *http.Request) *http.Response {
		pages, err := r.List()
		if err != nil {
			return http.JSON(map[string]any{"message": err.Error()}).Status(500)
		}
		if req.WantsJSON() {
			nav, _ := r.Navigation()
			return http.JSON(map[string]any{"data": pages, "navigation": nav})
		}
		nav, _ := r.Navigation()
		var b strings.Builder
		b.WriteString(`<!doctype html><html><head><meta charset="utf-8"><title>Docs</title>
<style>body{font-family:ui-sans-serif,system-ui;background:#0b1220;color:#e8eef8;padding:2rem;max-width:800px;margin:0 auto}
h1{color:#3dd6c6}a{color:#3dd6c6}li{margin:.4rem 0}h2{margin-top:1.5rem;font-size:1rem;color:#9fb0c8}</style></head><body>`)
		b.WriteString(`<h1>Documentation</h1>`)
		for _, section := range nav {
			b.WriteString(`<h2>` + section.Title + `</h2><ul>`)
			for _, page := range section.Pages {
				b.WriteString(fmt.Sprintf(`<li><a href="/documentation/%s">%s</a></li>`, page.Slug, page.Title))
			}
			b.WriteString(`</ul>`)
		}
		b.WriteString(`</body></html>`)
		return http.HTML(b.String())
	}
}

// ShowHandler renders a markdown doc page.
func (r *Repository) ShowHandler() routing.HandlerFunc {
	return func(req *http.Request) *http.Response {
		slug := req.Route("slug", "index")
		html, page, err := r.HTML(slug)
		if err != nil {
			if req.WantsJSON() {
				return http.JSON(map[string]any{"message": "Document not found"}).Status(404)
			}
			return http.HTML("<h1>Not Found</h1>").Status(404)
		}
		if req.WantsJSON() {
			prev, next, _ := r.Neighbors(slug)
			return http.JSON(map[string]any{
				"slug":    page.Slug,
				"title":   page.Title,
				"content": page.Content,
				"prev":    prev,
				"next":    next,
			})
		}
		prev, next, _ := r.Neighbors(slug)
		var navHTML strings.Builder
		navHTML.WriteString(`<p><a href="/documentation">&larr; Docs</a></p>`)
		if prev != nil || next != nil {
			navHTML.WriteString(`<p>`)
			if prev != nil {
				navHTML.WriteString(fmt.Sprintf(`<a href="/documentation/%s">&larr; %s</a>`, prev.Slug, prev.Title))
			}
			if prev != nil && next != nil {
				navHTML.WriteString(` &middot; `)
			}
			if next != nil {
				navHTML.WriteString(fmt.Sprintf(`<a href="/documentation/%s">%s &rarr;</a>`, next.Slug, next.Title))
			}
			navHTML.WriteString(`</p>`)
		}
		body := fmt.Sprintf(`<!doctype html><html><head><meta charset="utf-8"><title>%s</title>
<style>body{font-family:ui-sans-serif,system-ui;background:#0b1220;color:#e8eef8;padding:2rem;max-width:800px;margin:0 auto;line-height:1.6}
h1,h2,h3{color:#3dd6c6}a{color:#3dd6c6}code{background:#121a2b;padding:.1rem .35rem;border-radius:4px}
pre{background:#121a2b;padding:1rem;border-radius:8px;overflow:auto}</style></head>
<body>%s%s</body></html>`, page.Title, navHTML.String(), html)
		return http.HTML(body)
	}
}

func humanize(slug string) string {
	parts := strings.Split(strings.ReplaceAll(slug, "_", "-"), "-")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}
