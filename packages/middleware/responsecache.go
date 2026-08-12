package middleware

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/zatrano/framework/packages/cache"
	"github.com/zatrano/framework/packages/http"
	"github.com/zatrano/framework/packages/routing"
)

type cachedHTTPResponse struct {
	Status      int               `json:"status"`
	ContentType string            `json:"content_type"`
	Body        string            `json:"body"`
	Headers     map[string]string `json:"headers,omitempty"`
}

// ResponseCache caches successful GET responses in the given cache store.
func ResponseCache(store cache.Store, ttl time.Duration, keyFn ...func(*http.Request) string) routing.MiddlewareFunc {
	buildKey := func(req *http.Request) string {
		if len(keyFn) > 0 && keyFn[0] != nil {
			return keyFn[0](req)
		}
		return DefaultResponseCacheKey(req)
	}
	return func(next routing.HandlerFunc) routing.HandlerFunc {
		return func(req *http.Request) *http.Response {
			if store == nil || !strings.EqualFold(req.Method(), "GET") {
				return next(req)
			}
			key := buildKey(req)
			if raw, ok := store.Get(key); ok {
				if cached, err := decodeCachedResponse(raw); err == nil {
					ct := cached.ContentType
					if ct == "" {
						ct = "application/json"
					}
					resp := http.Text("").Status(cached.Status).SetContent([]byte(cached.Body), ct)
					for k, v := range cached.Headers {
						resp.Header(k, v)
					}
					resp.Header("X-Response-Cache", "HIT")
					return resp
				}
			}

			resp := next(req)
			if resp == nil || resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
				return resp
			}
			if len(resp.Cookies()) > 0 {
				return resp
			}
			payload := cachedHTTPResponse{
				Status:      resp.StatusCode(),
				ContentType: resp.ContentType(),
				Body:        string(resp.Content()),
				Headers:     map[string]string{},
			}
			for k, values := range resp.Headers() {
				if len(values) > 0 {
					payload.Headers[k] = values[0]
				}
			}
			_ = store.Put(key, payload, ttl)
			resp.Header("X-Response-Cache", "MISS")
			return resp
		}
	}
}

// DefaultResponseCacheKey builds a stable cache key for the request.
func DefaultResponseCacheKey(req *http.Request) string {
	raw := strings.ToUpper(req.Method()) + " " + req.Path()
	if req.Raw() != nil && req.Raw().URL != nil && req.Raw().URL.RawQuery != "" {
		raw += "?" + req.Raw().URL.RawQuery
	}
	sum := sha1.Sum([]byte(raw))
	return "response:" + hex.EncodeToString(sum[:])
}

func decodeCachedResponse(raw any) (cachedHTTPResponse, error) {
	switch v := raw.(type) {
	case cachedHTTPResponse:
		return v, nil
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return cachedHTTPResponse{}, err
		}
		var out cachedHTTPResponse
		err = json.Unmarshal(b, &out)
		return out, err
	}
}
