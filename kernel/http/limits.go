package http

import (
	"strings"

	"github.com/zatrano/framework/kernel/env"
)

// DefaultMaxBodyBytes is the JSON/raw body cap unless MAX_BODY_BYTES is set.
const DefaultMaxBodyBytes = 2 << 20 // 2 MiB

// MaxBodyBytes returns the JSON/raw request body limit.
func MaxBodyBytes() int64 {
	if raw := strings.TrimSpace(env.Get("MAX_BODY_BYTES", "")); raw != "" {
		if n, err := strconvAtoiSafe(raw); err == nil && n > 0 {
			return int64(n)
		}
	}
	return DefaultMaxBodyBytes
}

// MaxRequestBytes is the server-level absolute body ceiling (the larger of
// JSON and multipart upload limits). JSON() / Body() still apply MaxBodyBytes.
func MaxRequestBytes() int64 {
	body := MaxBodyBytes()
	upload := int64(maxUploadBytes())
	if upload > body {
		return upload
	}
	return body
}
