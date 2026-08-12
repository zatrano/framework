package pages

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
	"github.com/zatrano/framework/packages/view"
)

// Registrar mounts file-based pages onto a router.
type Registrar struct {
	root   string
	engine *view.Engine
	prefix string
}

// New creates a page registrar for views under root (e.g. views/pages).
func New(root string, engine *view.Engine) *Registrar {
	return &Registrar{root: root, engine: engine, prefix: ""}
}

// Prefix sets a URL prefix (e.g. "/pages").
func (r *Registrar) Prefix(prefix string) *Registrar {
	r.prefix = strings.TrimRight(prefix, "/")
	return r
}

// Register scans page templates and registers GET routes.
// Mapping:
//
//	index.html      -> /
//	about.html      -> /about
//	docs/index.html -> /docs
//	docs/install.html -> /docs/install
//	users/[id].html -> /users/{id}
func (r *Registrar) Register(router *routing.Router) error {
	if r.engine == nil {
		return nil
	}
	entries := make([]pageEntry, 0)
	err := filepath.WalkDir(r.root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".html" && ext != ".zblade" {
			return nil
		}
		rel, err := filepath.Rel(r.root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		routePath, viewName := mapPage(rel)
		entries = append(entries, pageEntry{route: routePath, view: viewName})
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		routePath := joinURL(r.prefix, entry.route)
		viewName := entry.view
		name := "pages" + strings.ReplaceAll(routePath, "/", ".")
		if routePath == "/" || routePath == "" {
			name = "pages.home"
		}
		router.Get(routePath, func(req *http.Request) *http.Response {
			data := map[string]any{
				"title":  strings.Trim(routePath, "/"),
				"params": map[string]string{},
			}
			if data["title"] == "" {
				data["title"] = "Home"
			}
			rawRouteParams(req, data)
			return http.View(viewName, data)
		}).As(name)
	}
	return nil
}

type pageEntry struct {
	route string
	view  string
}

func mapPage(rel string) (routePath, viewName string) {
	ext := filepath.Ext(rel)
	rel = strings.TrimSuffix(rel, ext)
	parts := strings.Split(rel, "/")
	routeParts := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "index" {
			continue
		}
		if strings.HasPrefix(part, "[") && strings.HasSuffix(part, "]") {
			param := strings.TrimSuffix(strings.TrimPrefix(part, "["), "]")
			routeParts = append(routeParts, "{"+param+"}")
			continue
		}
		routeParts = append(routeParts, part)
	}
	if len(routeParts) == 0 {
		routePath = "/"
	} else {
		routePath = "/" + strings.Join(routeParts, "/")
	}
	viewName = "pages." + strings.ReplaceAll(rel, "/", ".")
	return routePath, viewName
}

func joinURL(prefix, path string) string {
	if prefix == "" {
		return path
	}
	if path == "/" {
		return prefix
	}
	return prefix + path
}

func rawRouteParams(req *http.Request, data map[string]any) {
	params := map[string]string{}
	// Best-effort: common dynamic segments used by demos.
	for _, key := range []string{"id", "slug", "user"} {
		if v := req.Route(key); v != "" {
			params[key] = v
		}
	}
	data["params"] = params
	for k, v := range params {
		data[k] = v
	}
}
