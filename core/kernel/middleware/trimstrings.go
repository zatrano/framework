package middleware

import (
	"strings"

	"github.com/zatrano/framework/v3/core/kernel/http"
	"github.com/zatrano/framework/v3/core/kernel/routing"
)

// TrimStrings queues whitespace trimming for request inputs, skipping excepted keys.
// The transform runs on the first Input, Query, All, Only, Except, Merge, Replace,
// or Forget. A request with no query and no body does no parsing. Body and JSON
// stay raw. Query keeps the raw query value; form and JSON input maps are trimmed.
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

// ConvertEmptyStringsToNull queues removal of empty-string inputs so they behave as missing/null.
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
