package trustedproxy

import (
	"net"
	"strconv"
	"strings"

	"github.com/zatrano/framework/packages/env"
	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

const clientIPKey = "_client_ip"
const forwardedProtoKey = "_forwarded_proto"
const forwardedHostKey = "_forwarded_host"

// Middleware trusts X-Forwarded-* headers only when the remote peer is a trusted proxy.
// Pass "*" to trust all proxies (development only).
func Middleware(proxies ...string) routing.MiddlewareFunc {
	trustAll, nets := parseProxies(proxies)
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			req.Set(clientIPKey, Resolve(req, trustAll, nets))
			if trustAll || ipInNets(RemoteAddr(req), nets) {
				if proto := firstHeaderValue(req.Header("X-Forwarded-Proto")); proto != "" {
					req.Set(forwardedProtoKey, proto)
				}
				if host := firstHeaderValue(req.Header("X-Forwarded-Host")); host != "" {
					req.Set(forwardedHostKey, host)
				}
			}
			return next(req)
		}
	}
}

// FromEnv builds middleware from TRUSTED_PROXIES (comma-separated CIDRs/IPs, or *).
func FromEnv() routing.MiddlewareFunc {
	raw := strings.TrimSpace(env.Get("TRUSTED_PROXIES", ""))
	if raw == "" {
		return Middleware()
	}
	parts := strings.Split(raw, ",")
	cleaned := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			cleaned = append(cleaned, p)
		}
	}
	return Middleware(cleaned...)
}

// Resolve returns the client IP using trusted proxy rules.
func Resolve(req *http.Request, trustAll bool, nets []*net.IPNet) string {
	remote := RemoteAddr(req)
	if !trustAll && !ipInNets(remote, nets) {
		return remote
	}
	if forwarded := strings.TrimSpace(req.Header("X-Forwarded-For")); forwarded != "" {
		parts := strings.Split(forwarded, ",")
		candidate := strings.TrimSpace(parts[0])
		if candidate != "" {
			return candidate
		}
	}
	if realIP := strings.TrimSpace(req.Header("X-Real-IP")); realIP != "" {
		return realIP
	}
	return remote
}

// RemoteAddr returns the direct connection IP (ignores forwarding headers).
func RemoteAddr(req *http.Request) string {
	if req == nil || req.Raw() == nil {
		return ""
	}
	host := req.Raw().RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx != -1 {
		// handle [ipv6]:port
		if strings.HasPrefix(host, "[") {
			if end := strings.Index(host, "]"); end != -1 {
				return host[1:end]
			}
		}
		return host[:idx]
	}
	return host
}

func parseProxies(proxies []string) (trustAll bool, nets []*net.IPNet) {
	for _, p := range proxies {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if p == "*" {
			trustAll = true
			continue
		}
		if !strings.Contains(p, "/") {
			if ip := net.ParseIP(p); ip != nil {
				bits := 32
				if ip.To4() == nil {
					bits = 128
				}
				p = p + "/" + strconv.Itoa(bits)
			}
		}
		_, network, err := net.ParseCIDR(p)
		if err == nil {
			nets = append(nets, network)
		}
	}
	return trustAll, nets
}

func ipInNets(ip string, nets []*net.IPNet) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	for _, n := range nets {
		if n.Contains(parsed) {
			return true
		}
	}
	return false
}

func firstHeaderValue(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if idx := strings.Index(raw, ","); idx >= 0 {
		raw = strings.TrimSpace(raw[:idx])
	}
	return raw
}
