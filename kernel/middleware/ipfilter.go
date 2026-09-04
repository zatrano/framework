package middleware

import (
	"net"
	"strings"

	"github.com/zatrano/framework/kernel/http"
	"github.com/zatrano/framework/kernel/routing"
)

// AllowIP permits only the listed IPs or CIDR ranges (empty = allow all).
func AllowIP(cidrs ...string) routing.MiddlewareFunc {
	nets := parseCIDRs(cidrs)
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			if len(nets) == 0 {
				return next(req)
			}
			if !ipInNets(req.IP(), nets) {
				return http.Abort(403, "IP not allowed")
			}
			return next(req)
		}
	}
}

// DenyIP blocks the listed IPs or CIDR ranges.
func DenyIP(cidrs ...string) routing.MiddlewareFunc {
	nets := parseCIDRs(cidrs)
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			if len(nets) > 0 && ipInNets(req.IP(), nets) {
				return http.Abort(403, "IP denied")
			}
			return next(req)
		}
	}
}

func parseCIDRs(cidrs []string) []*net.IPNet {
	out := make([]*net.IPNet, 0, len(cidrs))
	for _, raw := range cidrs {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if !strings.Contains(raw, "/") {
			if ip := net.ParseIP(raw); ip != nil {
				bits := 32
				if ip.To4() == nil {
					bits = 128
				}
				raw = raw + "/" + itoa(bits)
			}
		}
		_, network, err := net.ParseCIDR(raw)
		if err == nil {
			out = append(out, network)
		}
	}
	return out
}

func ipInNets(ipStr string, nets []*net.IPNet) bool {
	ip := net.ParseIP(strings.TrimSpace(ipStr))
	if ip == nil {
		return false
	}
	for _, n := range nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [4]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
