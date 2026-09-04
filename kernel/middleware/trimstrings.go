package middleware

import (
	"strings"

	"github.com/zatrano/framework/kernel/http"
	"github.com/zatrano/framework/kernel/routing"
)

// TrimStrings trims whitespace from request inputs, skipping excepted keys.
func TrimStrings(except ...string) routing.MiddlewareFunc {
	skip := exceptSet(except...)
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			req.TransformInputs(func(key, value string) (string, bool) {
				if skip[key] {
					return value, true
				}
				return strings.TrimSpace(value), true
			})
			return next(req)
		}
	}
}

// ConvertEmptyStringsToNull removes empty-string inputs so they behave as missing/null.
func ConvertEmptyStringsToNull(except ...string) routing.MiddlewareFunc {
	skip := exceptSet(except...)
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			req.TransformInputs(func(key, value string) (string, bool) {
				if skip[key] {
					return value, true
				}
				if value == "" {
					return "", false
				}
				return value, true
			})
			return next(req)
		}
	}
}

func exceptSet(keys ...string) map[string]bool {
	out := make(map[string]bool, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key != "" {
			out[key] = true
		}
	}
	return out
}
