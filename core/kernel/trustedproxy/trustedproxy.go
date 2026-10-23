package trustedproxy

import (
	"fmt"
	"math"
	"net"
	"strconv"
	"strings"

	"github.com/zatrano/framework/v3/core/kernel/env"
	"github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/framework/v3/core/kernel/routing"
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
	if production {
		for _, n := range c.nets {
			// 0.0.0.0/0 or ::/0 trusts X-Forwarded-* from every peer, the same as *.
			if ones, _ := n.Mask.Size(); ones == 0 {
				return fmt.Errorf("refusing to boot: TRUSTED_PROXIES=%s trusts every address in production", n.String())
			}
		}
		// Several wide networks can add up to the same thing, for example
		// 0.0.0.0/1,128.0.0.0/1, or ::ffff:0:0/96, which covers every IPv4
		// address. No real proxy fleet needs half of an address family.
		if v4, v6 := coverage(c.nets); v4 >= 0.5 || v6 >= 0.5 {
			return fmt.Errorf("refusing to boot: TRUSTED_PROXIES covers at least half of all IPv4 or IPv6 addresses in production; list only the proxies you run")
		}
	}
	return nil
}

// coverage returns the share of the IPv4 and of the IPv6 address space the
// networks cover, counting overlaps more than once. IPv4-mapped IPv6
// networks count as IPv4, the way net.IPNet.Contains matches them.
func coverage(nets []*net.IPNet) (v4, v6 float64) {
	for _, n := range nets {
		ones, bits := n.Mask.Size()
		if n.IP.To4() != nil {
			if bits == 128 {
				ones -= 96
			}
			if ones < 0 {
				ones = 0
			}
			v4 += math.Ldexp(1, -ones)
			continue
		}
		v6 += math.Ldexp(1, -ones)
	}
	return v4, v6
}

// Middleware trusts X-Forwarded-* headers only when the remote peer is a trusted proxy.
func (c Config) Middleware() routing.MiddlewareFunc {
	trustAll, nets := c.TrustAll, c.nets
	if !trustAll && len(nets) == 0 {
		return func(next routing.HandlerFunc) routing.HandlerFunc { return next }
	}
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			req.Set(clientIPKey, Resolve(req, trustAll, nets))
			if trustAll || ipInNets(RemoteAddr(req), nets) {
				if proto := lastForwardedValue(req, "X-Forwarded-Proto"); proto != "" {
					req.Set(forwardedProtoKey, proto)
				}
				if host := lastForwardedValue(req, "X-Forwarded-Host"); host != "" {
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
// prepending is ignored. Every X-Forwarded-For line counts, in order. A hop
// that is not an IP address stops the walk and the peer address is returned.
// X-Real-IP is read only when no X-Forwarded-For line is present.
func Resolve(req *http.Request, trustAll bool, nets []*net.IPNet) string {
	remote := RemoteAddr(req)
	if !trustAll && !ipInNets(remote, nets) {
		return remote
	}
	lines := req.HeaderValues("X-Forwarded-For")
	if len(lines) == 0 {
		if realIP := parseSingleIP(req.Header("X-Real-IP")); realIP != "" {
			return realIP
		}
		return remote
	}
	hops, ok := parseForwardedIPs(strings.Join(lines, ","))
	if len(hops) == 0 {
		return remote
	}
	if trustAll {
		if !ok {
			return remote
		}
		return hops[0]
	}
	for i := len(hops) - 1; i >= 0; i-- {
		if hops[i] == "" {
			return remote
		}
		if !ipInNets(hops[i], nets) {
			return hops[i]
		}
	}
	return hops[0]
}

// RemoteAddr returns the direct connection IP (ignores forwarding headers).
func RemoteAddr(req *http.Request) string {
	if req == nil {
		return ""
	}
	return req.RemoteIP()
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

// parseForwardedIPs returns one entry per comma-separated hop; a hop that is
// not an IP address is "" and ok is false. Empty list elements are skipped.
func parseForwardedIPs(raw string) (hops []string, ok bool) {
	ok = true
	for _, part := range strings.Split(raw, ",") {
		if strings.TrimSpace(part) == "" {
			continue
		}
		ip := parseSingleIP(part)
		if ip == "" {
			ok = false
		}
		hops = append(hops, ip)
	}
	return hops, ok
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

// lastForwardedValue takes the rightmost hop across every header line, the
// value a proxy that appends would write. A client that prepends is ignored.
func lastForwardedValue(req *http.Request, key string) string {
	vals := req.HeaderValues(key)
	if len(vals) == 0 {
		return ""
	}
	raw := strings.TrimSpace(strings.Join(vals, ","))
	if raw == "" {
		return ""
	}
	if idx := strings.LastIndex(raw, ","); idx >= 0 {
		raw = strings.TrimSpace(raw[idx+1:])
	}
	return raw
}
