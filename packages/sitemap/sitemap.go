package sitemap

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

// URL is a sitemap entry.
type URL struct {
	Loc        string
	LastMod    time.Time
	ChangeFreq string
	Priority   float64
}

// Builder builds sitemap.xml and robots.txt.
type Builder struct {
	mu      sync.Mutex
	baseURL string
	urls    []URL
}

// New creates a sitemap builder.
func New(baseURL string) *Builder {
	return &Builder{
		baseURL: strings.TrimRight(baseURL, "/"),
		urls:    make([]URL, 0),
	}
}

// Add appends a URL entry (path or absolute).
func (b *Builder) Add(loc string, opts ...URL) *Builder {
	b.mu.Lock()
	defer b.mu.Unlock()
	entry := URL{Loc: b.resolve(loc), Priority: 0.5, ChangeFreq: "weekly"}
	if len(opts) > 0 {
		o := opts[0]
		if o.LastMod.IsZero() {
			o.LastMod = time.Now().UTC()
		}
		if o.ChangeFreq == "" {
			o.ChangeFreq = "weekly"
		}
		if o.Priority == 0 {
			o.Priority = 0.5
		}
		o.Loc = b.resolve(loc)
		entry = o
	} else {
		entry.LastMod = time.Now().UTC()
	}
	b.urls = append(b.urls, entry)
	return b
}

// XML renders sitemap.xml.
func (b *Builder) XML() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sb.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	for _, u := range b.urls {
		sb.WriteString("  <url>\n")
		sb.WriteString("    <loc>" + xmlEscape(u.Loc) + "</loc>\n")
		if !u.LastMod.IsZero() {
			sb.WriteString("    <lastmod>" + u.LastMod.Format("2006-01-02") + "</lastmod>\n")
		}
		if u.ChangeFreq != "" {
			sb.WriteString("    <changefreq>" + xmlEscape(u.ChangeFreq) + "</changefreq>\n")
		}
		sb.WriteString(fmt.Sprintf("    <priority>%.1f</priority>\n", u.Priority))
		sb.WriteString("  </url>\n")
	}
	sb.WriteString("</urlset>\n")
	return sb.String()
}

// Robots renders robots.txt.
func (b *Builder) Robots(extra ...string) string {
	var sb strings.Builder
	sb.WriteString("User-agent: *\n")
	sb.WriteString("Allow: /\n")
	for _, line := range extra {
		sb.WriteString(line)
		if !strings.HasSuffix(line, "\n") {
			sb.WriteString("\n")
		}
	}
	sb.WriteString("Sitemap: " + b.baseURL + "/sitemap.xml\n")
	return sb.String()
}

// SitemapHandler serves sitemap.xml.
func (b *Builder) SitemapHandler() routing.HandlerFunc {
	return func(req *http.Request) *http.Response {
		resp := http.Text(b.XML())
		resp.Header("Content-Type", "application/xml; charset=utf-8")
		return resp
	}
}

// RobotsHandler serves robots.txt.
func (b *Builder) RobotsHandler() routing.HandlerFunc {
	return func(req *http.Request) *http.Response {
		return http.Text(b.Robots())
	}
}

func (b *Builder) resolve(loc string) string {
	if strings.HasPrefix(loc, "http://") || strings.HasPrefix(loc, "https://") {
		return loc
	}
	if !strings.HasPrefix(loc, "/") {
		loc = "/" + loc
	}
	return b.baseURL + loc
}

func xmlEscape(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	)
	return r.Replace(s)
}
