package fingerprint

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/zatrano/framework/packages/http"
)

// Hash builds a stable fingerprint from parts.
func Hash(parts ...string) string {
	var b strings.Builder
	for i, part := range parts {
		if i > 0 {
			b.WriteByte('|')
		}
		b.WriteString(strings.TrimSpace(part))
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

// FromRequest derives a device fingerprint from request signals.
func FromRequest(req *http.Request, extras ...string) string {
	if req == nil {
		return Hash(extras...)
	}
	parts := []string{
		req.IP(),
		req.Header("User-Agent"),
		req.Header("Accept-Language"),
		req.Header("Sec-CH-UA"),
		req.Header("Sec-CH-UA-Platform"),
	}
	parts = append(parts, extras...)
	return Hash(parts...)
}

// Short returns the first n hex chars of a fingerprint (default 16).
func Short(fp string, n ...int) string {
	limit := 16
	if len(n) > 0 && n[0] > 0 {
		limit = n[0]
	}
	if len(fp) <= limit {
		return fp
	}
	return fp[:limit]
}
