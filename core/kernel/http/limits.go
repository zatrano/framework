package http

import (
	"mime"
	"strconv"
	"strings"

	"github.com/zatrano/framework/v3/core/kernel/env"
	"github.com/zatrano/rawhttp"
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

// MaxRequestBodySize is MaxRequestBytes clamped to int for rawhttp.Server.
func MaxRequestBodySize() int {
	n := MaxRequestBytes()
	maxInt := int64(^uint(0) >> 1)
	if n > maxInt {
		return int(maxInt)
	}
	if n <= 0 {
		return int(DefaultMaxBodyBytes)
	}
	return int(n)
}

// HeaderBodyLimit is the header-time cap for a Content-Type.
// JSON (application/json and any +json suffix), form urlencoded, and text/*
// are cut at MaxBodyBytes. Multipart and every other type, including a missing
// Content-Type, stay at the server ceiling (MaxRequestBytes). Request.Body
// and Request.JSON still apply MaxBodyBytes when the handler reads them.
func HeaderBodyLimit(contentType string) int64 {
	if earlyBodyMedia(contentType) {
		return MaxBodyBytes()
	}
	return MaxRequestBytes()
}

// DefaultMaxInflightBodyBytes is the in-flight reservation ceiling.
const DefaultMaxInflightBodyBytes = 256 << 20 // 256 MiB

// MaxInflightBodyBytes reads HTTP_MAX_INFLIGHT_BODY_BYTES.
// Unset means 256 MiB. A negative value disables the budget. Zero is a
// budget of zero. An invalid value keeps the default.
func MaxInflightBodyBytes() int64 {
	raw := strings.TrimSpace(env.Get("HTTP_MAX_INFLIGHT_BODY_BYTES", ""))
	if raw == "" {
		return DefaultMaxInflightBodyBytes
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return DefaultMaxInflightBodyBytes
	}
	return n
}

// BodyLimitConfig builds a HeaderReceived override. Non-positive n keeps the
// server ceiling (rawhttp treats 0 as "no override"). A positive n replaces
// the server ceiling for that request, including values above MaxRequestBytes.
func BodyLimitConfig(n int64) rawhttp.RequestConfig {
	if n <= 0 {
		return rawhttp.RequestConfig{}
	}
	maxInt := int64(^uint(0) >> 1)
	if n > maxInt {
		n = maxInt
	}
	return rawhttp.RequestConfig{MaxRequestBodySize: int(n)}
}

func earlyBodyMedia(contentType string) bool {
	media := mediaType(contentType)
	if media == "" {
		return false
	}
	if media == "application/json" || media == "application/x-www-form-urlencoded" {
		return true
	}
	if strings.HasSuffix(media, "+json") {
		return true
	}
	return strings.HasPrefix(media, "text/")
}

func mediaType(contentType string) string {
	ct := strings.TrimSpace(contentType)
	if ct == "" {
		return ""
	}
	media, _, err := mime.ParseMediaType(ct)
	if err != nil {
		media = ct
		if i := strings.IndexByte(media, ';'); i >= 0 {
			media = media[:i]
		}
		media = strings.TrimSpace(media)
	}
	return strings.ToLower(media)
}
