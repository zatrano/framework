package middleware

import (
	"strings"

	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

// Domain allows requests only for the listed hosts (exact or "*.example.com").
func Domain(hosts ...string) routing.MiddlewareFunc {
	allowed := normalizeHosts(hosts)
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			if len(allowed) == 0 {
				return next(req)
			}
			host := requestHost(req)
			if !hostAllowed(host, allowed) {
				return http.Abort(404, "Host not allowed")
			}
			return next(req)
		}
	}
}

func requestHost(req *http.Request) string {
	if req == nil {
		return ""
	}
	host := req.Header("Host")
	if host == "" && req.Raw() != nil {
		host = req.Raw().Host
	}
	host = strings.ToLower(strings.TrimSpace(host))
	if i := strings.Index(host, ":"); i >= 0 {
		host = host[:i]
	}
	return host
}

func normalizeHosts(hosts []string) []string {
	out := make([]string, 0, len(hosts))
	for _, h := range hosts {
		h = strings.ToLower(strings.TrimSpace(h))
		if h == "" {
			continue
		}
		if i := strings.Index(h, ":"); i >= 0 {
			h = h[:i]
		}
		out = append(out, h)
	}
	return out
}

func hostAllowed(host string, allowed []string) bool {
	for _, pattern := range allowed {
		if pattern == "*" || pattern == host {
			return true
		}
		if strings.HasPrefix(pattern, "*.") {
			suffix := strings.TrimPrefix(pattern, "*")
			if strings.HasSuffix(host, suffix) && host != strings.TrimPrefix(pattern, "*.") {
				return true
			}
			// also allow apex? usually *.example.com does not match example.com
		}
	}
	return false
}
