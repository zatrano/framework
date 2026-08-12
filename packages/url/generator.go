package url

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/zatrano/framework/packages/routing"
)

// Generator builds application URLs.
type Generator struct {
	router     *routing.Router
	root       string
	assetRoot  string
	signingKey string
}

// New creates a URL generator.
func New(router *routing.Router, root string) *Generator {
	root = strings.TrimRight(root, "/")
	return &Generator{
		router:    router,
		root:      root,
		assetRoot: root,
	}
}

// SetAssetRoot sets the asset base URL.
func (g *Generator) SetAssetRoot(root string) {
	g.assetRoot = strings.TrimRight(root, "/")
}

// To builds an absolute URL for a path.
func (g *Generator) To(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return g.root + path
}

// Route builds a named route URL.
func (g *Generator) Route(name string, params ...map[string]string) (string, error) {
	path, err := g.router.URL(name, params...)
	if err != nil {
		return "", err
	}
	return g.To(path), nil
}

// MustRoute builds a named route URL or panics.
func (g *Generator) MustRoute(name string, params ...map[string]string) string {
	u, err := g.Route(name, params...)
	if err != nil {
		panic(err)
	}
	return u
}

// Asset builds an asset URL.
func (g *Generator) Asset(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	path = strings.TrimPrefix(path, "/")
	return g.assetRoot + "/" + path
}

// Query appends query parameters to a URL.
func (g *Generator) Query(path string, query map[string]string) string {
	base := g.To(path)
	values := url.Values{}
	for key, value := range query {
		values.Set(key, value)
	}
	encoded := values.Encode()
	if encoded == "" {
		return base
	}
	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}
	return base + sep + encoded
}

// Current builds a URL from request path and optional query overrides.
func (g *Generator) Current(path string, query ...map[string]string) string {
	if len(query) == 0 {
		return g.To(path)
	}
	return g.Query(path, query[0])
}

// HasRoute reports whether a named route exists.
func (g *Generator) HasRoute(name string) bool {
	_, err := g.router.URL(name)
	return err == nil
}

// Format formats a path with sprintf-style args then absolutizes it.
func (g *Generator) Format(format string, args ...any) string {
	return g.To(fmt.Sprintf(format, args...))
}
