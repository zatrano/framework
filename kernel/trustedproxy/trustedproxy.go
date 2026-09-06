package trustedproxy

import (
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/zatrano/framework/v2/kernel/env"
	"github.com/zatrano/framework/v2/kernel/http"
	"github.com/zatrano/framework/v2/kernel/routing"
)

const clientIPKey = "_client_ip"
const forwardedProtoKey = "_forwarded_proto"
const forwardedHostKey = "_forwarded_host"

// Config is a parsed trusted-proxy list. "*" is a valid parser token
// (TrustAll); production must reject it via Validate, not by dropping
// the token during parse.
type Config struct {
	TrustAll bool
	nets     []*net.IPNet
}

// Parse accepts CIDRs, IPs, or "*". Malformed entries fail-loud.
func Parse(proxies ...string) (Config, error) {
	var cfg Config
	for _, p := range proxies {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if p == "*" {
			cfg.TrustAll = true
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
		if err != nil {
			return Config{}, fmt.Errorf("invalid trusted proxy %q", p)
		}
		cfg.nets = append(cfg.nets, network)
	}
	return cfg, nil
}

// Validate applies production policy. Wildcard trust cannot form a
// safe client-IP provenance model in production.
func (c Config) Validate(production bool) error {
	if production && c.TrustAll {
		return fmt.Errorf("refusing to boot: TRUSTED_PROXIES=* is not allowed in production")
	}
	return nil
}

// Middleware trusts X-Forwarded-* headers only when the remote peer is a trusted proxy.
func (c Config) Middleware() routing.MiddlewareFunc {
	trustAll, nets := c.TrustAll, c.nets
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

// Middleware trusts X-Forwarded-* headers only when the remote peer is a trusted proxy.
// Pass "*" to trust all proxies (development only). Malformed entries panic.
func Middleware(proxies ...string) routing.MiddlewareFunc {
	cfg, err := Parse(proxies...)
	if err != nil {
		panic("trustedproxy: " + err.Error())
	}
	return cfg.Middleware()
}

// ParseEnv reads TRUSTED_PROXIES (comma-separated CIDRs/IPs, or *).
// It does not apply production policy; call Validate for that.
func ParseEnv() (Config, error) {
	raw := strings.TrimSpace(env.Get("TRUSTED_PROXIES", ""))
	if raw == "" {
		return Config{}, nil
	}
	return Parse(strings.Split(raw, ",")...)
}

// FromEnv parses TRUSTED_PROXIES and applies production policy.
// production must come from the application (IsProduction), not a
// second APP_ENV parse, so wildcard cannot re-enter via another env path.
func FromEnv(production bool) (routing.MiddlewareFunc, error) {
	cfg, err := ParseEnv()
	if err != nil {
		return nil, fmt.Errorf("TRUSTED_PROXIES: %w", err)
	}
	if err := cfg.Validate(production); err != nil {
		return nil, err
	}
	return cfg.Middleware(), nil
}

// Resolve returns the client IP using trusted proxy rules.
// Untrusted peers cannot set the client IP via X-Forwarded-For. When the
// peer is a trusted proxy, hops are read from the right (the value the
// proxy appended), skipping trusted proxy addresses. Leftmost attacker
// prepending is ignored.
func Resolve(req *http.Request, trustAll bool, nets []*net.IPNet) string {
	remote := RemoteAddr(req)
	if !trustAll && !ipInNets(remote, nets) {
		return remote
	}
	hops := parseForwardedIPs(req.Header("X-Forwarded-For"))
	if len(hops) == 0 {
		if realIP := parseSingleIP(req.Header("X-Real-IP")); realIP != "" {
			return realIP
		}
		return remote
	}
	if trustAll {
		return hops[0]
	}
	for i := len(hops) - 1; i >= 0; i-- {
		if !ipInNets(hops[i], nets) {
			return hops[i]
		}
	}
	return hops[0]
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

func parseForwardedIPs(raw string) []string {
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if ip := parseSingleIP(part); ip != "" {
			out = append(out, ip)
		}
	}
	return out
}

func parseSingleIP(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.Trim(raw, `"`)
	if raw == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(raw); err == nil {
		raw = host
	}
	raw = strings.Trim(raw, "[]")
	if net.ParseIP(raw) == nil {
		return ""
	}
	return raw
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
