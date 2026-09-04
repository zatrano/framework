package middleware

import (
	"crypto/sha1"
	"encoding/hex"
	stdhttp "net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zatrano/framework/kernel/http"
	"github.com/zatrano/framework/kernel/routing"
)

// CacheControl sets Cache-Control (and optional Expires) on successful responses.
func CacheControl(directive string, maxAge ...time.Duration) routing.MiddlewareFunc {
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			resp := next(req)
			if resp == nil {
				return resp
			}
			if resp.StatusCode() >= 200 && resp.StatusCode() < 300 {
				value := directive
				if len(maxAge) > 0 && maxAge[0] > 0 {
					secs := int(maxAge[0].Seconds())
					if value == "" {
						value = "public, max-age=" + strconv.Itoa(secs)
					} else if !strings.Contains(strings.ToLower(value), "max-age=") {
						value = value + ", max-age=" + strconv.Itoa(secs)
					}
					resp.Header("Expires", time.Now().Add(maxAge[0]).UTC().Format(time.RFC1123))
				}
				if value != "" {
					resp.Header("Cache-Control", value)
				}
			}
			return resp
		}
	}
}

// ETag adds a weak ETag for response bodies and returns 304 when If-None-Match matches.
func ETag(next routing.HandlerFunc) routing.HandlerFunc {
	return func(req *http.Request) *http.Response {
		resp := next(req)
		if resp == nil || resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
			return resp
		}
		body := resp.Content()
		if len(body) == 0 {
			return resp
		}
		sum := sha1.Sum(body)
		tag := `W/"` + hex.EncodeToString(sum[:]) + `"`
		resp.Header("ETag", tag)

		if match := req.Header("If-None-Match"); match != "" && etagMatch(match, tag) {
			out := http.NoContent().Status(304)
			out.Header("ETag", tag)
			return out
		}
		return resp
	}
}

func etagMatch(header, tag string) bool {
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		if part == "*" || part == tag {
			return true
		}
	}
	return false
}

// LastModified sets Last-Modified and returns 304 when If-Modified-Since is fresh.
func LastModified(modTime func(req *http.Request) time.Time) routing.MiddlewareFunc {
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			var mod time.Time
			if modTime != nil {
				mod = modTime(req)
			}
			if mod.IsZero() {
				return next(req)
			}
			mod = mod.UTC().Truncate(time.Second)
			headerValue := mod.Format(stdhttp.TimeFormat)

			if since := req.Header("If-Modified-Since"); since != "" {
				if t, err := stdhttp.ParseTime(since); err == nil {
					if !mod.After(t.UTC().Truncate(time.Second)) {
						out := http.NoContent().Status(304)
						out.Header("Last-Modified", headerValue)
						return out
					}
				}
			}

			resp := next(req)
			if resp == nil {
				return resp
			}
			if resp.StatusCode() >= 200 && resp.StatusCode() < 300 {
				resp.Header("Last-Modified", headerValue)
			}
			return resp
		}
	}
}
