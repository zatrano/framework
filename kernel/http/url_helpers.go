package http

import (
	"fmt"
	"strings"

	"github.com/zatrano/framework/kernel/http/useragent"
)

// Host returns the request host (prefers trusted forwarded host).
func (r *Request) Host() string {
	if v, ok := r.Get("_forwarded_host").(string); ok && v != "" {
		return v
	}
	if r.raw == nil {
		return ""
	}
	return r.raw.Host
}

// Scheme returns "https" or "http".
func (r *Request) Scheme() string {
	if r.Secure() {
		return "https"
	}
	return "http"
}

// Secure reports whether the request is HTTPS (TLS or trusted forwarded proto).
func (r *Request) Secure() bool {
	if v, ok := r.Get("_forwarded_proto").(string); ok {
		return strings.EqualFold(v, "https")
	}
	if r.raw != nil && r.raw.TLS != nil {
		return true
	}
	return false
}

// Root returns scheme://host.
func (r *Request) Root() string {
	host := r.Host()
	if host == "" {
		return r.Scheme() + "://"
	}
	return r.Scheme() + "://" + host
}

// FullURL returns the full request URL including query string.
func (r *Request) FullURL() string {
	if r.raw == nil || r.raw.URL == nil {
		return r.Root()
	}
	uri := r.raw.URL.RequestURI()
	if uri == "" {
		uri = "/"
	}
	return r.Root() + uri
}

// Ajax reports whether the request was made via XMLHttpRequest.
func (r *Request) Ajax() bool {
	return strings.EqualFold(r.Header("X-Requested-With"), "XMLHttpRequest")
}

// Pjax reports whether the request was made via PJAX (X-PJAX).
func (r *Request) Pjax() bool {
	return r.Header("X-PJAX") != ""
}

// PrefersJSON is an alias for WantsJSON.
func (r *Request) PrefersJSON() bool {
	return r.WantsJSON()
}

// Accepts reports whether the Accept header matches any of the given types.
// Types may be short names ("json", "html", "xml", "text") or full MIME types.
func (r *Request) Accepts(types ...string) bool {
	return r.Prefers(types...) != ""
}

// AcceptsJSON reports whether the client accepts JSON.
func (r *Request) AcceptsJSON() bool {
	return r.Accepts("json", "application/json")
}

// AcceptsHtml reports whether the client accepts HTML.
func (r *Request) AcceptsHtml() bool {
	return r.Accepts("html", "text/html")
}

// Prefers returns the first offered type that matches the Accept header, or "".
func (r *Request) Prefers(types ...string) string {
	if len(types) == 0 {
		return ""
	}
	acceptable := r.acceptableTypes()
	if len(acceptable) == 0 {
		return types[0]
	}
	for _, media := range acceptable {
		if media == "*/*" {
			return types[0]
		}
		for _, offered := range types {
			if typeMatchesAccept(offered, media) {
				return offered
			}
		}
	}
	return ""
}

// ExpectsJSON reports whether the client expects a JSON response.
func (r *Request) ExpectsJSON() bool {
	if r.WantsJSON() {
		return true
	}
	return r.Ajax() && !r.Pjax() && r.AcceptsJSON()
}

func (r *Request) acceptableTypes() []string {
	accept := r.Header("Accept")
	if accept == "" {
		return nil
	}
	parts := strings.Split(accept, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		media := strings.TrimSpace(strings.Split(part, ";")[0])
		if media != "" {
			out = append(out, strings.ToLower(media))
		}
	}
	return out
}

func expandAcceptType(t string) []string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "json":
		return []string{"application/json", "text/json"}
	case "html":
		return []string{"text/html", "application/xhtml+xml"}
	case "xml":
		return []string{"application/xml", "text/xml"}
	case "text", "plain":
		return []string{"text/plain"}
	case "any", "*/*":
		return []string{"*/*"}
	default:
		trimmed := strings.ToLower(strings.TrimSpace(t))
		if trimmed == "" {
			return nil
		}
		return []string{trimmed}
	}
}

func typeMatchesAccept(offered, acceptMedia string) bool {
	acceptMedia = strings.ToLower(strings.TrimSpace(acceptMedia))
	if acceptMedia == "" || acceptMedia == "*/*" {
		return true
	}
	for _, candidate := range expandAcceptType(offered) {
		if candidate == "*/*" || candidate == acceptMedia {
			return true
		}
		if strings.HasSuffix(acceptMedia, "/*") {
			prefix := strings.TrimSuffix(acceptMedia, "/*")
			if prefix != "" && strings.HasPrefix(candidate, prefix+"/") {
				return true
			}
		}
		if strings.HasSuffix(candidate, "+json") && (acceptMedia == "application/json" || strings.HasSuffix(acceptMedia, "+json")) {
			return true
		}
	}
	return false
}

// IsMethod reports whether the request method matches any of the given methods.
func (r *Request) IsMethod(methods ...string) bool {
	current := r.Method()
	for _, method := range methods {
		if strings.EqualFold(current, method) {
			return true
		}
	}
	return false
}

// ExactPath reports whether the request path equals path (trailing slashes ignored).
func (r *Request) ExactPath(path string) bool {
	return normalizePath(r.Path()) == normalizePath(path)
}

// PathIs reports whether the request path matches any pattern (* wildcards supported).
func (r *Request) PathIs(patterns ...string) bool {
	path := normalizePath(r.Path())
	for _, pattern := range patterns {
		if matchPattern(path, normalizePath(pattern)) {
			return true
		}
	}
	return false
}

// Segments returns non-empty path segments.
func (r *Request) Segments() []string {
	trimmed := strings.Trim(r.Path(), "/")
	if trimmed == "" {
		return []string{}
	}
	return strings.Split(trimmed, "/")
}

// Segment returns the 1-based path segment.
func (r *Request) Segment(n int, fallback ...string) string {
	segs := r.Segments()
	if n < 1 || n > len(segs) {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return ""
	}
	return segs[n-1]
}

// SetRouteName stores the matched route name on the request.
func (r *Request) SetRouteName(name string) {
	r.Set("_route", name)
}

// RouteName returns the matched route name.
func (r *Request) RouteName() string {
	if v, ok := r.Get("_route").(string); ok {
		return v
	}
	return ""
}

// RouteIs reports whether the matched route name matches any pattern (* wildcards supported).
func (r *Request) RouteIs(patterns ...string) bool {
	name := r.RouteName()
	for _, pattern := range patterns {
		if matchPattern(name, pattern) {
			return true
		}
	}
	return false
}

func normalizePath(path string) string {
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if len(path) > 1 {
		path = strings.TrimRight(path, "/")
	}
	return path
}

func matchPattern(value, pattern string) bool {
	if pattern == "*" || pattern == "/*" {
		return true
	}
	if !strings.Contains(pattern, "*") {
		return value == pattern
	}
	parts := strings.Split(pattern, "*")
	if !strings.HasPrefix(value, parts[0]) {
		return false
	}
	rest := value[len(parts[0]):]
	for i := 1; i < len(parts); i++ {
		part := parts[i]
		if part == "" {
			if i == len(parts)-1 {
				return true
			}
			continue
		}
		idx := strings.Index(rest, part)
		if idx < 0 {
			return false
		}
		rest = rest[idx+len(part):]
	}
	return rest == "" || strings.HasSuffix(pattern, "*")
}

// QueryAll returns all query parameters (multi-value).
func (r *Request) QueryAll() map[string][]string {
	if r == nil || r.raw == nil || r.raw.URL == nil {
		return map[string][]string{}
	}
	values := r.raw.URL.Query()
	out := make(map[string][]string, len(values))
	for key, items := range values {
		copied := make([]string, len(items))
		copy(copied, items)
		out[key] = copied
	}
	return out
}

// Queries returns the first value for each query parameter.
func (r *Request) Queries() map[string]string {
	all := r.QueryAll()
	out := make(map[string]string, len(all))
	for key, items := range all {
		if len(items) > 0 {
			out[key] = items[0]
		}
	}
	return out
}

// UserAgent returns the raw User-Agent header.
func (r *Request) UserAgent() string {
	return r.Header("User-Agent")
}

// Agent returns a parsed User-Agent summary.
func (r *Request) Agent() useragent.Agent {
	return useragent.Parse(r.UserAgent())
}

// Old returns a previously flashed input value (same key as flash.OldValue).
func (r *Request) Old(key string, fallback ...string) string {
	if r == nil {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return ""
	}
	sess := r.Session()
	if sess == nil {
		if len(fallback) > 0 {
			return fallback[0]
		}
		return ""
	}
	raw := sess.Get("_old_input")
	switch v := raw.(type) {
	case map[string]string:
		if value, ok := v[key]; ok {
			return value
		}
	case map[string]any:
		if value, ok := v[key]; ok {
			return fmt.Sprint(value)
		}
	}
	if len(fallback) > 0 {
		return fallback[0]
	}
	return ""
}
